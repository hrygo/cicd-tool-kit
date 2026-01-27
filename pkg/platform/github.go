// Package platform provides GitHub platform implementation
package platform

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/errors"
)

// GitHubClient implements Platform for GitHub
type GitHubClient struct {
	token   string
 baseURL string // For GitHub Enterprise
	repo    string // owner/repo format
	client  *http.Client
}

// GitHubAPIResponse represents common GitHub API response structure
type GitHubAPIResponse struct {
	Message string `json:"message"`
	Docs    string `json:"documentation_url"`
}

// GitHubPR represents GitHub pull request response
type GitHubPR struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Head       GitHubPRRef `json:"head"`
	Base       GitHubPRRef `json:"base"`
	User       GitHubUser `json:"user"`
	HTMLURL    string `json:"html_url"`
	State      string `json:"state"`
	MergedAt   *time.Time `json:"merged_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// GitHubPRRef represents a git reference in a PR
type GitHubPRRef struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
	Repo GitHubRepo `json:"repo"`
}

// GitHubRepo represents GitHub repository info
type GitHubRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    GitHubUser `json:"owner"`
}

// GitHubUser represents GitHub user info
type GitHubUser struct {
	Login string `json:"login"`
}

// GitHubReviewComment represents a review comment
type GitHubReviewComment struct {
	Body     string `json:"body"`
	Path     string `json:"path,omitempty"`
	Position *int    `json:"position,omitempty"`
	Line     *int    `json:"line,omitempty"`
}

// NewGitHubClient creates a new GitHub platform client
func NewGitHubClient(token, repo string) *GitHubClient {
	return &GitHubClient{
		token:   token,
		baseURL: "https://api.github.com",
		repo:    repo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL sets a custom base URL for GitHub Enterprise
func (c *GitHubClient) SetBaseURL(url string) {
	c.baseURL = url
}

// Name returns the platform name
func (c *GitHubClient) Name() string {
	return "github"
}

// PostComment posts a review comment to a pull request
func (c *GitHubClient) PostComment(ctx context.Context, opts CommentOptions) error {
	if opts.AsReview {
		return c.postReviewComment(ctx, opts)
	}
	return c.postSimpleComment(ctx, opts)
}

// postSimpleComment posts a simple PR comment
func (c *GitHubClient) postSimpleComment(ctx context.Context, opts CommentOptions) error {
	url := fmt.Sprintf("%s/repos/%s/issues/%d/comments", c.baseURL, c.repo, opts.PRID)

	payload := map[string]string{
		"body": opts.Body,
	}

	return c.doRequest(ctx, "POST", url, payload, nil)
}

// postReviewComment posts a review comment
func (c *GitHubClient) postReviewComment(ctx context.Context, opts CommentOptions) error {
	// For GitHub, we use the Pull Request review API
	// First, we need to get the latest commit SHA
	pr, err := c.getPR(ctx, opts.PRID)
	if err != nil {
		return fmt.Errorf("failed to get PR info: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/pulls/%d/reviews", c.baseURL, c.repo, opts.PRID)

	payload := map[string]interface{}{
		"commit_id": pr.Head.SHA,
		"body":      opts.Body,
		"event":     "COMMENT", // Can be APPROVE, REQUEST_CHANGES, or COMMENT
		"comments":  []GitHubReviewComment{},
	}

	// If position is specified, add line-specific comment
	if opts.Position != nil {
		payload["comments"] = []GitHubReviewComment{
			{
				Body: opts.Body,
				Path: opts.Position.Path,
				Line: &opts.Position.Line,
			},
		}
	}

	return c.doRequest(ctx, "POST", url, payload, nil)
}

// GetDiff retrieves the diff for a pull request
func (c *GitHubClient) GetDiff(ctx context.Context, prID int) (string, error) {
	// Try the API first
	url := fmt.Sprintf("%s/repos/%s/pulls/%d", c.baseURL, c.repo, prID)

	var pr GitHubPR
	if err := c.doRequest(ctx, "GET", url, nil, &pr); err != nil {
		return "", err
	}

	// Get diff URL
	diffURL := fmt.Sprintf("%s/repos/%s/pulls/%d.diff", c.baseURL, c.repo, prID)

	req, err := http.NewRequestWithContext(ctx, "GET", diffURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get diff: %s", string(body))
	}

	diff, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read diff: %w", err)
	}

	return string(diff), nil
}

// GetFile retrieves a file's content at a specific ref
func (c *GitHubClient) GetFile(ctx context.Context, path, ref string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", c.baseURL, c.repo, path, ref)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get file: %s", string(body))
	}

	var result struct {
		Content string `json:"content"`
		Encoding string `json:"encoding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// GitHub returns base64 encoded content
	if result.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(result.Content)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 content: %w", err)
		}
		return string(decoded), nil
	}

	return result.Content, nil
}

