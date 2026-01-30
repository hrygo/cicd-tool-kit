// Package runner provides runner implementation tests
package runner

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/ai"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/config"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/platform"
)

// MockPlatform for testing
type mockPlatform struct{}

func (m *mockPlatform) Name() string { return "mock" }
func (m *mockPlatform) PostComment(ctx context.Context, opts platform.CommentOptions) error {
	return nil
}
func (m *mockPlatform) GetDiff(ctx context.Context, prID int) (string, error) {
	return "diff content", nil
}
func (m *mockPlatform) GetFile(ctx context.Context, path, ref string) (string, error) {
	return "file content", nil
}
func (m *mockPlatform) GetPRInfo(ctx context.Context, prID int) (*platform.PRInfo, error) {
	return &platform.PRInfo{Number: prID}, nil
}
func (m *mockPlatform) Health(ctx context.Context) error { return nil }

func TestNewRunner(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{
			CacheDir:    ".cache",
			EnableCache: false,
			DiffContext: 3,
		},
		Claude: config.ClaudeConfig{
			Model:        "sonnet",
			MaxTurns:     50,
			Timeout:      "30m",
			MaxBudgetUSD: 10.0,
		},
		Skills: []config.SkillConfig{
			{Name: "test-skill", Path: "./skills/test", Enabled: true},
		},
	}

	runner, err := NewRunner(cfg, &mockPlatform{}, ".")
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if runner == nil {
		t.Error("NewRunner() returned nil runner")
	}
}

func TestSummarizeIssues(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{CacheDir: ".cache", EnableCache: false},
		Claude: config.ClaudeConfig{},
	}
	runner, _ := NewRunner(cfg, &mockPlatform{}, ".")

	issues := []ai.Issue{
		{Severity: "critical", Category: "security", File: "auth.go", Line: 10},
		{Severity: "high", Category: "performance", File: "cache.go", Line: 25},
		{Severity: "medium", Category: "style", File: "utils.go", Line: 5},
		{Severity: "low", Category: "style", File: "utils.go", Line: 6},
		{Severity: "low", Category: "style", File: "main.go", Line: 1},
	}

	summary := runner.summarizeIssues(issues)

	if summary.TotalIssues != 5 {
		t.Errorf("TotalIssues = %d, want 5", summary.TotalIssues)
	}

	if summary.Critical != 1 {
		t.Errorf("Critical = %d, want 1", summary.Critical)
	}

	if summary.High != 1 {
		t.Errorf("High = %d, want 1", summary.High)
	}

	if summary.Medium != 1 {
		t.Errorf("Medium = %d, want 1", summary.Medium)
	}

	if summary.Low != 2 {
		t.Errorf("Low = %d, want 2", summary.Low)
	}

	// FilesChanged counts unique files with issues
	// We have 4 unique files: auth.go, cache.go, utils.go, main.go
	if summary.FilesChanged != 4 {
		t.Errorf("FilesChanged = %d, want 4", summary.FilesChanged)
	}
}

func TestFormatReviewComment(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{CacheDir: ".cache", EnableCache: false},
		Claude: config.ClaudeConfig{},
	}
	runner, _ := NewRunner(cfg, &mockPlatform{}, ".")

	result := &ReviewResult{
		Summary: ReviewSummary{
			FilesChanged: 2,
			TotalIssues:  3,
			Critical:     1,
			High:         1,
			Medium:       1,
			Low:          0,
		},
		Issues: []ai.Issue{
			{
				Severity: "critical",
				Category: "security",
				File:     "auth.go",
				Line:     10,
				Message:  "SQL injection vulnerability",
			},
		},
	}

	comment := runner.formatReviewComment(result)

	if comment == "" {
		t.Error("formatReviewComment() returned empty string")
	}

	// Check key elements are present
	checks := []struct {
		substring string
		negative  bool
	}{
		{"Code Review Results", false},
		{"**Files Changed**: 2", false},
		{"**Total Issues**: 3", false},
		{"SQL injection vulnerability", false},
	}

	for _, check := range checks {
		contains := strings.Contains(comment, check.substring)
		if check.negative {
			if contains {
				t.Errorf("Comment should not contain: %s", check.substring)
			}
		} else {
			if !contains {
				t.Errorf("Comment should contain: %s", check.substring)
			}
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"handler.js", "javascript"},
		{"component.ts", "typescript"},
		{"utils.py", "python"},
		{"lib.rs", "rust"},
		{"App.java", "java"},
		{"unknown.xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := detectLanguage(tt.path); got != tt.expected {
				t.Errorf("detectLanguage(%s) = %s, want %s", tt.path, got, tt.expected)
			}
		})
	}
}

func TestSeverityIcon(t *testing.T) {
	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "🔴"},
		{"high", "🟠"},
		{"medium", "🟡"},
		{"low", "🟢"},
		{"unknown", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			if got := severityIcon(tt.severity); got != tt.expected {
				t.Errorf("severityIcon(%s) = %s, want %s", tt.severity, got, tt.expected)
			}
		})
	}
}

