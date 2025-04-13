.PHONY: test test-verbose test-coverage clean

# Default target
all: test

# Build the CLI
build:
	go build -o bin/helm-schema-gen cmd/helm-schema-gen/main.go

# Run tests
test:
	go test ./pkg/... -v

# Run tests with verbose output
test-verbose:
	go test ./pkg/... -v

# Run tests with coverage report
test-coverage:
	go test ./pkg/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Run tests for a specific package
test-pkg:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-pkg PKG=<package-path>"; \
		echo "Example: make test-pkg PKG=./pkg/logging"; \
		exit 1; \
	fi
	go test $(PKG) -v

# Clean up
clean:
	rm -f coverage.out coverage.html helm-schema-generator
	find . -type f -name "*.test" -delete

# Help
help:
	@echo "Available targets:"
	@echo "  make test          - Run all tests"
	@echo "  make test-verbose  - Run tests with verbose output"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make test-pkg      - Run tests for a specific package (use PKG=./pkg/...)"
	@echo "  make clean         - Clean up test artifacts"
	@echo "  make help          - Show this help" 