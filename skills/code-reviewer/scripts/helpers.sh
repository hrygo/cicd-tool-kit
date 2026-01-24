#!/usr/bin/env bash
# Copyright 2026 CICD AI Toolkit. All rights reserved.
#
# Helper scripts for code-reviewer skill

# Get files changed in the current branch
get_changed_files() {
    git diff --name-only origin/main...HEAD
}

# Get diff for a specific file
get_file_diff() {
    local file="$1"
    git diff origin/main...HEAD -- "$file"
}

# Get commit messages for the PR
get_commit_messages() {
    git log origin/main..HEAD --pretty=format:"%s%n%b"
}

# Main entry point
case "${1:-}" in
    files)
        get_changed_files
        ;;
    diff)
        get_file_diff "$2"
        ;;
    commits)
        get_commit_messages
        ;;
    *)
        echo "Usage: $0 {files|diff|commits}" >&2
        exit 1
        ;;
esac
