# Makefile for growth-spark-ai-service

.PHONY: test test-unit test-integration test-coverage clean build run

# Default target
all: test

# Run all tests
test:
	go test -v ./...

# Run only unit tests (fast)cl
test-unit:
	go test -v -short ./...

# Run only integration tests
test-integration:
	go test -v -run Integration ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run specific test
test-models:
	go test -v ./models/

test-database:
	go test -v ./database/

test-handlers:
	go test -v ./handlers/

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html
	go clean -testcache

# Build the application
build:
	go build -o bin/growth-spark-ai-service .

# Run the application
run:
	go run main.go

# Install dependencies
deps:
	go mod tidy
	go mod download

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Run tests in verbose mode with race detection
test-race:
	go test -v -race ./...

# Benchmark tests
benchmark:
	go test -bench=. -benchmem ./...

# Help
help:
	@echo "Available targets:"
	@echo "  test           - Run all tests"
	@echo "  test-unit      - Run unit tests only (skip integration)"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-models    - Run model tests only"
	@echo "  test-database  - Run database tests only"
	@echo "  test-handlers  - Run handler tests only"
	@echo "  clean          - Clean test artifacts"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  deps           - Install dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  test-race      - Run tests with race detection"
	@echo "  benchmark      - Run benchmark tests"
	@echo "  help           - Show this help"
