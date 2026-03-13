# Go application Makefile
BINARY_NAME = kubara
SRC = main.go

.PHONY: all build run clean modtidy build-all build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64 test test-verbose test-coverage test-race lint format vet help

# Default target: Build the binary for the local platform
all: build

# Build the binary in the project root
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@go build -o $(BINARY_NAME) $(SRC)

# Run the application without building a persistent binary
run:
	@echo "Running $(BINARY_NAME)..."
	@go run $(SRC)

# Clean: remove all generated binaries
clean:
	@echo "Cleaning Go binaries..."
	@go clean
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME)-linux-amd64
	@rm -f $(BINARY_NAME)-linux-arm64
	@rm -f $(BINARY_NAME)-darwin-amd64
	@rm -f $(BINARY_NAME)-darwin-arm64
	@rm -f $(BINARY_NAME)-windows-amd64.exe
	@rm -f $(BINARY_NAME)-windows-arm64.exe

# Tidy the module files (go.mod and go.sum)
modtidy:
	@echo "Tidying Go modules..."
	@go mod tidy

# Run tests with default settings
test:
	@echo "Running Go tests..."
	@go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running Go tests with verbose output..."
	@go test -v ./...

# Run tests with coverage report
test-coverage:
	@echo "Running Go tests with coverage..."
	@go test -cover ./...

# Run tests with race detector enabled
test-race:
	@echo "Running Go tests with race detector..."
	@go test -race ./...

# Run go vet to analyze code for suspicious constructs
vet:
	@echo "Running go vet..."
	@go vet ./...

# Format code
format:
	@echo "Formatting Go code..."
	@go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting Go code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "Or visit https://golangci-lint.run/usage/install/ for other installation methods"; \
		exit 1; \
	fi

# Build binaries for all common platforms
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64

build-linux-amd64:
	@echo "Building for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 $(SRC)

build-linux-arm64:
	@echo "Building for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME)-linux-arm64 $(SRC)

build-darwin-amd64:
	@echo "Building for macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 $(SRC)

build-darwin-arm64:
	@echo "Building for macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 $(SRC)

build-windows-amd64:
	@echo "Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe $(SRC)

build-windows-arm64:
	@echo "Building for Windows ARM64..."
	@GOOS=windows GOARCH=arm64 go build -o $(BINARY_NAME)-windows-arm64.exe $(SRC)

help:
	@echo "Go application targets:"
	@echo "  build              - Build for current platform"
	@echo "  run                - Run application"
	@echo "  clean              - Clean binaries"
	@echo "  modtidy            - Tidy Go modules"
	@echo "  test               - Run tests"
	@echo "  test-verbose       - Run tests with verbose output"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  test-race          - Run tests with race detector"
	@echo "  vet                - Run go vet analyzer"
	@echo "  lint               - Lint code (requires golangci-lint)"
	@echo "  format             - Format code"
	@echo "  build-all          - Build for all platforms"
	@echo "  build-linux-*      - Build for Linux platforms"
	@echo "  build-darwin-*     - Build for macOS platforms"
	@echo "  build-windows-*    - Build for Windows platforms"
	@echo "  help               - Show this help"
