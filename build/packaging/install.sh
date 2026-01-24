#!/bin/bash
# install.sh - One-click installation script for cicd-runner
#
# Usage:
#   curl -fsSL https://get.cicd-toolkit.com | bash
#   curl -fsSL https://get.cicd-toolkit.com | bash -s -- --version v1.0.0
#   curl -fsSL https://get.cicd-toolkit.com | bash -s -- --help
#
# Environment variables:
#   CICD_INSTALL_DIR  - Installation directory (default: /usr/local/bin)
#   CICD_VERSION      - Version to install (default: latest)
#   CICD_VERIFY_SIG   - Enable signature verification (default: true)

set -euo pipefail

# ============================================
# Configuration
# ============================================

# Installation URLs
# BASE_URL can be set via environment variable during release
readonly BASE_URL="${CICD_BASE_URL:-https://github.com/cicd-ai-toolkit/cicd-runner/releases}"
readonly CHECKSUM_URL="${CICD_CHECKSUM_URL:-${BASE_URL}/download/latest/checksums.txt}"

# Installation directory
INSTALL_DIR="${CICD_INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="cicd-runner"

# Version to install
VERSION="${CICD_VERSION:-latest}"

# Enable signature verification (default: true)
VERIFY_SIG="${CICD_VERIFY_SIG:-true}"

# Temporary directory
TMP_DIR="$(mktemp -d)"

# Cleanup on exit
trap 'rm -rf "${TMP_DIR}"' EXIT

# ============================================
# Colors
# ============================================

readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# ============================================
# Logging
# ============================================

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    if [[ "${DEBUG:-0}" == "1" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# ============================================
# OS/Arch Detection
# ============================================

detect_os_arch() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux)    os="linux" ;;
        Darwin)   os="darwin" ;;
       MINGW*|MSYS*|CYGWIN*)
            os="windows"
            BINARY_NAME="cicd-runner.exe"
            ;;
        *)
            log_error "Unsupported OS: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect Architecture
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l)       arch="armv7" ;;
        i386|i686)    arch="386" ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    OS="${os}"
    ARCH="${arch}"
    log_debug "Detected: ${OS}/${ARCH}"
}

# ============================================
# Download Functions
# ============================================

download_file() {
    local url="$1"
    local output="$2"

    log_debug "Downloading: ${url}"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "${url}" -o "${output}"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "${url}" -O "${output}"
    else
        log_error "Neither curl nor wget is installed"
        exit 1
    fi
}

# ============================================
# Checksum Verification
# ============================================

verify_checksum() {
    local file="$1"
    local expected_checksum="$2"

    log_info "Verifying checksum..."

    local actual_checksum
    if command -v sha256sum >/dev/null 2>&1; then
        actual_checksum="$(sha256sum "${file}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        actual_checksum="$(shasum -a 256 "${file}" | awk '{print $1}')"
    else
        log_warn "No checksum tool available, skipping verification"
        return 0
    fi

    if [[ "${actual_checksum}" != "${expected_checksum}" ]]; then
        log_error "Checksum mismatch!"
        log_error "Expected: ${expected_checksum}"
        log_error "Actual:   ${actual_checksum}"
        exit 1
    fi

    log_info "Checksum verified successfully"
}

# ============================================
# Signature Verification (Cosign)
# ============================================

verify_signature() {
    local file="$1"
    local sig_url="${BASE_URL}/download/${VERSION}/${BINARY_NAME}-${OS}-${ARCH}.sig"
    local cert_url="${BASE_URL}/download/${VERSION}/cert.pem"
    local sig_file="${TMP_DIR}/${BINARY_NAME}.sig"
    local cert_file="${TMP_DIR}/cert.pem"

    # Skip if cosign is not available or verification is disabled
    if [[ "${VERIFY_SIG}" != "true" ]]; then
        log_warn "Signature verification disabled by CICD_VERIFY_SIG"
        return 0
    fi

    if ! command -v cosign >/dev/null 2>&1; then
        log_warn "cosign not found. Install for signature verification:"
        log_warn "  https://docs.sigstore.dev/cosign/installation/"
        log_warn "Skipping signature verification..."
        return 0
    fi

    log_info "Verifying signature..."

    # Download signature and certificate
    download_file "${sig_url}" "${sig_file}" || {
        log_warn "Signature file not found, skipping verification"
        return 0
    }
    download_file "${cert_url}" "${cert_file}" || {
        log_warn "Certificate not found, skipping verification"
        return 0
    }

    # Verify using cosign
    if cosign verify-blob \
        --certificate "${cert_file}" \
        --signature "${sig_file}" \
        --certificate-identity "https://github.com/cicd-ai-toolkit/cicd-runner/.github/workflows/release.yml@refs/tags/${VERSION}" \
        --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
        "${file}"; then
        log_info "Signature verified successfully"
        return 0
    else
        log_error "Signature verification failed!"
        log_error "The binary may have been tampered with."
        exit 1
    fi
}

