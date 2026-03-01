.PHONY: all build run test lint clean docker-build docker-up docker-down dev help

# Variables
BINARY_NAME=salesmate
DOCKER_IMAGE=salesmate
GO=go
GOFLAGS=-v

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd

## run: Run the gateway
run: build
	@echo "Running $(BINARY_NAME) gateway..."
	./$(BINARY_NAME) gateway

## dev: Run with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

## test: Run tests
test:
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	$(GO) clean

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):latest .

## docker-up: Start all services with Docker Compose
docker-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up --build -d
	@echo "Services started. Use 'make docker-logs' to view logs."

## docker-down: Stop all Docker services
docker-down:
	@echo "Stopping Docker services..."
	docker-compose down

## docker-logs: View Docker logs
docker-logs:
	docker-compose logs -f app

## docker-ps: Show Docker container status
docker-ps:
	docker-compose ps

## init: Initialize the project (create .env, initialize knowledge base)
init:
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
		echo "Please edit .env with your configuration"; \
	fi
	@chmod +x scripts/init_kb.sh 2>/dev/null || true
	@./scripts/init_kb.sh 2>/dev/null || echo "Knowledge base initialization skipped"

## install: Install dependencies
install:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## help: Show this help message
help:
	@echo "SalesMate AI - Makefile Commands"
	@echo "================================="
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'

# Include dependencies as phony targets
.PHONY: build run test lint clean docker-build docker-up docker-down dev help init install fmt vet test-coverage