# Makefile for CI/CD Toolkit
.PHONY: help install-hooks lint test test-full build clean check-all fmt vet staticcheck

# Default target
help:
	@echo "CI/CD Toolkit - Available commands:"
	@echo ""
	@echo "  make install-hooks  - Install git hooks (pre-commit, pre-push, commit-msg)"
	@echo "  make lint           - Run all linters (gofmt, vet, staticcheck)"
	@echo "  make test           - Run tests (short mode)"
	@echo "  make test-full      - Run full test suite"
	@echo "  make build          - Build all packages"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make check-all      - Run all checks (lint + test)"
	@echo "  make fmt            - Format Go code with gofmt"
	@echo "  make vet            - Run go vet"
	@echo "  make staticcheck    - Run staticcheck (requires installation)"
	@echo ""

# Install git hooks
install-hooks:
	@echo "Installing git hooks..."
	@./scripts/install-hooks.sh

# Format code
fmt:
	@echo "Formatting Go code..."
	@gofmt -l -w .
	@echo "✓ Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run staticcheck (requires installation)
staticcheck:
	@echo "Running staticcheck..."
	@which staticcheck > /dev/null || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	@staticcheck ./...

# Run all linters
lint: fmt vet staticcheck
	@echo "✓ All linters passed"

# Run tests (short mode - skips integration tests)
test:
	@echo "Running tests (short mode)..."
	@go test -short ./...

# Run full test suite
test-full:
	@echo "Running full test suite..."
	@go test -timeout=5m ./...

# Build all packages
build:
	@echo "Building all packages..."
	@go build ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@go clean -cache -testcache
	@rm -rf dist/ bin/

# Run all checks (used by CI)
check-all: vet test
	@echo "✓ All checks passed"
