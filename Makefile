.PHONY: build clean install install-config go-install test fmt lint migrate setup systemd-install systemd-uninstall

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/msmhq/msm/cmd/msm/cmd.Version=$(VERSION)"
PREFIX := /usr/local
SYSCONFDIR := /etc
MSM_USER := minecraft
MSM_HOME := /opt/msm

build:
	go build $(LDFLAGS) -o bin/msm ./cmd/msm

go-install:
	go install $(LDFLAGS) ./cmd/msm

install: build
	@echo "Installing msm to $(PREFIX)/bin..."
	install -d $(PREFIX)/bin
	install -m 755 bin/msm $(PREFIX)/bin/msm
	@echo "Installing default config to $(SYSCONFDIR)/msm.conf..."
	@if [ ! -f $(SYSCONFDIR)/msm.conf ]; then \
		install -m 644 msm.conf $(SYSCONFDIR)/msm.conf; \
		echo "Created $(SYSCONFDIR)/msm.conf"; \
	else \
		echo "$(SYSCONFDIR)/msm.conf already exists, skipping"; \
	fi
	@echo "Installing cron job..."
	$(PREFIX)/bin/msm cron install
	@echo "Installation complete!"

uninstall:
	rm -f $(PREFIX)/bin/msm
	rm -f /etc/cron.d/msm
	@echo "Uninstalled msm (config file $(SYSCONFDIR)/msm.conf preserved)"

install-config:
	@echo "Installing default config to $(SYSCONFDIR)/msm.conf..."
	@if [ ! -f $(SYSCONFDIR)/msm.conf ]; then \
		install -m 644 msm.conf $(SYSCONFDIR)/msm.conf; \
		echo "Created $(SYSCONFDIR)/msm.conf"; \
	else \
		echo "$(SYSCONFDIR)/msm.conf already exists, skipping"; \
	fi

