#!/bin/bash
# install-hooks.sh - Install git hooks from .githooks directory

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_SRC_DIR="$PROJECT_ROOT/.githooks"
HOOKS_DIR="$(git rev-parse --git-common-dir)/hooks"

echo "📦 Installing git hooks for CICD Runner..."
echo ""

# Ensure .githooks directory exists
if [ ! -d "$HOOKS_SRC_DIR" ]; then
    echo "  ✗ .githooks directory not found at $HOOKS_SRC_DIR"
    exit 1
fi

# Copy pre-commit hook (lightweight - runs on every commit)
if [ -f "$HOOKS_SRC_DIR/pre-commit" ]; then
    cp "$HOOKS_SRC_DIR/pre-commit" "$HOOKS_DIR/pre-commit"
    chmod +x "$HOOKS_DIR/pre-commit"
    echo "  ✓ pre-commit  → 快速检查 (fmt + vet + tidy)，~2秒"
else
    echo "  ✗ pre-commit hook not found in $HOOKS_SRC_DIR"
    exit 1
fi

# Copy pre-push hook (full CI checks - runs on git push)
if [ -f "$HOOKS_SRC_DIR/pre-push" ]; then
    cp "$HOOKS_SRC_DIR/pre-push" "$HOOKS_DIR/pre-push"
    chmod +x "$HOOKS_DIR/pre-push"
    echo "  ✓ pre-push   → 完整 CI 检查 (lint + test)，~1分钟"
else
    echo "  ✗ pre-push hook not found in $HOOKS_SRC_DIR"
    exit 1
fi

# Copy commit-msg hook if exists (validates commit message format)
if [ -f "$HOOKS_SRC_DIR/commit-msg" ]; then
    cp "$HOOKS_SRC_DIR/commit-msg" "$HOOKS_DIR/commit-msg"
    chmod +x "$HOOKS_DIR/commit-msg"
    echo "  ✓ commit-msg → 提交信息格式验证"
fi

echo ""
echo "✅ Git hooks installed successfully!"
echo ""
echo "检查时机:"
echo "  • pre-commit  → 每次 commit 时"
echo "  • pre-push     → 每次 push 到远程时"
echo "  • commit-msg   → 每次提交信息时"
echo ""
echo "跳过检查:"
echo "  • commit:  git commit --no-verify -m 'msg'"
echo "  • push:   git push --no-verify"
echo ""
echo "更多信息请参考: .claude/rules/git-workflow.md"
