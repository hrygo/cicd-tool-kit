#!/usr/bin/env bash
# Copyright 2026 CICD AI Toolkit. All rights reserved.
#
# CI test script

set -euo pipefail

echo "Running tests..."

go test -v -race -coverprofile=coverage.out ./...

echo "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo "Tests complete!"
