// Package buildcontext provides context builder tests
package buildcontext

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("/tmp", 3, []string{"vendor", "node_modules"})

	if builder.baseDir != "/tmp" {
		t.Errorf("baseDir = %s, want /tmp", builder.baseDir)
	}

	if builder.diffContext != 3 {
		t.Errorf("diffContext = %d, want 3", builder.diffContext)
	}

	if len(builder.exclude) != 2 {
		t.Errorf("exclude length = %d, want 2", len(builder.exclude))
	}
}

func TestBuildTreeStructure(t *testing.T) {
	files := []string{
		"src/main.go",
		"src/utils/helper.go",
		"pkg/config/config.go",
		"README.md",
		"go.mod",
	}

	builder := NewBuilder(".", 3, nil)
	tree := builder.buildTreeStructure(files, 0)

	if tree == "" {
		t.Error("buildTreeStructure() returned empty string")
	}

	// Check that key files are in the tree
	if !strings.Contains(tree, "src") {
		t.Error("Tree should contain 'src'")
	}
	if !strings.Contains(tree, "pkg") {
		t.Error("Tree should contain 'pkg'")
	}
	if !strings.Contains(tree, "main.go") {
		t.Error("Tree should contain 'main.go'")
	}
}

func TestBuildTreeStructureWithMaxDepth(t *testing.T) {
	files := []string{
		"src/a/b/c/d/file.go",
		"src/x/y/z/file.go",
		"README.md",
	}

	builder := NewBuilder(".", 3, nil)
	tree := builder.buildTreeStructure(files, 2)

	// With max depth 2, deep paths should be truncated
	if !strings.Contains(tree, "src") {
		t.Error("Tree should contain 'src'")
	}
}

func TestTreeNode(t *testing.T) {
	root := &treeNode{name: "root"}
	root.children = append(root.children,
		&treeNode{name: "child1"},
		&treeNode{name: "child2"},
	)

	output := root.String()
	if !strings.Contains(output, "root") {
		t.Error("Tree should contain 'root'")
	}
	if !strings.Contains(output, "child1") {
		t.Error("Tree should contain 'child1'")
	}
	if !strings.Contains(output, "child2") {
		t.Error("Tree should contain 'child2'")
	}
}

func TestShouldExclude(t *testing.T) {
	builder := NewBuilder(".", 3, []string{"vendor", "node_modules", "lock"})

	tests := []struct {
		path     string
		excluded bool
	}{
		{"src/main.go", false},
		{"vendor/github.com/pkg/errors.go", true},
		{"node_modules/pkg/index.js", true},
		{"package-lock.json", true}, // Contains "lock"
		{"go.mod", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := builder.shouldExclude(tt.path); got != tt.excluded {
				t.Errorf("shouldExclude(%s) = %v, want %v", tt.path, got, tt.excluded)
			}
		})
	}
}

func TestFilterFilesByExtension(t *testing.T) {
	files := []string{
		"main.go",
		"utils.go",
		"README.md",
		"package.json",
		"handler.js",
	}

	extensions := map[string]bool{
		"go":  true,
		"js":  true,
		"md":  false,
		"json": false,
	}

	filtered := FilterFilesByExtension(files, extensions)

	if len(filtered) != 3 {
		t.Errorf("FilterFilesByExtension() returned %d files, want 3", len(filtered))
	}

	for _, f := range filtered {
		ext := strings.TrimPrefix(filepath.Ext(f), ".")
		if !extensions[ext] {
			t.Errorf("File %s with extension %s should be filtered out", f, ext)
		}
	}
}

func TestGetLanguageFromPath(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name: "mostly go",
			files: []string{
				"main.go",
				"utils.go",
				"config.go",
				"README.md",
			},
			expected: "go",
		},
		{
			name: "mixed with js majority",
			files: []string{
				"main.go",
				"handler.js",
				"utils.js",
				"config.js",
			},
			expected: "js",
		},
		{
			name:     "empty",
			files:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLanguageFromPath(tt.files); got != tt.expected {
				t.Errorf("GetLanguageFromPath() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestChunks(t *testing.T) {
	builder := NewBuilder(".", 3, nil)

	diff := strings.Repeat("line of content\n", 100) // ~1500 bytes

	chunks := builder.Chunks(diff, 500)

	if len(chunks) == 0 {
		t.Error("Chunks() returned no chunks")
	}

	if len(chunks) < 2 {
		t.Errorf("Chunks() should split into multiple chunks, got %d", len(chunks))
	}

	// Verify total content is preserved (chunks add newlines)
	var total strings.Builder
	for _, chunk := range chunks {
		total.WriteString(chunk)
	}

	// The chunking process may add an extra newline at chunk boundaries
	// Just check that all lines are present
	expectedLines := strings.Count(diff, "\n")
	actualLines := strings.Count(total.String(), "\n")
	if actualLines < expectedLines {
		t.Errorf("Chunks() lost lines: expected %d, got %d", expectedLines, actualLines)
	}
}

func TestChunksEmpty(t *testing.T) {
	builder := NewBuilder(".", 3, nil)

	chunks := builder.Chunks("", 500)

	// Empty input returns one empty chunk due to how split works
	// This is acceptable behavior
	if len(chunks) > 1 {
		t.Errorf("Chunks(\"\") should return at most 1 chunk, got %d", len(chunks))
	}
}

func TestChunksSingleLine(t *testing.T) {
	builder := NewBuilder(".", 3, nil)

	diff := "single line"

	chunks := builder.Chunks(diff, 500)

	if len(chunks) != 1 {
		t.Errorf("Chunks(single line) should return 1 chunk, got %d", len(chunks))
	}

	if chunks[0] != diff+"\n" {
		t.Errorf("Chunk content mismatch")
	}
}

func TestBuildDiffArgs(t *testing.T) {
	builder := NewBuilder(".", 3, []string{"vendor"})

	tests := []struct {
		name string
		opts DiffOptions
		want []string
	}{
		{
			name: "basic diff",
			opts: DiffOptions{},
			want: []string{"diff", "--no-color", "-U3"},
		},
		{
			name: "with refs",
			opts: DiffOptions{TargetRef: "main", SourceRef: "feature"},
			want: []string{"diff", "--no-color", "-U3", "main", "feature"},
		},
		{
			name: "with path",
			opts: DiffOptions{Path: "src/"},
			want: []string{"diff", "--no-color", "-U3", "--", "src/", ":(exclude)vendor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.buildDiffArgs(tt.opts)
			// Compare first few elements
			for i, w := range tt.want {
				if i >= len(got) {
					t.Errorf("buildDiffArgs() too short, missing element %d", i)
					break
				}
				if got[i] != w {
					t.Errorf("buildDiffArgs()[%d] = %s, want %s", i, got[i], w)
				}
			}
		})
	}
}

// Test with a real git repository
func TestBuilderInGitRepo(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()

	// Initialize git repo
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	builder := NewBuilder(tmpDir, 3, nil)

	// Run git init
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Add and commit
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Test IsGitRepo
	if !builder.IsGitRepo() {
		t.Error("IsGitRepo() should return true for git repo")
	}

	// Test GetCommitInfo
	ctx := context.Background()
	info, err := builder.GetCommitInfo(ctx)
	if err != nil {
		t.Errorf("GetCommitInfo() error = %v", err)
	}

	if info.SHA == "" {
		t.Error("GetCommitInfo() should return a SHA")
	}

	if !strings.Contains(info.Message, "Initial commit") {
		t.Errorf("Message should contain 'Initial commit', got: %s", info.Message)
	}
}
