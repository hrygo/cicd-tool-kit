// Package platform provides Gitee status check functionality
package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// StatusState represents the state of a status check
type StatusState string

const (
	// StatusPending indicates the check is pending
	StatusPending StatusState = "pending"
	// StatusRunning indicates the check is running
	StatusRunning StatusState = "running"
	// StatusSuccess indicates the check passed
	StatusSuccess StatusState = "success"
	// StatusFailed indicates the check failed
	StatusFailed StatusState = "fail"
	// StatusError indicates an error occurred
	StatusError StatusState = "error"
	// StatusCancelled indicates the check was cancelled
	StatusCancelled StatusState = "cancelled"
)

// String returns the string representation of the status state
func (s StatusState) String() string {
	return string(s)
}

// StatusOptions represents options for creating a status check
type StatusOptions struct {
	// State is the status state
	State StatusState
	// TargetURL is a URL to associate with this status
	TargetURL string
	// Description is a short description of the status
	Description string
	// Context is a string label to differentiate this status from others
	Context string
}

// GiteeStatus represents a Gitee commit status
type GiteeStatus struct {
	ID          int         `json:"id"`
	State       StatusState `json:"state"`
	TargetURL   string      `json:"target_url"`
	Description string      `json:"description"`
	Context     string      `json:"context"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Creator     GiteeUser   `json:"creator"`
	SHA         string      `json:"sha"`
}

// StatusListResponse represents the response when listing statuses
type StatusListResponse struct {
	Statuses []GiteeStatus `json:"statuses"`
	Total    int           `json:"total_count"`
}

// CreateStatus creates a status check on a commit
func (g *GiteeClient) CreateStatus(ctx context.Context, sha string, opts StatusOptions) (*GiteeStatus, error) {
	if sha == "" {
		return nil, fmt.Errorf("commit SHA cannot be empty")
	}

	if opts.State == "" {
		opts.State = StatusPending
	}

	if opts.Context == "" {
		opts.Context = "cicd-ai-toolkit"
	}

	payload := map[string]interface{}{
		"sha":         sha,
		"state":       opts.State.String(),
		"description": opts.Description,
		"context":     opts.Context,
	}

	if opts.TargetURL != "" {
		payload["target_url"] = opts.TargetURL
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/statuses/%s", g.baseURL, url.QueryEscape(g.repo), url.QueryEscape(sha))

	resp, err := g.doRequest(ctx, "POST", apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create status (status %d): %s", resp.StatusCode, string(respBody))
	}

	var status GiteeStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &status, nil
}

// GetStatuses retrieves all statuses for a commit
func (g *GiteeClient) GetStatuses(ctx context.Context, sha string) ([]GiteeStatus, error) {
	if sha == "" {
		return nil, fmt.Errorf("commit SHA cannot be empty")
	}

	apiURL := fmt.Sprintf("%s/repos/%s/statuses/%s", g.baseURL, url.QueryEscape(g.repo), url.QueryEscape(sha))

	resp, err := g.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get statuses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get statuses (status %d)", resp.StatusCode)
	}

	var result StatusListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode statuses response: %w", err)
	}

	return result.Statuses, nil
}

// GetCombinedStatus retrieves the combined status for a commit
func (g *GiteeClient) GetCombinedStatus(ctx context.Context, sha string) (*GiteeStatus, error) {
	if sha == "" {
		return nil, fmt.Errorf("commit SHA cannot be empty")
	}

	apiURL := fmt.Sprintf("%s/repos/%s/status/%s", g.baseURL, url.QueryEscape(g.repo), url.QueryEscape(sha))

	resp, err := g.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get combined status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get combined status (status %d)", resp.StatusCode)
	}

	var status GiteeStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode combined status response: %w", err)
	}

	return &status, nil
}

// StatusCheckResult represents the result of checking if PR can merge
type StatusCheckResult struct {
	SHA              string            `json:"sha"`
	State            StatusState       `json:"state"`
	TotalCount       int               `json:"total_count"`
	Statuses         []GiteeStatus     `json:"statuses"`
	Contexts         map[string]string `json:"contexts"` // context -> state mapping
	CanMerge         bool              `json:"can_merge"`
	RequiredContexts []string          `json:"required_contexts"`
}

// CheckPRStatusChecks checks if all required status checks have passed for a PR
func (g *GiteeClient) CheckPRStatusChecks(ctx context.Context, prID int, requiredContexts []string) (*StatusCheckResult, error) {
	prInfo, err := g.GetPRInfo(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR info: %w", err)
	}

	statuses, err := g.GetStatuses(ctx, prInfo.SHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get statuses: %w", err)
	}

	result := &StatusCheckResult{
		SHA:              prInfo.SHA,
		Statuses:         statuses,
		TotalCount:       len(statuses),
		Contexts:         make(map[string]string),
		RequiredContexts: requiredContexts,
	}

	// Build context map and determine overall state
	hasFailed := false
	hasPending := false
	allSuccess := true

	for _, status := range statuses {
		result.Contexts[status.Context] = status.State.String()
		switch status.State {
		case StatusFailed, StatusError:
			hasFailed = true
			allSuccess = false
		case StatusPending, StatusRunning:
			hasPending = true
			allSuccess = false
		case StatusSuccess:
			// Continue checking
		case StatusCancelled:
			allSuccess = false
		}
	}

	// Determine overall state
	if hasFailed {
		result.State = StatusFailed
	} else if hasPending {
		result.State = StatusPending
	} else if allSuccess && len(statuses) > 0 {
		result.State = StatusSuccess
	} else {
		result.State = StatusPending
	}

	// Check if all required contexts have passed
	result.CanMerge = true
	for _, required := range requiredContexts {
		if state, ok := result.Contexts[required]; !ok || state != StatusSuccess.String() {
			result.CanMerge = false
			break
		}
	}

	return result, nil
}

// CreatePendingStatus creates a pending status check
func (g *GiteeClient) CreatePendingStatus(ctx context.Context, sha, description, context string) (*GiteeStatus, error) {
	return g.CreateStatus(ctx, sha, StatusOptions{
		State:       StatusPending,
		Description: description,
		Context:     context,
	})
}

// CreateRunningStatus creates a running status check
func (g *GiteeClient) CreateRunningStatus(ctx context.Context, sha, description, context string) (*GiteeStatus, error) {
	return g.CreateStatus(ctx, sha, StatusOptions{
		State:       StatusRunning,
		Description: description,
		Context:     context,
	})
}

// CreateSuccessStatus creates a success status check
func (g *GiteeClient) CreateSuccessStatus(ctx context.Context, sha, description, context string, targetURL string) (*GiteeStatus, error) {
	return g.CreateStatus(ctx, sha, StatusOptions{
		State:       StatusSuccess,
		Description: description,
		Context:     context,
		TargetURL:   targetURL,
	})
}

// CreateFailureStatus creates a failure status check
func (g *GiteeClient) CreateFailureStatus(ctx context.Context, sha, description, context string) (*GiteeStatus, error) {
	return g.CreateStatus(ctx, sha, StatusOptions{
		State:       StatusFailed,
		Description: description,
		Context:     context,
	})
}

// CreateErrorStatus creates an error status check
func (g *GiteeClient) CreateErrorStatus(ctx context.Context, sha, description, context string) (*GiteeStatus, error) {
	return g.CreateStatus(ctx, sha, StatusOptions{
		State:       StatusError,
		Description: description,
		Context:     context,
	})
}

// StatusCheckConfig represents configuration for status check validation
type StatusCheckConfig struct {
	RequiredContexts  []string
	WaitForCompletion bool
	Timeout           time.Duration
	PollInterval      time.Duration
}

// WaitForStatusChecks waits for all required status checks to complete
func (g *GiteeClient) WaitForStatusChecks(ctx context.Context, prID int, config StatusCheckConfig) (*StatusCheckResult, error) {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute
	}
	if config.PollInterval == 0 {
		config.PollInterval = 10 * time.Second
	}

	deadline := time.Now().Add(config.Timeout)
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	for {
		result, err := g.CheckPRStatusChecks(ctx, prID, config.RequiredContexts)
		if err != nil {
			return nil, err
		}

		// Check if all required contexts have completed
		allCompleted := true
		for _, required := range config.RequiredContexts {
			if state, ok := result.Contexts[required]; !ok {
				allCompleted = false
				break
			} else if state == StatusPending.String() || state == StatusRunning.String() {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			return result, nil
		}

		// Check timeout
		if time.Now().After(deadline) {
			return result, fmt.Errorf("timeout waiting for status checks")
		}

		// Wait for next poll or context cancellation
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// MergeStatus represents the merge status of a PR
type MergeStatus struct {
	CanMerge       bool       `json:"can_merge"`
	Mergeable      bool       `json:"mergeable"`
	Merged         bool       `json:"merged"`
	MergedAt       *time.Time `json:"merged_at"`
	MergeCommitSHA string     `json:"merge_commit_sha"`
	Message        string     `json:"message"`
}

// GetPRMergeStatus checks if a PR can be merged
func (g *GiteeClient) GetPRMergeStatus(ctx context.Context, prID int) (*MergeStatus, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%d/merge", g.baseURL, url.QueryEscape(g.repo), prID)

	resp, err := g.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get merge status (status %d)", resp.StatusCode)
	}

	var result struct {
		Mergeable      bool       `json:"mergeable"`
		Merged         bool       `json:"merged"`
		MergedAt       *time.Time `json:"merged_at"`
		MergeCommitSHA string     `json:"merge_commit_sha"`
		Message        string     `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode merge status response: %w", err)
	}

	return &MergeStatus{
		CanMerge:       result.Mergeable && !result.Merged,
		Mergeable:      result.Mergeable,
		Merged:         result.Merged,
		MergedAt:       result.MergedAt,
		MergeCommitSHA: result.MergeCommitSHA,
		Message:        result.Message,
	}, nil
}

