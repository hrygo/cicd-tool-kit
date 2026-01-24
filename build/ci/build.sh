#!/usr/bin/env bash
# Copyright 2026 CICD AI Toolkit. All rights reserved.
#
# CI build script

set -euo pipefail

echo "Running CI build..."

# Format check
echo "Checking code format..."
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
    echo "Unformatted files:"
    echo "$UNFORMATTED"
    exit 1
fi

# Run tests
echo "Running tests..."
go test -race -coverprofile=coverage.out ./...

# Build
echo "Building..."
go build -v -o bin/cicd-runner ./cmd/cicd-runner

echo "CI build complete!"
