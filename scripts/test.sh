#!/usr/bin/env bash
# Copyright 2026 CICD AI Toolkit. All rights reserved.
#
# Run tests

set -euo pipefail

echo "Running tests..."

go test -race -coverprofile=coverage.out ./...

echo "Tests complete!"
