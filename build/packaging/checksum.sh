#!/bin/bash
# checksum.sh - Generate and verify SHA256 checksums
#
# Usage: ./checksum.sh [generate|verify] [file...]

set -euo pipefail

# Colors
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[0;33m'
readonly NC='\033[0m'

# Script directory
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly BUILD_DIR="${PROJECT_ROOT}/build/bin"
readonly CHECKSUM_FILE="${BUILD_DIR}/checksums.txt"

# Logging
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Generate checksums
generate() {
    log_info "Generating checksums..."

    cd "${BUILD_DIR}"

    # Remove existing checksums file
    rm -f "${CHECKSUM_FILE}"

    # Generate checksums for cicd-runner binaries
    for binary in cicd-runner-*; do
        if [[ -f "${binary}" ]]; then
            sha256sum "${binary}" >> "${CHECKSUM_FILE}" 2>/dev/null || shasum -a 256 "${binary}" >> "${CHECKSUM_FILE}"
            log_info "Added: ${binary}"
        fi
    done

    cd "${PROJECT_ROOT}"

    log_info "Checksums written to: ${CHECKSUM_FILE}"
    cat "${CHECKSUM_FILE}"
}

# Verify checksums
verify() {
    if [[ ! -f "${CHECKSUM_FILE}" ]]; then
        log_error "Checksum file not found: ${CHECKSUM_FILE}"
        exit 1
    fi

    log_info "Verifying checksums..."

    cd "${BUILD_DIR}"

    if sha256sum -c "${CHECKSUM_FILE}" 2>/dev/null; then
        log_info "All checksums verified successfully!"
    elif shasum -a 256 -c "${CHECKSUM_FILE}" 2>/dev/null; then
        log_info "All checksums verified successfully!"
    else
        log_error "Checksum verification failed!"
        exit 1
    fi

    cd "${PROJECT_ROOT}"
}

# Verify specific file
verify_file() {
    local file="$1"

    if [[ ! -f "${file}" ]]; then
        log_error "File not found: ${file}"
        exit 1
    fi

    if [[ ! -f "${CHECKSUM_FILE}" ]]; then
        log_error "Checksum file not found: ${CHECKSUM_FILE}"
        exit 1
    fi

    local filename
    filename="$(basename "${file}")"

    log_info "Verifying: ${filename}"

    cd "${BUILD_DIR}"

    if grep -q "${filename}" "${CHECKSUM_FILE}"; then
        sha256sum -c <(grep "${filename}" "${CHECKSUM_FILE}") 2>/dev/null || \
        shasum -a 256 -c <(grep "${filename}" "${CHECKSUM_FILE}") 2>/dev/null || \
        (log_error "Verification failed for ${filename}" && exit 1)

        log_info "Verified: ${filename}"
    else
        log_warn "No checksum found for: ${filename}"
    fi

    cd "${PROJECT_ROOT}"
}

# Main
main() {
    local action="${1:-generate}"
    shift || true

    case "${action}" in
        generate)
            generate
            ;;
        verify)
            if [[ $# -gt 0 ]]; then
                for file in "$@"; do
                    verify_file "${file}"
                done
            else
                verify
            fi
            ;;
        *)
            echo "Usage: $0 [generate|verify] [file...]"
            exit 1
            ;;
    esac
}

main "$@"