func TestCache(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewCache(tmpDir, true)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	// Test Set and Get
	review := CachedReview{
		Summary: ReviewSummary{TotalIssues: 5},
		Issues: []ai.Issue{
			{Severity: "critical", Message: "Test issue"},
		},
		Comment: "Test comment",
	}

	cache.SetReview(123, review)

	got, ok := cache.GetReview(123)
	if !ok {
		t.Fatal("GetReview() returned ok=false")
	}

	if got.Summary.TotalIssues != 5 {
		t.Errorf("TotalIssues = %d, want 5", got.Summary.TotalIssues)
	}

	// Test Invalidate
	cache.Invalidate(123)

	_, ok = cache.GetReview(123)
	if ok {
		t.Error("GetReview() should return ok=false after Invalidate")
	}
}

func TestCacheDisabled(t *testing.T) {
	cache, err := NewCache("", false)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	review := CachedReview{Comment: "test"}
	cache.SetReview(123, review)

	_, ok := cache.GetReview(123)
	if ok {
		t.Error("GetReview() should return ok=false when cache disabled")
	}
}

func TestCacheTTL(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewCache(tmpDir, true)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	// Set a very short TTL
	cache.SetTTL(1 * time.Nanosecond)

	review := CachedReview{Comment: "test"}
	cache.SetReview(123, review)

	// Wait for TTL to expire
	time.Sleep(10 * time.Millisecond)

	_, ok := cache.GetReview(123)
	if ok {
		t.Error("GetReview() should return ok=false after TTL expires")
	}
}

func TestGetDiffHash(t *testing.T) {
	diff1 := "some diff content"
	diff2 := "some diff content"
	diff3 := "different content"

	hash1 := GetDiffHash(diff1)
	hash2 := GetDiffHash(diff2)
	hash3 := GetDiffHash(diff3)

	if hash1 != hash2 {
		t.Error("Same diff should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("Different diffs should produce different hashes")
	}
}

func TestSummarizeTests(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{CacheDir: ".cache", EnableCache: false},
		Claude: config.ClaudeConfig{},
	}
	runner, _ := NewRunner(cfg, &mockPlatform{}, ".")

	tests := []GeneratedTest{
		{Path: "test_main.go", Tests: 5},
		{Path: "test_utils.go", Tests: 3},
		{Path: "test_auth.go", Tests: 7},
	}

	summary := runner.summarizeTests(tests)

	if summary.FilesCreated != 3 {
		t.Errorf("FilesCreated = %d, want 3", summary.FilesCreated)
	}

	if summary.TotalTests != 15 {
		t.Errorf("TotalTests = %d, want 15", summary.TotalTests)
	}

	if summary.CoverageEst == "" {
		t.Error("CoverageEst should not be empty when tests exist")
	}
}

func TestParseTestsFromOutput(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{CacheDir: ".cache", EnableCache: false},
		Claude: config.ClaudeConfig{},
	}
	runner, _ := NewRunner(cfg, &mockPlatform{}, ".")

	t.Run("empty output", func(t *testing.T) {
		tests := runner.parseTestsFromOutput("")
		if len(tests) != 0 {
			t.Errorf("Empty output should produce 0 tests, got %d", len(tests))
		}
	})

	t.Run("Go test code block", func(t *testing.T) {
		output := "Here are the tests:\n```go\nfunc TestAdd(t *testing.T) {\n}\n```"
		tests := runner.parseTestsFromOutput(output)
		if len(tests) != 1 {
			t.Fatalf("Expected 1 test, got %d", len(tests))
		}
		if tests[0].Language != "go" {
			t.Errorf("Language = %s, want go", tests[0].Language)
		}
		if tests[0].Path != "generated_test.go" {
			t.Errorf("Path = %s, want generated_test.go", tests[0].Path)
		}
		if !strings.Contains(tests[0].Content, "func TestAdd") {
			t.Error("Content should contain TestAdd function")
		}
	})

	t.Run("Python test code block", func(t *testing.T) {
		output := "```python\nimport pytest\n\ndef test_calculate():\n    pass\n```"
		tests := runner.parseTestsFromOutput(output)
		if len(tests) != 1 {
			t.Fatalf("Expected 1 test, got %d", len(tests))
		}
		if tests[0].Language != "python" {
			t.Errorf("Language = %s, want python", tests[0].Language)
		}
		if tests[0].Tests != 1 {
			t.Errorf("Tests = %d, want 1", tests[0].Tests)
		}
	})

	t.Run("JavaScript test code block", func(t *testing.T) {
		output := "```js\ndescribe('suite', () => {\n  it('should work', () => {})\n})\n```"
		tests := runner.parseTestsFromOutput(output)
		if len(tests) != 1 {
			t.Fatalf("Expected 1 test, got %d", len(tests))
		}
		if tests[0].Language != "js" {
			t.Errorf("Language = %s, want js", tests[0].Language)
		}
	})

	t.Run("multiple test blocks", func(t *testing.T) {
		output := "```go\nfunc TestOne(t *testing.T) {}\n```\n```py\ndef test_two():\n    pass\n```"
		tests := runner.parseTestsFromOutput(output)
		if len(tests) != 2 {
			t.Fatalf("Expected 2 tests, got %d", len(tests))
		}
	})

	t.Run("non-test code block is ignored", func(t *testing.T) {
		output := "```go\nfunc RegularFunction() {}\n```"
		tests := runner.parseTestsFromOutput(output)
		if len(tests) != 0 {
			t.Errorf("Non-test code should produce 0 tests, got %d", len(tests))
		}
	})

	t.Run("code block without language is ignored", func(t *testing.T) {
		output := "```\nfunc TestSomething(t *testing.T) {}\n```"
		tests := runner.parseTestsFromOutput(output)
		if len(tests) != 0 {
			t.Errorf("Code block without language should produce 0 tests, got %d", len(tests))
		}
	})

	t.Run("multiple test functions counted correctly", func(t *testing.T) {
		output := "```go\nfunc TestA(t *testing.T) {}\nfunc TestB(t *testing.T) {}\nfunc TestC(t *testing.T) {}\n```"
		tests := runner.parseTestsFromOutput(output)
		if len(tests) != 1 {
			t.Fatalf("Expected 1 test file, got %d", len(tests))
		}
		if tests[0].Tests != 3 {
			t.Errorf("Tests = %d, want 3", tests[0].Tests)
		}
	})
}