migrate:
	@echo "Checking for old bash MSM installation..."
	@if [ -f /etc/init.d/msm ]; then \
		echo "Removing /etc/init.d/msm (old bash script)"; \
		rm -f /etc/init.d/msm; \
	fi
	@for link in /etc/rc*.d/*msm; do \
		if [ -L "$$link" ]; then \
			echo "Removing init.d symlink: $$link"; \
			rm -f "$$link"; \
		fi; \
	done 2>/dev/null || true
	@if [ -f /etc/cron.d/msm ] && grep -q '/etc/init.d/msm' /etc/cron.d/msm 2>/dev/null; then \
		echo "Old cron file detected, will be replaced by 'msm cron install'"; \
	fi
	@echo "Migration cleanup complete. Run 'sudo make install' to complete installation."

setup:
	@echo "Setting up MSM system user and directories..."
	@# Ensure group exists
	@if ! getent group $(MSM_USER) >/dev/null 2>&1; then \
		echo "Creating group '$(MSM_USER)'..."; \
		groupadd $(MSM_USER); \
	else \
		echo "Group '$(MSM_USER)' already exists"; \
	fi
	@# Ensure user exists
	@if ! id $(MSM_USER) >/dev/null 2>&1; then \
		echo "Creating system user '$(MSM_USER)'..."; \
		useradd --system --home-dir $(MSM_HOME) --shell /bin/bash --gid $(MSM_USER) $(MSM_USER) || \
		adduser --system --home $(MSM_HOME) --shell /bin/bash --ingroup $(MSM_USER) $(MSM_USER); \
	else \
		echo "User '$(MSM_USER)' already exists"; \
	fi
	@# Ensure user is in the minecraft group (handles pre-existing users)
	@if ! id -nG $(MSM_USER) 2>/dev/null | grep -qw $(MSM_USER); then \
		echo "Adding user '$(MSM_USER)' to group '$(MSM_USER)'..."; \
		usermod -aG $(MSM_USER) $(MSM_USER); \
	fi
	@# Ensure user's primary group is minecraft
	@if [ "$$(id -gn $(MSM_USER) 2>/dev/null)" != "$(MSM_USER)" ]; then \
		echo "Setting primary group for '$(MSM_USER)' to '$(MSM_USER)'..."; \
		usermod -g $(MSM_USER) $(MSM_USER); \
	fi
	@# Ensure home directory is correct
	@CURRENT_HOME=$$(getent passwd $(MSM_USER) | cut -d: -f6); \
	if [ "$$CURRENT_HOME" != "$(MSM_HOME)" ]; then \
		echo "Updating home directory for '$(MSM_USER)' from $$CURRENT_HOME to $(MSM_HOME)..."; \
		usermod -d $(MSM_HOME) $(MSM_USER); \
	fi
	@# Create directory structure (mkdir -p is idempotent)
	@echo "Ensuring directory structure at $(MSM_HOME)..."
	@mkdir -p $(MSM_HOME)/servers
	@mkdir -p $(MSM_HOME)/jars
	@mkdir -p $(MSM_HOME)/versioning
	@mkdir -p $(MSM_HOME)/archives/worlds
	@mkdir -p $(MSM_HOME)/archives/logs
	@mkdir -p $(MSM_HOME)/archives/backups
	@mkdir -p $(MSM_HOME)/fabric
	@# Fix ownership only on directories that need it (faster for large installs)
	@echo "Ensuring ownership and permissions..."
	@for dir in $(MSM_HOME) $(MSM_HOME)/servers $(MSM_HOME)/jars $(MSM_HOME)/versioning \
		$(MSM_HOME)/archives $(MSM_HOME)/archives/worlds $(MSM_HOME)/archives/logs \
		$(MSM_HOME)/archives/backups $(MSM_HOME)/fabric; do \
		if [ -d "$$dir" ]; then \
			OWNER=$$(stat -c '%U:%G' "$$dir" 2>/dev/null || stat -f '%Su:%Sg' "$$dir" 2>/dev/null); \
			if [ "$$OWNER" != "$(MSM_USER):$(MSM_USER)" ]; then \
				echo "  Fixing ownership on $$dir"; \
				chown $(MSM_USER):$(MSM_USER) "$$dir"; \
			fi; \
		fi; \
	done
	@# Ensure setgid on servers directory
	@PERMS=$$(stat -c '%a' $(MSM_HOME)/servers 2>/dev/null || stat -f '%OLp' $(MSM_HOME)/servers 2>/dev/null); \
	if [ "$$PERMS" != "2775" ]; then \
		echo "  Setting permissions on $(MSM_HOME)/servers"; \
		chmod 2775 $(MSM_HOME)/servers; \
	fi
	@echo "System setup complete!"

systemd-install:
	@echo "Installing systemd services..."
	@# Always update service files to match repo version
	install -m 644 init/msm.service /etc/systemd/system/
	install -m 644 init/msm@.service /etc/systemd/system/
	systemctl daemon-reload
	@# Enable is idempotent
	@if ! systemctl is-enabled msm >/dev/null 2>&1; then \
		echo "Enabling msm.service..."; \
		systemctl enable msm; \
	else \
		echo "msm.service already enabled"; \
	fi
	@echo "Systemd services ready:"
	@echo "  - msm.service: starts all servers on boot"
	@echo "  - msm@.service: per-server control (e.g., systemctl start msm@survival)"

systemd-uninstall:
	@echo "Removing systemd services..."
	-systemctl disable msm 2>/dev/null
	rm -f /etc/systemd/system/msm.service
	rm -f /etc/systemd/system/msm@.service
	systemctl daemon-reload
	@echo "Systemd services removed"

clean:
	rm -rf bin/

test:
	go test -v ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/msm-linux-amd64 ./cmd/msm
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/msm-linux-arm64 ./cmd/msm

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/msm-darwin-amd64 ./cmd/msm
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/msm-darwin-arm64 ./cmd/msm

build-all: build build-linux build-darwin

release: clean build-all
	@echo "Built binaries in bin/"
