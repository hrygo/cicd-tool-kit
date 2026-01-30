#!/bin/bash
# Install git hooks from .githooks directory to .git/hooks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

GITHOOKS_DIR=".githooks"
GIT_HOOKS_DIR=".git/hooks"

echo -e "${YELLOW}Installing git hooks...${NC}"

# Check if .githooks directory exists
if [ ! -d "$GITHOOKS_DIR" ]; then
    echo -e "${RED}✗ Error: $GITHOOKS_DIR directory not found${NC}"
    exit 1
fi

# Create .git/hooks directory if it doesn't exist
mkdir -p "$GIT_HOOKS_DIR"

# Copy each hook from .githooks to .git/hooks
HOOKS_INSTALLED=0
for hook in "$GITHOOKS_DIR"/*; do
    if [ -f "$hook" ]; then
        hook_name=$(basename "$hook")
        target="$GIT_HOOKS_DIR/$hook_name"

        # Backup existing hook if it's not a symlink
        if [ -f "$target" ] && [ ! -L "$target" ]; then
            backup="$target.backup.$(date +%Y%m%d%H%M%S)"
            echo -e "${YELLOW}Backing up existing $hook_name to $backup${NC}"
            cp "$target" "$backup"
        fi

        # Copy the hook and make it executable
        cp "$hook" "$target"
        chmod +x "$target"
        echo -e "${GREEN}✓ Installed $hook_name${NC}"
        HOOKS_INSTALLED=$((HOOKS_INSTALLED + 1))
    fi
done

# Configure git to use the githooks directory
git config core.hooksPath ".githooks" 2>/dev/null || true

echo ""
echo -e "${GREEN}✓ Successfully installed $HOOKS_INSTALLED git hook(s)${NC}"
echo ""
echo -e "${YELLOW}Installed hooks:${NC}"
echo "  - pre-commit: Runs go vet, gofmt, go test, staticcheck"
echo "  - pre-push:  Runs full test suite and build check"
echo ""
echo -e "${YELLOW}To bypass pre-commit hook: git commit --no-verify${NC}"
echo -e "${YELLOW}To bypass pre-push hook:  GIT_SKIP_PRE_PUSH=1 git push${NC}"
