.PHONY: build clean install go-install test fmt lint migrate

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/msmhq/msm/cmd/msm/cmd.Version=$(VERSION)"
PREFIX := /usr/local
SYSCONFDIR := /etc

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
