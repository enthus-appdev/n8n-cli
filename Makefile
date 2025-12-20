.PHONY: build install clean test lint fmt

BINARY_NAME=n8n
VERSION?=dev
LDFLAGS=-ldflags "-X github.com/hinne/n8n-cli/internal/cmd.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/n8n

install:
	go install $(LDFLAGS) ./cmd/n8n

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
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/n8n
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/n8n

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/n8n
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/n8n

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/n8n

# Development
dev:
	go run ./cmd/n8n $(ARGS)

# Release (requires goreleaser)
release:
	goreleaser release --clean

release-snapshot:
	goreleaser release --snapshot --clean
