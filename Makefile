.PHONY: help build run test clean docker

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the server binary
	@echo "Building server..."
	go build -o bin/server cmd/server/main.go

run: ## Run the server
	@echo "Starting server..."
	go run cmd/server/main.go

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -cover ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	
lint: fmt ## Lint code
	@echo "Linting code..."
	go vet ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

docker: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t timeservice:latest .
