# Chronicle Makefile

.PHONY: all test clean build doc lint docker format help examples

# Default target
all: lint test build

# Build the project
build:
	@echo "Building Chronicle..."
	go build -v ./...

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with short flag (skips integration tests)
test-short:
	@echo "Running short tests..."
	go test -v -short ./...

# Run only the example tests
examples:
	@echo "Running example tests..."
	cd examples && go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Format code
format:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Generate documentation
doc:
	@echo "Generating documentation..."
	go doc -all ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Stop any running infrastructure from tests
stop-infra:
	@echo "Stopping test infrastructure..."
	cd examples && docker-compose down -v

# Build Docker image with test environment
docker:
	@echo "Building Docker image..."
	docker build -t chronicle-test -f Dockerfile.test .

# Help target
help:
	@echo "Chronicle Makefile targets:"
	@echo "  all         - Default target: lint, test, and build"
	@echo "  build       - Build the project"
	@echo "  test        - Run all tests"
	@echo "  test-short  - Run tests with -short flag (skips integration tests)"
	@echo "  examples    - Run only the example tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  format      - Format code"
	@echo "  lint        - Lint code"
	@echo "  doc         - Generate documentation"
	@echo "  coverage    - Run tests with coverage"
	@echo "  stop-infra  - Stop any running infrastructure from tests"
	@echo "  docker      - Build Docker image with test environment"
	@echo "  help        - Show this help message"
