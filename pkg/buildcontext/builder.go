// Package buildcontext handles diff and context building for Claude analysis
package buildcontext

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/errors"
)

// validGitRefPattern matches safe git refs (branch names, tags, commits)
var validGitRefPattern = regexp.MustCompile(`^[a-zA-Z0-9/_\-\.]+$`)

// dangerousShellChars contains characters that must be rejected to prevent shell injection
var dangerousShellChars = []string{"|", "&", ";", "$", "(", ")", "`", "{", "}", ">", "<", "\n", "\t"}

// sanitizeGitRef validates that a git ref is safe to use in commands
func sanitizeGitRef(ref string) error {
	if ref == "" {
		return nil // Empty ref is valid (defaults to HEAD)
	}
	// Check for path traversal attempts
	if strings.Contains(ref, "..") || strings.Contains(ref, "\\") {
		return fmt.Errorf("invalid git ref: contains path traversal sequence")
	}
	// Check for shell metacharacters
	for _, ch := range dangerousShellChars {
		if strings.Contains(ref, ch) {
			return fmt.Errorf("invalid git ref: contains dangerous character '%s'", ch)
		}
	}
	// Check against safe pattern
	if !validGitRefPattern.MatchString(ref) {
		return fmt.Errorf("invalid git ref: contains invalid characters")
	}
	return nil
}

// sanitizePath validates that a file path is safe
func sanitizePath(path string) error {
	if path == "" {
		return nil // Empty path is valid
	}
	// Check for path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: contains path traversal")
	}
	// Check for absolute paths (only relative paths allowed)
	if filepath.IsAbs(path) {
		return fmt.Errorf("invalid path: absolute paths not allowed")
	}
	// Check for shell metacharacters
	for _, ch := range dangerousShellChars {
		if strings.Contains(path, ch) {
			return fmt.Errorf("invalid path: contains dangerous character '%s'", ch)
		}
	}
	return nil
}

// Builder builds context for Claude Code analysis
type Builder struct {
	baseDir     string
	diffContext int
	exclude     []string
}

// NewBuilder creates a new context builder
func NewBuilder(baseDir string, diffContext int, exclude []string) *Builder {
	return &Builder{
		baseDir:     baseDir,
		diffContext: diffContext,
		exclude:     exclude,
	}
}

// BuildDiff builds the git diff for the current changes
func (b *Builder) BuildDiff(ctx context.Context, opts DiffOptions) (string, error) {
	// Validate inputs to prevent command injection
	if err := sanitizeGitRef(opts.TargetRef); err != nil {
		return "", fmt.Errorf("invalid target ref: %w", err)
	}
	if err := sanitizeGitRef(opts.SourceRef); err != nil {
		return "", fmt.Errorf("invalid source ref: %w", err)
	}
	if err := sanitizePath(opts.Path); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	args := b.buildDiffArgs(opts)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = b.baseDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if context was cancelled
		if ctx.Err() != nil {
			return "", fmt.Errorf("git diff cancelled: %w", ctx.Err())
		}
		return "", errors.ClaudeError(fmt.Sprintf("git diff failed: %s", stderr.String()), err)
	}

	return stdout.String(), nil
}

// buildDiffArgs constructs git diff arguments
func (b *Builder) buildDiffArgs(opts DiffOptions) []string {
	args := []string{"diff", "--no-color"}

	// Add context lines
	if b.diffContext > 0 {
		args = append(args, fmt.Sprintf("-U%d", b.diffContext))
	}

	// Add source and target refs
	if opts.TargetRef != "" {
		args = append(args, opts.TargetRef)
	}
	if opts.SourceRef != "" {
		args = append(args, opts.SourceRef)
	}

	// Add path filter
	if opts.Path != "" {
		args = append(args, "--", opts.Path)
	}

	// Exclude patterns
	for _, excl := range b.exclude {
		args = append(args, ":(exclude)"+excl)
	}

	return args
}

// BuildFileTree builds a tree view of the repository
func (b *Builder) BuildFileTree(ctx context.Context, maxDepth int) (string, error) {
	// Use git ls-tree to get the file structure
	cmd := exec.CommandContext(ctx, "git", "ls-tree", "-r", "--name-only", "HEAD")
	cmd.Dir = b.baseDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", errors.ClaudeError(fmt.Sprintf("git ls-tree failed: %s", stderr.String()), err)
	}

	// Process the output into a tree structure
	lines := strings.Split(stdout.String(), "\n")
	tree := b.buildTreeStructure(lines, maxDepth)

	return tree, nil
}