func TestIsTestContent(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{CacheDir: ".cache", EnableCache: false},
		Claude: config.ClaudeConfig{},
	}
	runner, _ := NewRunner(cfg, &mockPlatform{}, ".")

	testCases := []struct {
		content string
		want    bool
	}{
		{"func TestSomething(t *testing.T) {}", true},
		{"def test_something():", true},
		{"describe('test', () => {})", true},
		{"it('should work', () => {})", true},
		{"@Test void testMethod() {}", true},
		{"import pytest", true},
		{`import "testing"`, true},
		{"func RegularFunction() {}", false},
		{"just some text", false},
		{"class MyClass {}", false},
	}

	for _, tc := range testCases {
		t.Run(tc.content, func(t *testing.T) {
			got := runner.isTestContent(tc.content)
			if got != tc.want {
				t.Errorf("isTestContent(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestCountTestFunctions(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{CacheDir: ".cache", EnableCache: false},
		Claude: config.ClaudeConfig{},
	}
	runner, _ := NewRunner(cfg, &mockPlatform{}, ".")

	testCases := []struct {
		lang    string
		content string
		want    int
	}{
		{"go", "func TestA(t *testing.T) {}", 1},
		{"go", "func TestA(t *testing.T) {}\nfunc TestB(t *testing.T) {}", 2},
		{"python", "def test_a():\n    pass", 1},
		{"python", "def test_a():\ndef test_b():", 2},
		{"js", "describe('suite', () => { it('a', () => {}) })", 1},
		{"js", "it('a', () => {})\nit('b', () => {})", 2},
		{"java", "@Test\npublic void testA() {}", 1},
		{"unknown", "func TestA(t *testing.T) {}", 1}, // Fallback to isTestContent
	}

	for _, tc := range testCases {
		t.Run(tc.lang+"_"+string(rune(tc.want)), func(t *testing.T) {
			got := runner.countTestFunctions(tc.content, tc.lang)
			if got != tc.want {
				t.Errorf("countTestFunctions(%q, %s) = %d, want %d", tc.content, tc.lang, got, tc.want)
			}
		})
	}
}

func TestGenerateTestPath(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{CacheDir: ".cache", EnableCache: false},
		Claude: config.ClaudeConfig{},
	}
	runner, _ := NewRunner(cfg, &mockPlatform{}, ".")

	testCases := []struct {
		lang string
		want string
	}{
		{"go", "generated_test.go"},
		{"py", "generated_test.py"},
		{"python", "generated_test.py"},
		{"js", "generated.test.js"},
		{"ts", "generated.test.ts"},
		{"java", "GeneratedTest.java"},
		{"rs", "generated_test.rs"},
		{"rust", "generated_test.rs"},
		{"cpp", "generated_test.cpp"},
		{"c", "generated_test.c"},
		{"ruby", "generated_test.ruby"},
	}

	for _, tc := range testCases {
		t.Run(tc.lang, func(t *testing.T) {
			got := runner.generateTestPath(tc.lang)
			if got != tc.want {
				t.Errorf("generateTestPath(%s) = %s, want %s", tc.lang, got, tc.want)
			}
		})
	}
}