// GetPRInfo retrieves pull request metadata
func (c *GitHubClient) GetPRInfo(ctx context.Context, prID int) (*PRInfo, error) {
	pr, err := c.getPR(ctx, prID)
	if err != nil {
		return nil, err
	}

	return &PRInfo{
		Number:      pr.Number,
		Title:       pr.Title,
		Description: pr.Body,
		Author:      pr.User.Login,
		SHA:         pr.Head.SHA,
		BaseBranch:  pr.Base.Ref,
		HeadBranch:  pr.Head.Ref,
		SourceRepo:  pr.Head.Repo.FullName,
	}, nil
}

// getPR fetches a pull request from GitHub
func (c *GitHubClient) getPR(ctx context.Context, prID int) (*GitHubPR, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d", c.baseURL, c.repo, prID)

	var pr GitHubPR
	if err := c.doRequest(ctx, "GET", url, nil, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

// Health checks if the GitHub API is accessible
func (c *GitHubClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.PlatformError("GitHub API unreachable", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.PlatformError(fmt.Sprintf("GitHub API returned status %d", resp.StatusCode), nil)
	}

	return nil
}

// doRequest performs an HTTP request with auth and JSON handling
func (c *GitHubClient) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeader(req)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.PlatformError("request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr GitHubAPIResponse
		_ = json.Unmarshal(respBody, &apiErr)
		return errors.PlatformError(fmt.Sprintf("GitHub API error: %s", apiErr.Message), nil)
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// setAuthHeader sets the authorization header
func (c *GitHubClient) setAuthHeader(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}
}

// ParseRepoFromEnv parses repo info from GitHub Actions environment variables
func ParseRepoFromEnv() (string, error) {
	// GitHub Actions sets GITHUB_REPOSITORY to owner/repo
	repo := osGetenv("GITHUB_REPOSITORY")
	if repo == "" {
		return "", errors.ConfigError("GITHUB_REPOSITORY not set", nil)
	}
	return repo, nil
}

// ParsePRIDFromEnv parses PR ID from GitHub Actions environment variables
func ParsePRIDFromEnv() (int, error) {
	// GitHub Actions sets the PR number in various places
	// For pull_request events, it's in github.event.number
	prNum := osGetenv("GITHUB_PR_NUMBER")
	if prNum == "" {
		// Try alternative env vars
		prNum = osGetenv("GH_PR_NUMBER")
	}
	if prNum == "" {
		return 0, errors.ConfigError("PR number not found in environment", nil)
	}

	var id int
	if _, err := fmt.Sscanf(prNum, "%d", &id); err != nil {
		return 0, errors.ConfigError("invalid PR number format", err)
	}

	// Add bounds validation to prevent unreasonable values
	if id <= 0 || id > 100000000 {
		return 0, errors.ConfigError("PR number out of valid range (1-100000000)", nil)
	}

	return id, nil
}

// IsGitHubEnv returns true if running in GitHub Actions
func IsGitHubEnv() bool {
	return osGetenv("GITHUB_ACTIONS") == "true"
}

// osGetenv is a wrapper for os.Getenv to allow testing
var osGetenv = os.Getenv
