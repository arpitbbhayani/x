# Makefile for x - Natural Language Shell Command Executor

BINARY_NAME=x
VERSION=$(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_DIR=build
LDFLAGS=-ldflags "-X github.com/REDFOX1899/ask-sh/internal/cli.Version=$(VERSION)"

.PHONY: all build clean install uninstall test fmt lint help

all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/x

## build-all: Build for multiple platforms
build-all: clean
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/x
	@echo "Building for Linux (arm64)..."
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/x
	@echo "Building for macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/x
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/x
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/x

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

## install: Install the binary to /usr/local/bin
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Done! Run 'x --help' to get started."

## uninstall: Remove the binary from /usr/local/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

## test: Run tests
test:
	go test -v ./...

## fmt: Format code
fmt:
	go fmt ./...

## lint: Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

## tidy: Tidy go modules
tidy:
	go mod tidy

## run: Build and run with arguments
run: build
	./$(BINARY_NAME) $(ARGS)

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
