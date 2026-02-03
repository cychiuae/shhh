.PHONY: all build test test-security test-integration clean install lint fmt vet help

# Build variables
BINARY_NAME := shhh
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X github.com/cychiuae/shhh/cmd.Version=$(VERSION) -X github.com/cychiuae/shhh/cmd.BuildTime=$(BUILD_TIME)"

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOINSTALL := $(GO) install
GOFMT := gofmt
GOVET := $(GO) vet

# Default target
all: build

## build: Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

## build-all: Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .

build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

## install: Install the binary to $GOPATH/bin
install:
	$(GOINSTALL) $(LDFLAGS) .

## test: Run all tests
test:
	$(GOTEST) -v ./...

## test-unit: Run unit tests only
test-unit:
	$(GOTEST) -v ./internal/...

## test-security: Run security tests
test-security:
	$(GOTEST) -v ./test/security/...

## test-integration: Run integration tests
test-integration:
	$(GOTEST) -v ./test/integration/...

## test-cover: Run tests with coverage
test-cover:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run linters
lint: vet
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GOFMT) -s -w .

## fmt-check: Check code formatting
fmt-check:
	@test -z "$$($(GOFMT) -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

## vet: Run go vet
vet:
	$(GOVET) ./...

## security-scan: Run security scanners
security-scan:
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec -severity medium ./...
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html

## tidy: Tidy go modules
tidy:
	$(GO) mod tidy

## deps: Download dependencies
deps:
	$(GO) mod download

## version: Print version
version:
	@echo $(VERSION)

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'
