# Copyright 2026 CICD AI Toolkit. All rights reserved.

.PHONY: build build-all clean test lint docker release help

# Variables
BINARY_NAME=cicd-runner
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT_SHA=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS=-X github.com/cicd-ai-toolkit/pkg/version.Version=$(VERSION) \
        -X github.com/cicd-ai-toolkit/pkg/version.GitCommit=$(COMMIT_SHA) \
        -X github.com/cicd-ai-toolkit/pkg/version.BuildDate=$(BUILD_DATE)

# Default target
all: build

## build: Build the binary for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Built bin/$(BINARY_NAME)"

## build-all: Build binaries for all platforms
build-all:
	@echo "Building for all platforms..."
	@./build/packaging/build-all.sh

## test: Run tests
test:
	@echo "Running tests..."
	@go test -race -coverprofile=coverage.out ./...
	@echo "Tests passed"

## lint: Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping..."; \
	fi
	@echo "Lint passed"

## docker: Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t cicd-ai-toolkit:latest .
	@echo "Docker image built"

## docker-push: Build and push Docker image
docker-push: docker
	@echo "Pushing Docker image..."
	@docker push cicd-ai-toolkit:latest

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## release: Create a release
release:
	@echo "Creating release $(VERSION)..."
	@./build/packaging/release.sh $(VERSION)

## install: Install binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install -ldflags="$(LDFLAGS)" ./cmd/$(BINARY_NAME)

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@gofmt -w -s .
	@go mod tidy

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
