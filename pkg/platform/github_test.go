// Package platform provides GitHub platform tests
package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewGitHubClient(t *testing.T) {
	client := NewGitHubClient("test-token", "owner/repo")

	if client.Name() != "github" {
		t.Errorf("Name() = %s, want github", client.Name())
	}

	if client.token != "test-token" {
		t.Errorf("token = %s, want test-token", client.token)
	}

	if client.repo != "owner/repo" {
		t.Errorf("repo = %s, want owner/repo", client.repo)
	}
}

func TestGitHubClient_SetBaseURL(t *testing.T) {
	client := NewGitHubClient("token", "owner/repo")
	client.SetBaseURL("https://github.enterprise.com/api/v3")

	if client.baseURL != "https://github.enterprise.com/api/v3" {
		t.Errorf("baseURL = %s, want https://github.enterprise.com/api/v3", client.baseURL)
	}
}

func TestGitHubClient_PostComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Method = %s, want POST", r.Method)
		}

		// Check auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %s, want Bearer test-token", auth)
		}

		// Check content type
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", ct)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	client := NewGitHubClient("test-token", "owner/repo")
	client.baseURL = server.URL

	ctx := context.Background()
	err := client.PostComment(ctx, CommentOptions{
		PRID: 123,
		Body: "Test comment",
	})

	if err != nil {
		t.Errorf("PostComment() error = %v", err)
	}
}

func TestGitHubClient_PostCommentAsReview(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First request is get PR
		if r.URL.Path == "/repos/owner/repo/pulls/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"number": 123,
				"title": "Test PR",
				"head": {"sha": "abc123", "ref": "feature"}
			}`))
			return
		}

		// Second request is post review
		if r.URL.Path == "/repos/owner/repo/pulls/123/reviews" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 1}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewGitHubClient("test-token", "owner/repo")
	client.baseURL = server.URL

	ctx := context.Background()
	err := client.PostComment(ctx, CommentOptions{
		PRID:     123,
		Body:     "Review comment",
		AsReview: true,
	})

	if err != nil {
		t.Errorf("PostComment(AsReview=true) error = %v", err)
	}
}

func TestGitHubClient_GetDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/pulls/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"number": 123,
				"head": {"sha": "abc123"}
			}`))
			return
		}

		if r.URL.Path == "/repos/owner/repo/pulls/123.diff" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("diff --git a/file.go b/file.go\n+new line"))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewGitHubClient("test-token", "owner/repo")
	client.baseURL = server.URL

	ctx := context.Background()
	diff, err := client.GetDiff(ctx, 123)

	if err != nil {
		t.Errorf("GetDiff() error = %v", err)
	}

	if diff == "" {
		t.Error("GetDiff() returned empty diff")
	}
}

