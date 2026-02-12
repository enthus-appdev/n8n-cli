.PHONY: build install clean test lint fmt

BINARY_NAME=n8nctl
MAIN_PKG=./cmd/$(BINARY_NAME)
VERSION?=dev
LDFLAGS=-ldflags "-X github.com/enthus-appdev/n8n-cli/internal/cmd.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PKG)

install:
	go install $(LDFLAGS) $(MAIN_PKG)

clean:
	rm -rf bin/
	go clean

test:
	go test -v ./...

lint:
	golangci-lint run

fmt:
	go fmt ./...

# Cross-compilation
build-all: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 $(MAIN_PKG)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 $(MAIN_PKG)

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 $(MAIN_PKG)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 $(MAIN_PKG)

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PKG)

# Development
dev:
	go run $(MAIN_PKG) $(ARGS)

# Release (requires goreleaser)
release:
	goreleaser release --clean

release-snapshot:
	goreleaser release --snapshot --clean
