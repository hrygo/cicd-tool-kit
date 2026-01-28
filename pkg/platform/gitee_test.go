// Package platform tests for Gitee client
package platform

import (
	"context"
	"testing"
)

func TestNewGiteeClient(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	if client == nil {
		t.Fatal("NewGiteeClient returned nil")
	}

	if client.token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", client.token)
	}

	if client.repo != "owner/repo" {
		t.Errorf("Expected repo 'owner/repo', got '%s'", client.repo)
	}

	if client.baseURL != "https://gitee.com/api/v5" {
		t.Errorf("Expected default baseURL 'https://gitee.com/api/v5', got '%s'", client.baseURL)
	}
}

func TestGiteeClientSetBaseURL(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")
	customURL := "https://gitee.enterprise.com/api/v5"

	client.SetBaseURL(customURL)

	if client.baseURL != customURL {
		t.Errorf("Expected baseURL '%s', got '%s'", customURL, client.baseURL)
	}
}

func TestIsGiteeEnv(t *testing.T) {
	// Test without environment variable
	if IsGiteeEnv() {
		t.Error("IsGiteeEnv should return false when GITEE_API_URL is not set")
	}
}

func TestParseRepoFromGiteeEnvErrors(t *testing.T) {
	// Test without environment variables set
	_, err := ParseRepoFromGiteeEnv()
	if err == nil {
		t.Error("Expected error when Gitee env vars are not set")
	}
}

func TestParsePRIDFromGiteeEnvErrors(t *testing.T) {
	// Test without environment variables set
	_, err := ParsePRIDFromGiteeEnv()
	if err == nil {
		t.Error("Expected error when Gitee env vars are not set")
	}
}

func TestGiteePostCommentValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name    string
		opts    CommentOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options (network error expected)",
			opts: CommentOptions{
				PRID: 123,
				Body: "Test comment",
			},
			wantErr: true, // Will fail on network/auth
			errMsg:  "",
		},
		{
			name: "missing PR ID",
			opts: CommentOptions{
				Body: "Test comment",
			},
			wantErr: true,
			errMsg:  "PR ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.PostComment(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("PostComment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}

func TestGiteeHealthCheck(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	// Health check will fail without valid credentials/network
	// but we can test the method exists and runs
	err := client.Health(context.Background())
	// We expect this to fail in test environment
	if err == nil {
		t.Log("Health check passed (unexpected in test environment)")
	}
}

// TestGiteeClientName tests the Name() method
func TestGiteeClientName(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	name := client.Name()
	if name != "gitee" {
		t.Errorf("Expected Name() to return 'gitee', got '%s'", name)
	}
}

// TestValidatePath tests path validation
func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantErr   bool
		errContains string
	}{
		{
			name:      "valid relative path",
			path:      "pkg/main.go",
			wantErr:   false,
		},
		{
			name:      "valid nested path",
			path:      "pkg/platform/client.go",
			wantErr:   false,
		},
		{
			name:        "path traversal with ..",
			path:        "../etc/passwd",
			wantErr:     true,
			errContains: "traversal sequence",
		},
		{
			name:        "path traversal with encoded",
			path:        "%2e%2e%2fpasswd",
			wantErr:     true,
			errContains: "traversal sequence",
		},
		{
			name:        "backslash in path",
			path:        "pkg\\main.go",
			wantErr:     true,
			errContains: "backslash",
		},
		{
			name:        "absolute path",
			path:        "/etc/passwd",
			wantErr:     true,
			errContains: "absolute paths not allowed",
		},
		{
			name:      "empty path",
			path:      "",
			wantErr:   true,
			errContains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
			}
		})
	}
}

// TestGetFileValidation tests GetFile with path validation
func TestGetFileValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name        string
		path        string
		ref         string
		wantErr     bool
		errContains string
	}{
		{
			name:        "path with ..",
			path:        "../../../etc/passwd",
			ref:         "main",
			wantErr:     true,
			errContains: "traversal sequence",
		},
		{
			name:        "absolute path",
			path:        "/etc/passwd",
			ref:         "main",
			wantErr:     true,
			errContains: "absolute paths not allowed",
		},
		{
			name:    "valid path (will fail on network)",
			path:    "README.md",
			ref:     "main",
			wantErr: true, // Network error in test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetFile(context.Background(), tt.path, tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
			}
		})
	}
}

// TestPostCommentMissingPRID tests PostComment validation
func TestPostCommentMissingPRID(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	err := client.PostComment(context.Background(), CommentOptions{
		Body: "Test comment",
	})

	if err == nil {
		t.Error("Expected error when PR ID is missing")
	}

	if !contains(err.Error(), "PR ID is required") {
		t.Errorf("Expected error message to contain 'PR ID is required', got '%s'", err.Error())
	}
}
