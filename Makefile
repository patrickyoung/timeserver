.PHONY: help deps fmt vet lint test test-verbose test-race test-coverage test-coverage-html build run docker clean ci-local security-audit vuln-check validate-versions

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
	@echo "  make validate-versions   - Validate version consistency across project"
	@echo "  make security-audit      - Run security audits (gosec + vuln check)"
	@echo "  make vuln-check          - Check for dependency vulnerabilities"
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

# Validate version consistency
validate-versions:
	@echo "Validating version consistency..."
	@./scripts/validate-versions.sh

# Build the server
build: validate-versions
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
docker: validate-versions
	@echo "Building Docker image..."
	@docker build -t timeservice:v1.0.0 -t timeservice:latest .
	@echo "Docker images built:"
	@echo "  - timeservice:v1.0.0 (use for production deployments)"
	@echo "  - timeservice:latest (for local development only)"

# Run security audits
security-audit:
	@echo "Running security audit..."
	@echo ""
	@echo "1. Running gosec security scanner..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -fmt=text ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi
	@echo ""
	@echo "2. Checking for dependency vulnerabilities..."
	@$(MAKE) vuln-check
	@echo ""
	@echo "Security audit complete!"

# Check for dependency vulnerabilities
vuln-check:
	@echo "Checking for known vulnerabilities in dependencies..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		echo "Falling back to go list -m all for dependency check..."; \
		go list -m all; \
	fi

# Run all CI checks locally
ci-local: validate-versions deps fmt vet lint test-race test-coverage security-audit
	@echo ""
	@echo "========================================"
	@echo "All CI checks passed! ✓"
	@echo "========================================"
