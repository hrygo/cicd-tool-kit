// Package platform provides Gitee review and comment functionality
package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ReviewComment represents a line-level review comment on Gitee
type ReviewComment struct {
	// Path is the file path relative to repository root
	Path string `json:"path"`
	// Position is the line number in the diff
	Position int `json:"position"`
	// Side indicates which side of the diff: "LEFT" (base) or "RIGHT" (head)
	Side string `json:"side"`
	// Body is the comment content
	Body string `json:"body"`
	// CommitID is the specific commit to attach comment to (optional)
	CommitID string `json:"commit_id,omitempty"`
}

// ReviewState represents the overall review state
type ReviewState string

const (
	// ReviewStateApproved indicates the PR is approved
	ReviewStateApproved ReviewState = "approved"
	// ReviewStateChanges requests changes
	ReviewStateChanges ReviewState = "changes_requested"
	// ReviewStateComment is a general review comment
	ReviewStateComment ReviewState = "commented"
	// ReviewStatePending indicates review is pending
	ReviewStatePending ReviewState = "pending"
)

// ReviewCommentRequest represents a request to create a review comment
type ReviewCommentRequest struct {
	// PRID is the pull request number
	PRID int `json:"-"`
	// Comments is the list of line-level comments
	Comments []ReviewComment `json:"comments"`
	// Body is the overall review summary
	Body string `json:"body"`
	// Event is the review action (approve, request_changes, comment)
	Event ReviewState `json:"event"`
	// CommitID is the head commit SHA
	CommitID string `json:"commit_id,omitempty"`
}

// ReviewCommentResponse represents the response from creating a review comment
type ReviewCommentResponse struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	Path      string    `json:"path"`
	Position  int       `json:"position"`
	Side      string    `json:"side"`
	CommitID  string    `json:"commit_id"`
	User      GiteeUser `json:"user"`
	CreatedAt string    `json:"created_at"`
}

