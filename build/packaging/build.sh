#!/usr/bin/env bash
# Copyright 2026 CICD AI Toolkit. All rights reserved.
#
# Build script for cross-platform compilation

set -euo pipefail

VERSION="${VERSION:-dev}"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
COMMIT_SHA="${COMMIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"

LDFLAGS="-X github.com/cicd-ai-toolkit/pkg/version.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X github.com/cicd-ai-toolkit/pkg/version.GitCommit=${COMMIT_SHA}"
LDFLAGS="${LDFLAGS} -X github.com/cicd-ai-toolkit/pkg/version.BuildDate=${BUILD_DATE}"

GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

echo "Building for ${GOOS}/${GOARCH}..."

CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
    -ldflags="${LDFLAGS} -w -s" \
    -o "bin/cicd-runner-${GOOS}-${GOARCH}" \
    ./cmd/cicd-runner

echo "Built bin/cicd-runner-${GOOS}-${GOARCH}"
