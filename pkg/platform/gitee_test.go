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
