// Package platform provides abstraction over CI/CD platforms (GitHub, GitLab, Gitee)
package platform

import "context"

// Platform is the abstraction layer over different CI/CD platforms
type Platform interface {
	// Name returns the platform name (github, gitlab, gitee)
	Name() string

	// PostComment posts a review comment to a pull/merge request
	PostComment(ctx context.Context, opts CommentOptions) error

	// GetDiff retrieves the diff for a pull/merge request
	GetDiff(ctx context.Context, prID int) (string, error)

	// GetFile retrieves a file's content at a specific ref
	GetFile(ctx context.Context, path, ref string) (string, error)

	// GetPRInfo retrieves pull/merge request metadata
	GetPRInfo(ctx context.Context, prID int) (*PRInfo, error)

	// Health checks if the platform API is accessible
	Health(ctx context.Context) error
}

// CommentOptions contains options for posting a comment
type CommentOptions struct {
	PRID      int
	Body      string
	AsReview  bool   // Post as a review comment vs simple comment
	Position  *Position // Optional: for line-specific comments
}

// Position specifies a location in a file for comments
type Position struct {
	Path    string
	Line    int
	SHA     string // Commit SHA for line tracking
}

// PRInfo contains pull/merge request metadata
type PRInfo struct {
	Number      int
	Title       string
	Description string
	Author      string
	SHA         string
	BaseBranch  string
	HeadBranch  string
	SourceRepo  string
	Labels      []string
}

// EventType represents webhook event types
type EventType string

const (
	EventPROpened     EventType = "opened"
	EventPRSynchronize EventType = "synchronize"
	EventPRReopened   EventType = "reopened"
	EventPRClosed     EventType = "closed"
	EventPRMerged     EventType = "merged"
)

// WebhookEvent represents a normalized webhook event
type WebhookEvent struct {
	Type      EventType
	Platform  string
	PRID      int
	Repo      string
	SHA       string
	BaseRef   string
	HeadRef   string
	Timestamp int64
}