func TestGitHubClient_GetPRInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"number": 123,
			"title": "Test PR",
			"body": "Test description",
			"user": {"login": "testuser"},
			"head": {
				"sha": "abc123",
				"ref": "feature",
				"repo": {"full_name": "owner/repo"}
			},
			"base": {"ref": "main"}
		}`))
	}))
	defer server.Close()

	client := NewGitHubClient("test-token", "owner/repo")
	client.baseURL = server.URL

	ctx := context.Background()
	info, err := client.GetPRInfo(ctx, 123)

	if err != nil {
		t.Errorf("GetPRInfo() error = %v", err)
	}

	if info.Number != 123 {
		t.Errorf("Number = %d, want 123", info.Number)
	}

	if info.Title != "Test PR" {
		t.Errorf("Title = %s, want Test PR", info.Title)
	}

	if info.Author != "testuser" {
		t.Errorf("Author = %s, want testuser", info.Author)
	}

	if info.SHA != "abc123" {
		t.Errorf("SHA = %s, want abc123", info.SHA)
	}
}

func TestGitHubClient_Health(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewGitHubClient("test-token", "owner/repo")
	client.baseURL = server.URL

	ctx := context.Background()
	err := client.Health(ctx)

	if err != nil {
		t.Errorf("Health() error = %v", err)
	}
}

func TestGitHubClient_HealthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewGitHubClient("test-token", "owner/repo")
	client.baseURL = server.URL

	ctx := context.Background()
	err := client.Health(ctx)

	if err == nil {
		t.Error("Health() expected error for unhealthy service, got nil")
	}
}

func TestParseRepoFromEnv(t *testing.T) {
	// Save original function
	origOsGetenv := osGetenv
	defer func() { osGetenv = origOsGetenv }()

	// Test success
	osGetenv = func(key string) string {
		if key == "GITHUB_REPOSITORY" {
			return "owner/repo"
		}
		return ""
	}

	repo, err := ParseRepoFromEnv()
	if err != nil {
		t.Errorf("ParseRepoFromEnv() error = %v", err)
	}

	if repo != "owner/repo" {
		t.Errorf("repo = %s, want owner/repo", repo)
	}

	// Test failure
	osGetenv = func(key string) string {
		return ""
	}

	_, err = ParseRepoFromEnv()
	if err == nil {
		t.Error("ParseRepoFromEnv() expected error for missing env var, got nil")
	}
}

func TestParsePRIDFromEnv(t *testing.T) {
	// Save original function
	origOsGetenv := osGetenv
	defer func() { osGetenv = origOsGetenv }()

	// Test success
	osGetenv = func(key string) string {
		if key == "GITHUB_PR_NUMBER" {
			return "123"
		}
		return ""
	}

	prID, err := ParsePRIDFromEnv()
	if err != nil {
		t.Errorf("ParsePRIDFromEnv() error = %v", err)
	}

	if prID != 123 {
		t.Errorf("prID = %d, want 123", prID)
	}

	// Test failure - not a number
	osGetenv = func(key string) string {
		return "abc"
	}

	_, err = ParsePRIDFromEnv()
	if err == nil {
		t.Error("ParsePRIDFromEnv() expected error for invalid number, got nil")
	}
}

func TestIsGitHubEnv(t *testing.T) {
	// Save original function
	origOsGetenv := osGetenv
	defer func() { osGetenv = origOsGetenv }()

	// Test in GitHub env
	osGetenv = func(key string) string {
		if key == "GITHUB_ACTIONS" {
			return "true"
		}
		return ""
	}

	if !IsGitHubEnv() {
		t.Error("IsGitHubEnv() = false, want true")
	}

	// Test not in GitHub env
	osGetenv = os.Getenv

	if IsGitHubEnv() {
		t.Error("IsGitHubEnv() = true, want false (not in GitHub Actions)")
	}
}

// Test that Position comments work
func TestGitHubClient_PostCommentWithPosition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get PR first
		if r.URL.Path == "/repos/owner/repo/pulls/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"number": 123,
				"head": {"sha": "abc123"}
			}`))
			return
		}

		// Post review
		if r.URL.Path == "/repos/owner/repo/pulls/123/reviews" {
			// Verify body contains position
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 1}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewGitHubClient("test-token", "owner/repo")
	client.baseURL = server.URL

	ctx := context.Background()
	line := 42
	err := client.PostComment(ctx, CommentOptions{
		PRID:     123,
		Body:     "Line-specific comment",
		AsReview: true,
		Position: &Position{
			Path: "path/to/file.go",
			Line: line,
		},
	})

	if err != nil {
		t.Errorf("PostComment(Position) error = %v", err)
	}
}

// Benchmark for GetDiff
func BenchmarkGitHubClient_GetDiff(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/pulls/123" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"head": {"sha": "abc123"}}`))
			return
		}
		if r.URL.Path == "/repos/owner/repo/pulls/123.diff" {
			// Simulate a large diff
			w.WriteHeader(http.StatusOK)
			for i := 0; i < 1000; i++ {
				w.Write([]byte("+line " + string(rune(i)) + "\n"))
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewGitHubClient("token", "owner/repo")
	client.baseURL = server.URL
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetDiff(ctx, 123)
	}
}
