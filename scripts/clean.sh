#!/usr/bin/env bash
# Copyright 2026 CICD AI Toolkit. All rights reserved.
#
# Clean temporary files

set -euo pipefail

echo "Cleaning..."

# Clean build artifacts
rm -rf bin/
rm -f coverage.out coverage.html

# Clean test cache
go clean -testcache

echo "Clean complete!"
