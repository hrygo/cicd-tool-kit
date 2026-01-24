#!/bin/bash
# build-all.sh - Multi-architecture build script for cicd-runner
#
# This script builds the cicd-runner binary for multiple platforms and architectures.
# Supported targets:
#   - linux/amd64
#   - linux/arm64
#   - darwin/amd64
#   - darwin/arm64
#
# Usage: ./build-all.sh [version]
#   version: Optional version string (default: dev)

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly NC='\033[0m' # No Color

# Script directory
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly BUILD_DIR="${PROJECT_ROOT}/build/bin"
readonly VERSION="${1:-dev}"

# Build targets
readonly TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# Binary name
readonly BINARY_NAME="cicd-runner"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Clean build directory
clean() {
    log_info "Cleaning build directory..."
    rm -rf "${BUILD_DIR}"
    mkdir -p "${BUILD_DIR}"
}

# Build for a specific target
build_target() {
    local target="$1"
    local goos goos_suffix goarch

    goos="${target%/*}"
    goarch="${target#*/}"

    # Determine output filename
    local output_name="${BINARY_NAME}-${goos}-${goarch}"
    if [[ "${goos}" == "windows" ]]; then
        output_name="${output_name}.exe"
    fi

    log_info "Building for ${target}..."

    # Build with GoReleaser-style ldflags
    local ldflags="
        -s -w
        -X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.Version=${VERSION}
        -X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)
        -X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
        -X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.GoVersion=$(go version | awk '{print $3}')
    "

    GOOS="${goos}" GOARCH="${goarch}" go build \
        -ldflags "${ldflags}" \
        -trimpath \
        -o "${BUILD_DIR}/${output_name}" \
        "${PROJECT_ROOT}/cmd/cicd-runner/main.go"

    # Create symlink without platform suffix for current platform
    local current_goos="$(go env GOOS)"
    local current_goarch="$(go env GOARCH)"
    if [[ "${goos}" == "${current_goos}" && "${goarch}" == "${current_goarch}" ]]; then
        ln -sf "${output_name}" "${BUILD_DIR}/${BINARY_NAME}"
    fi

    log_info "Created ${output_name}"
}

# Generate checksums
generate_checksums() {
    log_info "Generating SHA256 checksums..."
    cd "${BUILD_DIR}"
    sha256sum "${BINARY_NAME}"* > checksums.txt 2>/dev/null || shasum -a 256 "${BINARY_NAME}"* > checksums.txt
    cd "${PROJECT_ROOT}"
    log_info "Checksums written to ${BUILD_DIR}/checksums.txt"
}

# Main build process
main() {
    log_info "Starting multi-architecture build..."
    log_info "Version: ${VERSION}"
    log_info "Build directory: ${BUILD_DIR}"

    clean

    for target in "${TARGETS[@]}"; do
        build_target "${target}"
    done

    generate_checksums

    log_info "Build complete!"
    log_info "Binaries available in: ${BUILD_DIR}"
    ls -lh "${BUILD_DIR}"
}

main "$@"