// MergePR merges a pull request
func (g *GiteeClient) MergePR(ctx context.Context, prID int, opts MergeOptions) (*MergeStatus, error) {
	payload := map[string]interface{}{
		"merge_method": opts.Method,
	}

	if opts.CommitTitle != "" {
		payload["title"] = opts.CommitTitle
	}

	if opts.CommitMessage != "" {
		payload["body"] = opts.CommitMessage
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merge request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%d/merge", g.baseURL, url.QueryEscape(g.repo), prID)

	resp, err := g.doRequest(ctx, "PUT", apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to merge PR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to merge PR (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Merged         bool       `json:"merged"`
		MergedAt       *time.Time `json:"merged_at"`
		MergeCommitSHA string     `json:"merge_commit_sha"`
		Message        string     `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode merge response: %w", err)
	}

	return &MergeStatus{
		CanMerge:       false,
		Mergeable:      true,
		Merged:         result.Merged,
		MergedAt:       result.MergedAt,
		MergeCommitSHA: result.MergeCommitSHA,
		Message:        result.Message,
	}, nil
}

// MergeOptions represents options for merging a PR
type MergeOptions struct {
	// Method is the merge method: "merge", "squash", or "rebase"
	Method string
	// CommitTitle is the title for the merge commit
	CommitTitle string
	// CommitMessage is the message for the merge commit
	CommitMessage string
}

// DefaultMergeOptions returns default merge options
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{
		Method: "merge",
	}
}
