.PHONY: build clean install test fmt lint

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/msmhq/msm/cmd/msm/cmd.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/msm ./cmd/msm

install:
	go install $(LDFLAGS) ./cmd/msm

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

build-all: build build-linux

release: clean build-all
	@echo "Built binaries in bin/"
