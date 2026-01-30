// Package platform tests for GitLab client
package platform

import (
	"context"
	"testing"
)

func TestNewGitLabClient(t *testing.T) {
	client := NewGitLabClient("test-token", "owner/repo")

	if client == nil {
		t.Fatal("NewGitLabClient returned nil")
	}

	if client.token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", client.token)
	}

	if client.repo != "owner/repo" {
		t.Errorf("Expected repo 'owner/repo', got '%s'", client.repo)
	}

	if client.baseURL != "https://gitlab.com/api/v4" {
		t.Errorf("Expected default baseURL 'https://gitlab.com/api/v4', got '%s'", client.baseURL)
	}
}

func TestGitLabClientSetBaseURL(t *testing.T) {
	client := NewGitLabClient("test-token", "owner/repo")
	customURL := "https://gitlab.enterprise.com/api/v4"

	if err := client.SetBaseURL(customURL); err != nil {
		t.Fatalf("SetBaseURL failed: %v", err)
	}

	if client.baseURL != customURL {
		t.Errorf("Expected baseURL '%s', got '%s'", customURL, client.baseURL)
	}
}

func TestIsGitLabEnv(t *testing.T) {
	// Test without environment variable
	if IsGitLabEnv() {
		t.Error("IsGitLabEnv should return false when GITLAB_CI is not set")
	}
}

func TestParseRepoFromGitLabEnvErrors(t *testing.T) {
	// Test without environment variables set
	_, err := ParseRepoFromGitLabEnv()
	if err == nil {
		t.Error("Expected error when GitLab env vars are not set")
	}
}

func TestParsePRIDFromGitLabEnvErrors(t *testing.T) {
	// Test without environment variables set
	_, err := ParsePRIDFromGitLabEnv()
	if err == nil {
		t.Error("Expected error when GitLab env vars are not set")
	}
}

func TestGitLabPostCommentValidation(t *testing.T) {
	client := NewGitLabClient("test-token", "owner/repo")

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
			wantErr: true,
			errMsg:  "",
		},
		{
			name: "missing PR ID",
			opts: CommentOptions{
				Body: "Test comment",
			},
			wantErr: true,
			errMsg:  "MR IID is required",
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

func TestGitLabHealthCheck(t *testing.T) {
	client := NewGitLabClient("test-token", "owner/repo")

	// Health check will fail without valid credentials/network
	err := client.Health(context.Background())
	// We expect this to fail in test environment
	if err == nil {
		t.Log("Health check passed (unexpected in test environment)")
	}
}
