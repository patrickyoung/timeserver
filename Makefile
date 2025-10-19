.PHONY: help deps fmt vet lint test test-verbose test-race test-coverage test-coverage-html build run docker clean ci-local

# Default target
help:
	@echo "Available targets:"
	@echo "  make deps                - Download dependencies"
	@echo "  make fmt                 - Format code with go fmt"
	@echo "  make vet                 - Run go vet"
	@echo "  make lint                - Run golangci-lint"
	@echo "  make test                - Run all tests"
	@echo "  make test-verbose        - Run tests with verbose output"
	@echo "  make test-race           - Run tests with race detector"
	@echo "  make test-coverage       - Run tests with coverage report"
	@echo "  make test-coverage-html  - Generate HTML coverage report"
	@echo "  make build               - Build the server binary"
	@echo "  make run                 - Run the server"
	@echo "  make docker              - Build Docker image"
	@echo "  make clean               - Clean build artifacts and coverage files"
	@echo "  make ci-local            - Run all CI checks locally"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify
	@echo "Dependencies downloaded and verified"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "go vet passed"

# Run all tests
test:
	@echo "Running tests..."
	@go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test -race ./...
	@echo "Race detector tests passed"

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

# Build Docker image
# Builds with both version tag (for production) and latest tag (for convenience)
docker:
	@echo "Building Docker image..."
	@docker build -t timeservice:v1.0.0 -t timeservice:latest .
	@echo "Docker images built:"
	@echo "  - timeservice:v1.0.0 (use for production deployments)"
	@echo "  - timeservice:latest (for local development only)"

# Run all CI checks locally
ci-local: deps fmt vet lint test-race test-coverage
	@echo ""
	@echo "========================================"
	@echo "All CI checks passed! ✓"
	@echo "========================================"