// PostReviewComment posts a line-level review comment to a Gitee pull request
func (g *GiteeClient) PostReviewComment(ctx context.Context, prID int, comment ReviewComment) (*ReviewCommentResponse, error) {
	if err := validatePath(comment.Path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	if comment.Position <= 0 {
		return nil, fmt.Errorf("position must be positive, got %d", comment.Position)
	}

	if comment.Side != "LEFT" && comment.Side != "RIGHT" {
		return nil, fmt.Errorf("side must be LEFT or RIGHT, got %s", comment.Side)
	}

	payload := map[string]interface{}{
		"body":     comment.Body,
		"path":     comment.Path,
		"position": comment.Position,
		"side":     comment.Side,
	}

	if comment.CommitID != "" {
		payload["commit_id"] = comment.CommitID
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal review comment: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%d/comments", g.baseURL, url.QueryEscape(g.repo), prID)

	resp, err := g.doRequest(ctx, "POST", apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to post review comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to post review comment (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result ReviewCommentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode review comment response: %w", err)
	}

	return &result, nil
}

// PostBatchReviewComments posts multiple line-level comments as a single review
func (g *GiteeClient) PostBatchReviewComments(ctx context.Context, prID int, comments []ReviewComment, body string) error {
	if len(comments) == 0 {
		return fmt.Errorf("at least one comment is required")
	}

	// Validate all comments first
	for i, c := range comments {
		if err := validatePath(c.Path); err != nil {
			return fmt.Errorf("comment %d: invalid path: %w", i, err)
		}
		if c.Position <= 0 {
			return fmt.Errorf("comment %d: position must be positive, got %d", i, c.Position)
		}
		if c.Side != "LEFT" && c.Side != "RIGHT" {
			return fmt.Errorf("comment %d: side must be LEFT or RIGHT, got %s", i, c.Side)
		}
	}

	// Gitee API v5 doesn't have a batch review endpoint like GitHub
	// We need to post comments individually, then add a summary comment
	for _, comment := range comments {
		if _, err := g.PostReviewComment(ctx, prID, comment); err != nil {
			return fmt.Errorf("failed to post comment for %s:%d: %w", comment.Path, comment.Position, err)
		}
	}

	// Post summary as a regular PR comment
	if body != "" {
		if err := g.PostComment(ctx, CommentOptions{PRID: prID, Body: body}); err != nil {
			return fmt.Errorf("failed to post review summary: %w", err)
		}
	}

	return nil
}

// GetReviewComments retrieves all review comments for a pull request
func (g *GiteeClient) GetReviewComments(ctx context.Context, prID int) ([]ReviewCommentResponse, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%d/comments", g.baseURL, url.QueryEscape(g.repo), prID)

	resp, err := g.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get review comments: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get review comments (status %d)", resp.StatusCode)
	}

	var results []ReviewCommentResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode review comments response: %w", err)
	}

	return results, nil
}

// UpdateReviewComment updates an existing review comment
func (g *GiteeClient) UpdateReviewComment(ctx context.Context, prID, commentID int, body string) error {
	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	payload := map[string]string{
		"body": body,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%d/comments/%d", g.baseURL, url.QueryEscape(g.repo), prID, commentID)

	resp, err := g.doRequest(ctx, "PATCH", apiURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to update review comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update review comment (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteReviewComment deletes a review comment
func (g *GiteeClient) DeleteReviewComment(ctx context.Context, prID, commentID int) error {
	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%d/comments/%d", g.baseURL, url.QueryEscape(g.repo), prID, commentID)

	resp, err := g.doRequest(ctx, "DELETE", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to delete review comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete review comment (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ResolveReviewComment resolves a conversation thread on a review comment
func (g *GiteeClient) ResolveReviewComment(ctx context.Context, prID, commentID int) error {
	payload := map[string]bool{
		"resolved": true,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal resolve request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%d/comments/%d", g.baseURL, url.QueryEscape(g.repo), prID, commentID)

	resp, err := g.doRequest(ctx, "PATCH", apiURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to resolve review comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to resolve review comment (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// UnresolveReviewComment unresolves a conversation thread
func (g *GiteeClient) UnresolveReviewComment(ctx context.Context, prID, commentID int) error {
	payload := map[string]bool{
		"resolved": false,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal unresolve request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%d/comments/%d", g.baseURL, url.QueryEscape(g.repo), prID, commentID)

	resp, err := g.doRequest(ctx, "PATCH", apiURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to unresolve review comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to unresolve review comment (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// PostCommentAsReview posts a PR-level comment with review context
func (g *GiteeClient) PostCommentAsReview(ctx context.Context, prID int, body string, state ReviewState) error {
	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	if state != ReviewStateApproved && state != ReviewStateChanges && state != ReviewStateComment && state != ReviewStatePending {
		return fmt.Errorf("invalid review state: %s", state)
	}

	// Add state indicator to the comment
	var formattedBody string
	switch state {
	case ReviewStateApproved:
		formattedBody = fmt.Sprintf("✅ **APPROVED**\n\n%s", body)
	case ReviewStateChanges:
		formattedBody = fmt.Sprintf("🔄 **CHANGES REQUESTED**\n\n%s", body)
	case ReviewStateComment:
		formattedBody = fmt.Sprintf("💬 **REVIEWED**\n\n%s", body)
	case ReviewStatePending:
		formattedBody = fmt.Sprintf("⏳ **PENDING REVIEW**\n\n%s", body)
	}

	return g.PostComment(ctx, CommentOptions{PRID: prID, Body: formattedBody})
}

// GetLatestCommitID gets the latest commit SHA for a PR
func (g *GiteeClient) GetLatestCommitID(ctx context.Context, prID int) (string, error) {
	prInfo, err := g.GetPRInfo(ctx, prID)
	if err != nil {
		return "", fmt.Errorf("failed to get PR info: %w", err)
	}
	return prInfo.SHA, nil
}

// BatchReviewOptions represents options for batch review operations
type BatchReviewOptions struct {
	PRID     int
	Body     string
	State    ReviewState
	Comments []ReviewComment
}

// SubmitReview submits a complete review with optional line comments
func (g *GiteeClient) SubmitReview(ctx context.Context, opts BatchReviewOptions) error {
	if opts.PRID == 0 {
		return fmt.Errorf("PR ID is required")
	}

	if len(opts.Comments) == 0 && opts.Body == "" {
		return fmt.Errorf("either comments or body is required")
	}

	// Post all line-level comments
	if len(opts.Comments) > 0 {
		for _, comment := range opts.Comments {
			if _, err := g.PostReviewComment(ctx, opts.PRID, comment); err != nil {
				return fmt.Errorf("failed to post review comment: %w", err)
			}
		}
	}

	// Post the review state and summary
	if opts.Body != "" {
		state := opts.State
		if state == "" {
			state = ReviewStateComment
		}
		if err := g.PostCommentAsReview(ctx, opts.PRID, opts.Body, state); err != nil {
			return fmt.Errorf("failed to post review summary: %w", err)
		}
	}

	return nil
}
