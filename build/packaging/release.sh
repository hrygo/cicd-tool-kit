#!/bin/bash
# release.sh - Release automation script for cicd-runner
#
# This script creates release artifacts including:
#   - Tarballs for each platform
#   - SHA256 checksums
#   - Release notes draft
#
# Usage: ./release.sh <version> [git_tag]
#   version: Version string (e.g., v1.0.0)
#   git_tag: Optional git tag to create (default: false)

set -euo pipefail

# Colors
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly NC='\033[0m'

# Script directory
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly BUILD_DIR="${PROJECT_ROOT}/build/bin"
readonly DIST_DIR="${PROJECT_ROOT}/build/dist"

VERSION="${1:-}"
CREATE_GIT_TAG="${2:-false}"

# Logging
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Validate version
validate_version() {
    if [[ -z "${VERSION}" ]]; then
        echo "Error: Version is required"
        echo "Usage: $0 <version> [git_tag]"
        exit 1
    fi

    # Validate semver format (basic check)
    if [[ ! "${VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
        echo "Error: Invalid version format. Expected: v1.0.0 or v1.0.0-beta.1"
        exit 1
    fi
}

# Build all platforms
build_all() {
    log_info "Building all platforms..."
    "${SCRIPT_DIR}/build-all.sh" "${VERSION}"
}

# Create distribution tarballs
create_tarballs() {
    log_info "Creating distribution tarballs..."
    mkdir -p "${DIST_DIR}"

    for binary in "${BUILD_DIR}"/cicd-runner-*; do
        if [[ -f "${binary}" ]]; then
            local basename
            basename="$(basename "${binary}")"
            local tarball="${DIST_DIR}/${basename}.tar.gz"

            log_info "Packaging ${basename}..."
            tar -czf "${tarball}" -C "${BUILD_DIR}" "${basename}"
        fi
    done

    # Copy checksums
    cp "${BUILD_DIR}/checksums.txt" "${DIST_DIR}/"
}

# Create release notes
create_release_notes() {
    log_info "Creating release notes draft..."
    local notes_file="${DIST_DIR}/RELEASE_NOTES.md"

    cat > "${notes_file}" <<EOF
# CICD AI Toolkit ${VERSION}

## Downloads

| Platform | Architecture | Download | Checksum |
|----------|--------------|----------|----------|
EOF

    for tarball in "${DIST_DIR}"/cicd-runner-*.tar.gz; do
        if [[ -f "${tarball}" ]]; then
            local basename
            basename="$(basename "${tarball}")"
            local platform arch
            platform="$(echo "${basename}" | sed -E 's/cicd-runner-(linux|darwin|windows)-.*/\1/')"
            arch="$(echo "${basename}" | sed -E 's/cicd-runner-(linux|darwin|windows)-(.*)\.tar\.gz/\2/')"
            local checksum
            checksum=$(grep "${basename%.tar.gz}" "${DIST_DIR}/checksums.txt" | awk '{print $1}')

            echo "| ${platform} | ${arch} | [\`${basename}\`](https://github.com/cicd-ai-toolkit/cicd-runner/releases/download/${VERSION}/${basename}) | \`${checksum}\` |" >> "${notes_file}"
        fi
    done

    cat >> "${notes_file}" <<EOF

## Installation

### Using install script
\`\`\`bash
curl -fsSL https://get.cicd-toolkit.com | bash -s -- --version ${VERSION}
\`\`\`

### Manual installation
\`\`\`bash
# Download and verify
wget https://github.com/cicd-ai-toolkit/cicd-runner/releases/download/${VERSION}/cicd-runner-linux-amd64.tar.gz
tar -xzf cicd-runner-linux-amd64.tar.gz
sudo mv cicd-runner-linux-amd64 /usr/local/bin/cicd-runner
chmod +x /usr/local/bin/cicd-runner
\`\`\`

## Docker Image
\`\`\`bash
docker pull ghcr.io/cicd-ai-toolkit/runner:${VERSION}
\`\`\`

## What's Changed

*TODO: Add changelog entries*
EOF

    log_info "Release notes written to ${notes_file}"
}

# Create git tag
create_git_tag() {
    if [[ "${CREATE_GIT_TAG}" == "true" ]]; then
        log_info "Creating git tag ${VERSION}..."
        git tag -a "${VERSION}" -m "Release ${VERSION}"
        log_info "Tag created. Use 'git push origin ${VERSION}' to push."
    fi
}

# Main
main() {
    validate_version
    build_all
    create_tarballs
    create_release_notes
    create_git_tag

    log_info "Release complete!"
    log_info "Artifacts in: ${DIST_DIR}"
    ls -lh "${DIST_DIR}"
}

main "$@"
