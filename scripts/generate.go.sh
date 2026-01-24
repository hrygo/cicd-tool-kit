#!/usr/bin/env bash
# Copyright 2026 CICD AI Toolkit. All rights reserved.
#
# Generate Go code (mocks, etc.)

set -euo pipefail

echo "Generating Go code..."

# Run go generate
go generate ./...

echo "Code generation complete!"