# ============================================
# Installation
# ============================================

install_binary() {
    local download_url="${BASE_URL}/download/${VERSION}/${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
    local checksum_url="${BASE_URL}/download/${VERSION}/checksums.txt"

    local download_file="${TMP_DIR}/${BINARY_NAME}.tar.gz"
    local checksum_file="${TMP_DIR}/checksums.txt"

    # Download checksums first
    log_info "Downloading checksums..."
    download_file "${checksum_url}" "${checksum_file}"

    # Get expected checksum
    local expected_checksum
    expected_checksum="$(grep "${BINARY_NAME}-${OS}-${ARCH}" "${checksum_file}" | awk '{print $1}')"

    if [[ -z "${expected_checksum}" ]]; then
        log_warn "Checksum not found for ${BINARY_NAME}-${OS}-${ARCH}, skipping verification"
    fi

    # Download binary
    log_info "Downloading ${BINARY_NAME}-${OS}-${ARCH}..."
    download_file "${download_url}" "${download_file}"

    # Verify checksum
    if [[ -n "${expected_checksum}" ]]; then
        verify_checksum "${download_file}" "${expected_checksum}"
    fi

    # Verify signature (if cosign is available)
    verify_signature "${download_file}"

    # Extract
    log_info "Extracting..."
    tar -xzf "${download_file}" -C "${TMP_DIR}"

    # Install
    log_info "Installing to ${INSTALL_DIR}..."

    # Check if directory exists and is writable
    if [[ ! -d "${INSTALL_DIR}" ]]; then
        sudo mkdir -p "${INSTALL_DIR}"
    fi

    if [[ -w "${INSTALL_DIR}" ]]; then
        mv "${TMP_DIR}/${BINARY_NAME}-${OS}-${ARCH}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "${TMP_DIR}/${BINARY_NAME}-${OS}-${ARCH}" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chown "$(id -un):$(id -gn)" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    log_info "Installation complete!"
}

# ============================================
# Post-Install Verification
# ============================================

verify_installation() {
    if command -v "${BINARY_NAME}" >/dev/null 2>&1; then
        local version
        version="$("${BINARY_NAME}" --version 2>/dev/null || echo "unknown")"
        log_info "Successfully installed ${BINARY_NAME} (${version})"
    else
        log_warn "Binary installed but not in PATH. Add ${INSTALL_DIR} to your PATH."
    fi
}

# ============================================
# Help
# ============================================

show_help() {
    cat <<EOF
CICD AI Toolkit Installer

Usage:
  curl -fsSL https://get.cicd-toolkit.com | bash [OPTIONS]

Options:
  --version VERSION   Install specific version (default: latest)
  --install-dir DIR   Installation directory (default: /usr/local/bin)
  --help              Show this help message

Environment Variables:
  CICD_VERSION        Version to install
  CICD_INSTALL_DIR    Installation directory

Examples:
  # Install latest version
  curl -fsSL https://get.cicd-toolkit.com | bash

  # Install specific version
  curl -fsSL https://get.cicd-toolkit.com | bash -s -- --version v1.0.0

  # Install to custom directory
  curl -fsSL https://get.cicd-toolkit.com | CICD_INSTALL_DIR=~/bin bash

EOF
}

# ============================================
# Main
# ============================================

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    log_info "CICD AI Toolkit Installer"
    log_info "Version: ${VERSION}"
    log_info "Install directory: ${INSTALL_DIR}"

    detect_os_arch
    install_binary
    verify_installation
}

main "$@"
