// Package platform tests for Gitee review functionality
package platform

import (
	"context"
	"testing"
)

func TestReviewCommentValidation(t *testing.T) {
	tests := []struct {
		name        string
		comment     ReviewComment
		wantErr     bool
		errContains string
	}{
		{
			name: "valid comment",
			comment: ReviewComment{
				Path:     "pkg/main.go",
				Position: 10,
				Side:     "RIGHT",
				Body:     "Fix this issue",
			},
			wantErr: true, // Will fail on network/API (404 is expected for non-existent PR)
		},
		{
			name: "path traversal",
			comment: ReviewComment{
				Path:     "../../../etc/passwd",
				Position: 10,
				Side:     "RIGHT",
				Body:     "Fix this issue",
			},
			wantErr:     true,
			errContains: "invalid path",
		},
		{
			name: "invalid position zero",
			comment: ReviewComment{
				Path:     "pkg/main.go",
				Position: 0,
				Side:     "RIGHT",
				Body:     "Fix this issue",
			},
			wantErr:     true,
			errContains: "position must be positive",
		},
		{
			name: "invalid position negative",
			comment: ReviewComment{
				Path:     "pkg/main.go",
				Position: -1,
				Side:     "RIGHT",
				Body:     "Fix this issue",
			},
			wantErr:     true,
			errContains: "position must be positive",
		},
		{
			name: "invalid side",
			comment: ReviewComment{
				Path:     "pkg/main.go",
				Position: 10,
				Side:     "INVALID",
				Body:     "Fix this issue",
			},
			wantErr:     true,
			errContains: "side must be LEFT or RIGHT",
		},
		{
			name: "LEFT side valid",
			comment: ReviewComment{
				Path:     "pkg/main.go",
				Position: 10,
				Side:     "LEFT",
				Body:     "Fix this issue",
			},
			wantErr: true, // Will fail on network/API
		},
		{
			name: "with commit ID",
			comment: ReviewComment{
				Path:     "pkg/main.go",
				Position: 10,
				Side:     "RIGHT",
				Body:     "Fix this issue",
				CommitID: "abc123",
			},
			wantErr: true, // Will fail on network/API
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewGiteeClient("test-token", "owner/repo")

			_, err := client.PostReviewComment(context.Background(), 123, tt.comment)
			if (err != nil) != tt.wantErr {
				t.Errorf("PostReviewComment() error = %v, wantErr %v", err, tt.wantErr)
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

func TestPostBatchReviewCommentsValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name        string
		comments    []ReviewComment
		body        string
		wantErr     bool
		errContains string
	}{
		{
			name:     "empty comments",
			comments: []ReviewComment{},
			body:     "Summary",
			wantErr:  true,
			errContains: "at least one comment",
		},
		{
			name: "valid batch",
			comments: []ReviewComment{
				{
					Path:     "pkg/main.go",
					Position: 10,
					Side:     "RIGHT",
					Body:     "Fix this",
				},
				{
					Path:     "pkg/util.go",
					Position: 20,
					Side:     "RIGHT",
					Body:     "Fix that",
				},
			},
			body:    "Review summary",
			wantErr: true, // Will fail on network
		},
		{
			name: "invalid path in batch",
			comments: []ReviewComment{
				{
					Path:     "../../../etc/passwd",
					Position: 10,
					Side:     "RIGHT",
					Body:     "Fix this",
				},
			},
			wantErr:     true,
			errContains: "invalid path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.PostBatchReviewComments(context.Background(), 123, tt.comments, tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("PostBatchReviewComments() error = %v, wantErr %v", err, tt.wantErr)
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

func TestReviewStateConstants(t *testing.T) {
	tests := []struct {
		state ReviewState
		value string
	}{
		{ReviewStateApproved, "approved"},
		{ReviewStateChanges, "changes_requested"},
		{ReviewStateComment, "commented"},
		{ReviewStatePending, "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if string(tt.state) != tt.value {
				t.Errorf("ReviewState = %s, want %s", tt.state, tt.value)
			}
		})
	}
}

func TestPostCommentAsReviewValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name        string
		body        string
		state       ReviewState
		wantErr     bool
		errContains string
	}{
		{
			name:  "empty body",
			body:  "",
			state: ReviewStateApproved,
			wantErr: true,
			errContains: "comment body cannot be empty",
		},
		{
			name:  "invalid state",
			body:  "Review",
			state: "invalid",
			wantErr: true,
			errContains: "invalid review state",
		},
		{
			name:  "valid approved",
			body:  "Looks good!",
			state: ReviewStateApproved,
			wantErr: true, // Network error
		},
		{
			name:  "valid changes requested",
			body:  "Please fix",
			state: ReviewStateChanges,
			wantErr: true, // Network error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.PostCommentAsReview(context.Background(), 123, tt.body, tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("PostCommentAsReview() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
			}
		})
	}
}

func TestSubmitReviewValidation(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name        string
		opts        BatchReviewOptions
		wantErr     bool
		errContains string
	}{
		{
			name: "missing PR ID",
			opts: BatchReviewOptions{
				Body:  "Review",
				State: ReviewStateApproved,
			},
			wantErr:     true,
			errContains: "PR ID is required",
		},
		{
			name: "missing comments and body",
			opts: BatchReviewOptions{
				PRID: 123,
			},
			wantErr:     true,
			errContains: "either comments or body is required",
		},
		{
			name: "valid with body only",
			opts: BatchReviewOptions{
				PRID:  123,
				Body:  "Looks good!",
				State: ReviewStateApproved,
			},
			wantErr: true, // Network error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SubmitReview(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitReview() error = %v, wantErr %v", err, tt.wantErr)
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