// buildTreeStructure builds a visual tree from file paths
func (b *Builder) buildTreeStructure(files []string, maxDepth int) string {
	if len(files) == 0 {
		return ""
	}

	root := &treeNode{name: "."}

	for _, file := range files {
		if file == "" {
			continue
		}
		parts := strings.Split(file, "/")
		current := root

		for i, part := range parts {
			if maxDepth > 0 && i >= maxDepth {
				break
			}

			found := false
			for _, child := range current.children {
				if child.name == part {
					current = child
					found = true
					break
				}
			}

			if !found {
				newNode := &treeNode{name: part}
				current.children = append(current.children, newNode)
				current = newNode
			}
		}
	}

	return root.String()
}

// treeNode represents a node in the file tree
type treeNode struct {
	name     string
	children []*treeNode
}

// String returns the tree as a formatted string
func (n *treeNode) String() string {
	var buf strings.Builder
	n.writeTo(&buf, "", true)
	return buf.String()
}

func (n *treeNode) writeTo(buf *strings.Builder, prefix string, isLast bool) {
	connector := "├── "
	if prefix == "" {
		connector = ""
	} else if isLast {
		connector = "└── "
	}

	buf.WriteString(prefix + connector + n.name + "\n")

	for i, child := range n.children {
		newPrefix := prefix
		if prefix != "" {
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
		}

		child.writeTo(buf, newPrefix, i == len(n.children)-1)
	}
}

// GetChangedFiles returns a list of changed files
func (b *Builder) GetChangedFiles(ctx context.Context, opts DiffOptions) ([]string, error) {
	args := []string{"diff", "--name-only", "--no-color"}

	if opts.TargetRef != "" && opts.SourceRef != "" {
		args = append(args, opts.TargetRef, opts.SourceRef)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = b.baseDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, errors.ClaudeError(fmt.Sprintf("git diff --name-only failed: %s", stderr.String()), err)
	}

	lines := strings.Split(stdout.String(), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !b.shouldExclude(line) {
			files = append(files, line)
		}
	}

	return files, nil
}

// GetFileContent returns the content of a file at a specific ref
func (b *Builder) GetFileContent(ctx context.Context, path, ref string) (string, error) {
	// Validate inputs to prevent command injection
	if err := sanitizeGitRef(ref); err != nil {
		return "", fmt.Errorf("invalid ref: %w", err)
	}
	if err := sanitizePath(path); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// SECURITY: Construct blob ref with explicit validation
	// The blob ref format "ref:path" must be validated as a whole to prevent
	// injection attempts that might bypass individual sanitization
	blobRef := ref + ":" + path
	// Validate the constructed blob ref doesn't contain injection patterns
	if strings.Contains(blobRef, "..") || strings.Contains(blobRef, "\\") {
		return "", fmt.Errorf("invalid blob ref: contains path traversal")
	}
	// Check for shell metacharacters in the combined ref
	for _, ch := range dangerousShellChars {
		if strings.Contains(blobRef, ch) {
			return "", fmt.Errorf("invalid blob ref: contains dangerous character '%s'", ch)
		}
	}
	// Use --end-of-options to ensure git treats the blob ref as a revision, not an option
	args := []string{"--no-pager", "show", "--end-of-options", blobRef}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = b.baseDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", errors.ClaudeError(fmt.Sprintf("git show failed: %s", stderr.String()), err)
	}

	return stdout.String(), nil
}

// shouldExclude checks if a path should be excluded
func (b *Builder) shouldExclude(path string) bool {
	for _, pattern := range b.exclude {
		// First, check for safe exact path or prefix matches
		// Use filepath.Match for proper glob pattern matching
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also check if path starts with pattern (for directory prefixes)
		if strings.HasPrefix(path, pattern+"/") {
			return true
		}
		// Check exact match
		if path == pattern {
			return true
		}
		// For backward compatibility, also check if any path component matches
		// This allows "lock" to match "package-lock.json" but is still safe
		// because we only match against path components, not arbitrary substrings
		pathParts := strings.Split(path, "/")
		for _, part := range pathParts {
			// Exact match or extension match
			if part == pattern || part == pattern+".json" || part == pattern+".lock" {
				return true
			}
			// Substring match for simple patterns without path separators
			// NOTE: This allows patterns like "lock" to match "package-lock.json"
			// Users should be aware that short patterns may have broad matches
			if strings.Contains(part, pattern) && !strings.Contains(pattern, "/") && !strings.Contains(pattern, "\\") {
				return true
			}
		}
	}
	return false
}

