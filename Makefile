.PHONY: help test test-verbose test-coverage test-coverage-html build run clean lint

# Default target
help:
	@echo "Available targets:"
	@echo "  make test                - Run all tests"
	@echo "  make test-verbose        - Run tests with verbose output"
	@echo "  make test-coverage       - Run tests with coverage report"
	@echo "  make test-coverage-html  - Generate HTML coverage report"
	@echo "  make build               - Build the server binary"
	@echo "  make run                 - Run the server"
	@echo "  make clean               - Clean build artifacts and coverage files"
	@echo "  make lint                - Run linters (requires golangci-lint)"

# Run all tests
test:
	@echo "Running tests..."
	@go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...
	@echo ""
	@echo "Generating detailed coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

# Generate HTML coverage report
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build the server
build:
	@echo "Building server..."
	@mkdir -p bin
	@go build -o bin/server ./cmd/server
	@echo "Build complete: bin/server"

# Run the server
run: build
	@echo "Starting server..."
	@./bin/server

# Clean build artifacts and coverage files
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Run linters (requires golangci-lint to be installed)
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
	fi
