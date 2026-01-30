// Package platform tests for Gitee status functionality
package platform

import (
	"context"
	"testing"
	"time"
)

func TestStatusStateConstants(t *testing.T) {
	tests := []struct {
		state StatusState
		value string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusSuccess, "success"},
		{StatusFailed, "fail"},
		{StatusError, "error"},
		{StatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if tt.state.String() != tt.value {
				t.Errorf("StatusState.String() = %s, want %s", tt.state.String(), tt.value)
			}
		})
	}
}

func TestCreateStatusValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name        string
		sha         string
		opts        StatusOptions
		wantErr     bool
		errContains string
	}{
		{
			name: "empty SHA",
			sha:  "",
			opts: StatusOptions{
				State: StatusSuccess,
			},
			wantErr:     true,
			errContains: "commit SHA cannot be empty",
		},
		{
			name:    "valid with defaults",
			sha:     "abc123",
			opts:    StatusOptions{},
			wantErr: true, // Network error
		},
		{
			name: "valid with all options",
			sha:  "abc123",
			opts: StatusOptions{
				State:       StatusSuccess,
				TargetURL:   "https://example.com",
				Description: "All checks passed",
				Context:     "cicd-ai-toolkit",
			},
			wantErr: true, // Network error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateStatus(context.Background(), tt.sha, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateStatus() error = %v, wantErr %v", err, tt.wantErr)
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

func TestGetStatusesValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name        string
		sha         string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty SHA",
			sha:         "",
			wantErr:     true,
			errContains: "commit SHA cannot be empty",
		},
		{
			name:    "valid SHA",
			sha:     "abc123",
			wantErr: true, // Network error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetStatuses(context.Background(), tt.sha)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStatuses() error = %v, wantErr %v", err, tt.wantErr)
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

func TestGetCombinedStatusValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	_, err := client.GetCombinedStatus(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty SHA")
	}
	if !contains(err.Error(), "commit SHA cannot be empty") {
		t.Errorf("Expected error about empty SHA, got: %v", err)
	}
}

func TestHelperStatusFunctions(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name        string
		fn          func() (*GiteeStatus, error)
		wantErr     bool
		errContains string
	}{
		{
			name: "CreatePendingStatus",
			fn: func() (*GiteeStatus, error) {
				return client.CreatePendingStatus(context.Background(), "abc123", "Running...", "cicd")
			},
			wantErr: true, // Network error
		},
		{
			name: "CreateRunningStatus",
			fn: func() (*GiteeStatus, error) {
				return client.CreateRunningStatus(context.Background(), "abc123", "Running...", "cicd")
			},
			wantErr: true, // Network error
		},
		{
			name: "CreateSuccessStatus",
			fn: func() (*GiteeStatus, error) {
				return client.CreateSuccessStatus(context.Background(), "abc123", "Passed!", "cicd", "https://example.com")
			},
			wantErr: true, // Network error
		},
		{
			name: "CreateFailureStatus",
			fn: func() (*GiteeStatus, error) {
				return client.CreateFailureStatus(context.Background(), "abc123", "Failed", "cicd")
			},
			wantErr: true, // Network error
		},
		{
			name: "CreateErrorStatus",
			fn: func() (*GiteeStatus, error) {
				return client.CreateErrorStatus(context.Background(), "abc123", "Error", "cicd")
			},
			wantErr: true, // Network error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fn()
			if (err != nil) != tt.wantErr {
				t.Errorf("Function error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMergeOptionsDefaults(t *testing.T) {
	opts := DefaultMergeOptions()
	if opts.Method != "merge" {
		t.Errorf("DefaultMergeOptions().Method = %s, want 'merge'", opts.Method)
	}
}

func TestStatusCheckConfigDefaults(t *testing.T) {
	config := StatusCheckConfig{}
	if config.Timeout != 0 {
		// Should be 0 by default (set in WaitForStatusChecks)
	}
	if config.PollInterval != 0 {
		t.Errorf("PollInterval should be 0 by default, got %v", config.PollInterval)
	}
}

func TestCheckPRStatusChecksValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	// This will fail due to network error, but we can test the validation
	_, err := client.CheckPRStatusChecks(context.Background(), 123, []string{"cicd-ai-toolkit"})
	if err != nil {
		// Expected to fail on network
		if !contains(err.Error(), "failed to get PR info") && !contains(err.Error(), "failed to get statuses") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestMergeValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name    string
		prID    int
		opts    MergeOptions
		wantErr bool
	}{
		{
			name: "valid merge request",
			prID: 123,
			opts: MergeOptions{
				Method: "merge",
			},
			wantErr: true, // Network error
		},
		{
			name: "squash merge",
			prID: 123,
			opts: MergeOptions{
				Method:        "squash",
				CommitTitle:   "Squashed commit",
				CommitMessage: "Squashed message",
			},
			wantErr: true, // Network error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.MergePR(context.Background(), tt.prID, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergePR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPRMergeStatusValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	_, err := client.GetPRMergeStatus(context.Background(), 123)
	if err != nil {
		// Expected to fail on network
		if !contains(err.Error(), "failed to get merge status") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestStatusCheckResultFields(t *testing.T) {
	result := &StatusCheckResult{
		SHA:              "abc123",
		State:            StatusSuccess,
		TotalCount:       5,
		Statuses:         []GiteeStatus{},
		Contexts:         make(map[string]string),
		CanMerge:         true,
		RequiredContexts: []string{"cicd-ai-toolkit"},
	}

	if result.SHA != "abc123" {
		t.Errorf("SHA = %s, want abc123", result.SHA)
	}
	if result.State != StatusSuccess {
		t.Errorf("State = %s, want success", result.State)
	}
	if !result.CanMerge {
		t.Error("CanMerge should be true")
	}
}

// TestStatusCheckConfig verifies StatusCheckConfig struct
func TestStatusCheckConfig(t *testing.T) {
	config := StatusCheckConfig{
		RequiredContexts:  []string{"cicd-ai-toolkit", "security-scan"},
		WaitForCompletion: true,
		Timeout:           5 * time.Minute,
		PollInterval:      5 * time.Second,
	}

	if len(config.RequiredContexts) != 2 {
		t.Errorf("Expected 2 required contexts, got %d", len(config.RequiredContexts))
	}
	if !config.WaitForCompletion {
		t.Error("WaitForCompletion should be true")
	}
	if config.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want 5m", config.Timeout)
	}
	if config.PollInterval != 5*time.Second {
		t.Errorf("PollInterval = %v, want 5s", config.PollInterval)
	}
}

func TestMergeStatusFields(t *testing.T) {
	now := time.Now()
	status := &MergeStatus{
		CanMerge:       true,
		Mergeable:      true,
		Merged:         false,
		MergedAt:       &now,
		MergeCommitSHA: "merged123",
		Message:        "Ready to merge",
	}

	if !status.CanMerge {
		t.Error("CanMerge should be true")
	}
	if !status.Mergeable {
		t.Error("Mergeable should be true")
	}
	if status.Merged {
		t.Error("Merged should be false")
	}
	if status.MergeCommitSHA != "merged123" {
		t.Errorf("MergeCommitSHA = %s, want merged123", status.MergeCommitSHA)
	}
}