// GetCommitInfo returns information about the current commit
func (b *Builder) GetCommitInfo(ctx context.Context) (*CommitInfo, error) {
	// Get current branch
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = b.baseDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, errors.ClaudeError("failed to get branch", err)
	}

	branch := strings.TrimSpace(stdout.String())

	// Get current SHA
	cmd = exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = b.baseDir
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, errors.ClaudeError("failed to get SHA", err)
	}

	sha := strings.TrimSpace(stdout.String())

	// Get commit message
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--pretty=%B")
	cmd.Dir = b.baseDir
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, errors.ClaudeError("failed to get commit message", err)
	}

	message := strings.TrimSpace(stdout.String())

	// Get author
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--pretty=%an <%ae>")
	cmd.Dir = b.baseDir
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, errors.ClaudeError("failed to get author", err)
	}

	author := strings.TrimSpace(stdout.String())

	// Get timestamp
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--pretty=%ct")
	cmd.Dir = b.baseDir
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, errors.ClaudeError("failed to get timestamp", err)
	}

	var timestamp int64
	n, err := fmt.Sscanf(stdout.String(), "%d", &timestamp)
	if err != nil || n != 1 {
		// If timestamp parsing fails, use current time as fallback
		timestamp = time.Now().Unix()
	}

	return &CommitInfo{
		SHA:      sha,
		Branch:   branch,
		Message:  message,
		Author:   author,
		Time:     time.Unix(timestamp, 0),
		ChangedFiles: []string{},
	}, nil
}

// GetStats returns diff statistics
func (b *Builder) GetStats(ctx context.Context, opts DiffOptions) (*DiffStats, error) {
	args := []string{"diff", "--numstat", "--no-color"}

	if opts.TargetRef != "" && opts.SourceRef != "" {
		args = append(args, opts.TargetRef, opts.SourceRef)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = b.baseDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Diff with no changes is not an error
		if stderr.String() == "" {
			return &DiffStats{}, nil
		}
		return nil, errors.ClaudeError(fmt.Sprintf("git diff --numstat failed: %s", stderr.String()), err)
	}

	lines := strings.Split(stdout.String(), "\n")
	stats := &DiffStats{
		Files:    make(map[string]*FileStats),
		Additions: 0,
		Deletions: 0,
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		var additions, deletions int
		if _, err := fmt.Sscanf(parts[0], "%d", &additions); err != nil {
			continue // Skip malformed line
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &deletions); err != nil {
			continue // Skip malformed line
		}
		file := parts[2]

		stats.Files[file] = &FileStats{
			Additions: additions,
			Deletions: deletions,
		}
		stats.Additions += additions
		stats.Deletions += deletions
	}

	return stats, nil
}

// IsGitRepo checks if the base directory is a git repository
func (b *Builder) IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = b.baseDir
	return cmd.Run() == nil
}

// DiffOptions contains options for diff generation
type DiffOptions struct {
	TargetRef string // Base ref (e.g., "main", "HEAD~1")
	SourceRef string // Source ref (e.g., "HEAD", "feature-branch")
	Path      string // Optional path filter
}

// CommitInfo contains information about a commit
type CommitInfo struct {
	SHA          string
	Branch       string
	Message      string
	Author       string
	Time         time.Time
	ChangedFiles []string
}

// DiffStats contains diff statistics
type DiffStats struct {
	Files      map[string]*FileStats
	Additions  int
	Deletions  int
}

// FileStats contains stats for a single file
type FileStats struct {
	Additions int
	Deletions int
}

// Chunks breaks a large diff into smaller chunks for processing
func (b *Builder) Chunks(diff string, maxChunkSize int) []string {
	lines := strings.Split(diff, "\n")

	var chunks []string
	var currentChunk strings.Builder
	currentSize := 0

	for _, line := range lines {
		lineSize := len(line) + 1 // +1 for newline

		if currentSize+lineSize > maxChunkSize && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentSize = 0
		}

		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")
		currentSize += lineSize
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// FilterFilesByExtension filters files by their extensions
func FilterFilesByExtension(files []string, extensions map[string]bool) []string {
	var filtered []string
	for _, file := range files {
		ext := strings.TrimPrefix(filepath.Ext(file), ".")
		if extensions[ext] {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// GetLanguageFromPath determines the primary language from file paths
func GetLanguageFromPath(files []string) string {
	langCounts := make(map[string]int)

	for _, file := range files {
		ext := strings.TrimPrefix(filepath.Ext(file), ".")
		if ext != "" {
			langCounts[ext]++
		}
	}

	maxCount := 0
	lang := ""
	for l, count := range langCounts {
		if count > maxCount {
			maxCount = count
			lang = l
		}
	}

	return lang
}
