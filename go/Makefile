# Go Agent Framework SDK - Makefile
# 
# Common development commands for building, testing, and linting the Go SDK.
#

.PHONY: all build test test-verbose test-race coverage coverage-html lint vet fmt clean help

# Default target
all: build test lint

# Build all packages
build:
	go build -v ./...

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Run tests with coverage and show summary
coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	@echo ""
	@echo "=== Coverage Summary ==="
	@go tool cover -func=coverage.out | tail -n 1

# Generate HTML coverage report
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter (requires golangci-lint installed)
lint:
	golangci-lint run ./...

# Run go vet
vet:
	go vet ./...

# Format code
fmt:
	go fmt ./...

# Clean generated files
clean:
	rm -f coverage.out coverage.html

# Show help
help:
	@echo "Go Agent Framework SDK - Development Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all           Build, test, and lint (default)"
	@echo "  build         Build all packages"
	@echo "  test          Run tests"
	@echo "  test-verbose  Run tests with verbose output"
	@echo "  test-race     Run tests with race detector"
	@echo "  coverage      Run tests with coverage and show summary"
	@echo "  coverage-html Generate HTML coverage report"
	@echo "  lint          Run golangci-lint"
	@echo "  vet           Run go vet"
	@echo "  fmt           Format code with go fmt"
	@echo "  clean         Remove generated files"
	@echo "  help          Show this help message"
