# Makefile for cicd-ai-toolkit
#
# Usage:
#   make build          - Build for current platform
#   make build-all      - Build for all platforms
#   make test           - Run tests
#   make lint           - Run linters
#   make docker         - Build Docker image
#   make release        - Create release artifacts

# ============================================
# Variables
# ============================================

# Project metadata
BINARY_NAME=cicd-runner
PACKAGE_NAME=github.com/cicd-ai-toolkit/cicd-runner

# Version info (default to dev, override with VERSION=v1.0.0)
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION?=$(shell go version | awk '{print $$3}')

# Build directories
BUILD_DIR=build/bin
DIST_DIR=build/dist

# Go build flags
LDFLAGS=-s -w \
    -X $(PACKAGE_NAME)/pkg/version.Version=$(VERSION) \
    -X $(PACKAGE_NAME)/pkg/version.BuildDate=$(BUILD_DATE) \
    -X $(PACKAGE_NAME)/pkg/version.GitCommit=$(GIT_COMMIT) \
    -X $(PACKAGE_NAME)/pkg/version.GoVersion=$(GO_VERSION)

# Go build tags
BUILD_TAGS=-trimpath

# ============================================
# Default target
# ============================================

.PHONY: all
all: build

# ============================================
# Build targets
# ============================================

.PHONY: build
build: ## Build for current platform
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(BUILD_TAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@./build/packaging/build-all.sh $(VERSION)

.PHONY: build-linux-amd64
build-linux-amd64: ## Build for linux/amd64
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(BUILD_TAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/$(BINARY_NAME)

.PHONY: build-linux-arm64
build-linux-arm64: ## Build for linux/arm64
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=arm64 go build $(BUILD_TAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/$(BINARY_NAME)

.PHONY: build-darwin-amd64
build-darwin-amd64: ## Build for darwin/amd64
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(BUILD_TAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/$(BINARY_NAME)

.PHONY: build-darwin-arm64
build-darwin-arm64: ## Build for darwin/arm64
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=arm64 go build $(BUILD_TAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/$(BINARY_NAME)

# ============================================
# Test targets
# ============================================

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test -race -coverprofile=coverage.out ./...

.PHONY: test-short
test-short: ## Run short tests
	@echo "Running short tests..."
	@go test -race -short ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -race ./test/integration/...

.PHONY: test-e2e
test-e2e: ## Run E2E tests
	@echo "Running E2E tests..."
	@go test -race ./test/e2e/...

.PHONY: test-coverage
test-coverage: ## Generate coverage report
	@echo "Generating coverage report..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# ============================================
# Lint targets
# ============================================

.PHONY: lint
lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -l -w .; \
	fi

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

# ============================================
# Docker targets
# ============================================

.PHONY: docker
docker: ## Build Docker image (production)
	@echo "Building Docker image..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t cicd-ai-toolkit/runner:$(VERSION) \
		-t cicd-ai-toolkit/runner:latest \
		-f Dockerfile .

.PHONY: docker-slim
docker-slim: ## Build Docker image (slim variant)
	@echo "Building Docker image (slim)..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t cicd-ai-toolkit/runner:$(VERSION)-slim \
		-t cicd-ai-toolkit/runner:latest-slim \
		-f Dockerfile.slim .

.PHONY: docker-buildx
docker-buildx: ## Build Docker image for multiple platforms
	@echo "Building Docker image for multiple platforms..."
	@docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t cicd-ai-toolkit/runner:$(VERSION) \
		-f Dockerfile \
		--load .

# ============================================
# Release targets
# ============================================

.PHONY: release
release: clean build-all ## Create release artifacts
	@echo "Creating release for $(VERSION)..."
	@./build/packaging/release.sh $(VERSION)

.PHONY: checksum
checksum: ## Generate checksums
	@echo "Generating checksums..."
	@./build/packaging/checksum.sh generate

.PHONY: checksum-verify
checksum-verify: ## Verify checksums
	@echo "Verifying checksums..."
	@./build/packaging/checksum.sh verify

# ============================================
# Clean targets
# ============================================

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html

.PHONY: clean-all
clean-all: clean ## Clean everything including dependencies
	@echo "Cleaning all..."
	@go clean -cache -testcache -modcache

# ============================================
# Install targets
# ============================================

.PHONY: install
install: build ## Install binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@go install $(BUILD_TAGS) -ldflags "$(LDFLAGS)" ./cmd/$(BINARY_NAME)

.PHONY: install-local
install-local: build ## Install binary to /usr/local/bin (requires sudo)
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)

# ============================================
# Development targets
# ============================================

.PHONY: dev
dev: ## Run in development mode
	@echo "Running $(BINARY_NAME) in development mode..."
	@go run ./cmd/$(BINARY_NAME) run

.PHONY: run
run: build ## Run the built binary
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME) run

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# ============================================
# Info targets
# ============================================

.PHONY: info
info: ## Show build information
	@echo "Version:      $(VERSION)"
	@echo "Build Date:   $(BUILD_DATE)"
	@echo "Git Commit:   $(GIT_COMMIT)"
	@echo "Go Version:   $(GO_VERSION)"
	@echo "Build Dir:    $(BUILD_DIR)"
	@echo "Dist Dir:     $(DIST_DIR)"

.PHONY: help
help: ## Show this help message
	@echo "$(BINARY_NAME) - CICD AI Toolkit"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
