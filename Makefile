# Chronicle Makefile

.PHONY: all build web-build web-install clean dev test lint help

# Default target
all: build

# Build the Go binary (depends on web-build)
build: web-build
	go build -o bin/chronicle ./cmd/chronicle

# Build the web frontend
web-build: web-install
	cd web && npm run build

# Install web dependencies
web-install:
	cd web && npm install

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -rf web/node_modules/

# Run daemon in development mode (without embedded UI)
dev:
	go run ./cmd/chronicle daemon --config chronicle.yaml

# Run tests
test:
	go test ./...

# Run linter
lint:
	golangci-lint run ./...

# Help
help:
	@echo "Chronicle Makefile targets:"
	@echo "  all        - Build everything (default)"
	@echo "  build      - Build the Go binary with embedded web UI"
	@echo "  web-build  - Build the web frontend"
	@echo "  web-install- Install web dependencies"
	@echo "  clean      - Remove build artifacts"
	@echo "  dev        - Run daemon in development mode"
	@echo "  test       - Run tests"
	@echo "  lint       - Run linter"
	@echo "  help       - Show this help"
