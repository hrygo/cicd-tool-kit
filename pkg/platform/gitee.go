// Package platform provides Gitee Enterprise platform implementation
package platform

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// userAgent is the default User-Agent header for Gitee API requests
	userAgent = "cicd-ai-toolkit/1.0"
)

// GiteeClient implements Platform for Gitee Enterprise
type GiteeClient struct {
	token   string
	baseURL string // For Gitee Enterprise self-hosted
	repo    string // owner/repo format
	client  *http.Client
}

// GiteeAPIResponse represents common Gitee API response structure
type GiteeAPIResponse struct {
	Message string `json:"message"`
}

// GiteePR represents Gitee pull request response
type GiteePR struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Head   GiteePRRef `json:"head"`
	Base   GiteePRRef `json:"base"`
	User   GiteeUser `json:"user"`
	HTMLURL string `json:"html_url"`
	State  string `json:"state"`
	MergedAt *time.Time `json:"merged_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GiteePRRef represents a git reference in a PR
type GiteePRRef struct {
	Ref  string `json:"ref"`
	SHA  string `json:"sha"`
	Repo GiteeRepo `json:"repo"`
}

// GiteeRepo represents Gitee repository info
type GiteeRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    GiteeUser `json:"owner"`
}

// GiteeUser represents Gitee user info
type GiteeUser struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

// GiteeComment represents a comment on Gitee
type GiteeComment struct {
	Body string `json:"body"`
}

// GiteeDiffResponse represents diff response from Gitee
type GiteeDiffResponse struct {
	Files []struct {
		Filename string `json:"filename"`
		Patch    string `json:"patch"`
	} `json:"files"`
}

// Name returns the platform name
func (g *GiteeClient) Name() string {
	return "gitee"
}

// NewGiteeClient creates a new Gitee platform client
func NewGiteeClient(token, repo string) *GiteeClient {
	baseURL := os.Getenv("GITEE_API_URL")
	if baseURL == "" {
		baseURL = "https://gitee.com/api/v5"
	}

	// SECURITY: Validate baseURL to prevent SSRF attacks
	if err := validateBaseURL(baseURL); err != nil {
		// If validation fails, use the default URL
		baseURL = "https://gitee.com/api/v5"
	}

	return &GiteeClient{
		token:   token,
		baseURL: baseURL,
		repo:    repo,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// SetBaseURL sets a custom base URL for Gitee Enterprise
func (g *GiteeClient) SetBaseURL(url string) error {
	// SECURITY: Validate baseURL to prevent SSRF attacks
	if err := validateBaseURL(url); err != nil {
		return err
	}
	g.baseURL = strings.TrimSuffix(url, "/")
	return nil
}

// doRequest performs an HTTP request with common headers and error handling
func (g *GiteeClient) doRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	return g.client.Do(req)
}

// validatePath validates a file path to prevent path traversal attacks
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	// Check for null byte
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path cannot contain null byte")
	}
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains traversal sequence: %s", path)
	}
	// Check for URL-encoded path traversal (case-insensitive for %2e, %5c)
	lowerPath := strings.ToLower(path)
	if strings.Contains(lowerPath, "%2e") || strings.Contains(lowerPath, "%5c") {
		return fmt.Errorf("path contains URL-encoded dots or backslashes")
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("path contains backslash: %s", path)
	}
	// Check for absolute paths (only relative paths allowed for API calls)
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}
	// SECURITY: Reject wildcard characters that could cause glob-based attacks
	if strings.ContainsAny(path, "*?[") {
		return fmt.Errorf("path contains wildcard characters: %s", path)
	}
	return nil
}

// PostComment posts a comment to a Gitee pull request
func (g *GiteeClient) PostComment(ctx context.Context, opts CommentOptions) error {
	if opts.PRID == 0 {
		return fmt.Errorf("PR ID is required")
	}

	comment := GiteeComment{
		Body: opts.Body,
	}

	body, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal comment: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/pulls/%d/comments", g.baseURL, url.QueryEscape(g.repo), opts.PRID)

	resp, err := g.doRequest(ctx, "POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to post comment (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetDiff retrieves the diff for a Gitee pull request
func (g *GiteeClient) GetDiff(ctx context.Context, prID int) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d/files", g.baseURL, url.QueryEscape(g.repo), prID)

	resp, err := g.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get diff (status %d)", resp.StatusCode)
	}

	var result GiteeDiffResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode diff response: %w", err)
	}

	var diffBuilder bytes.Buffer
	for _, file := range result.Files {
		if file.Patch != "" {
			fmt.Fprintf(&diffBuilder, "diff --git a/%s b/%s\n%s\n\n", file.Filename, file.Filename, file.Patch)
		}
	}

	return diffBuilder.String(), nil
}

// GetFile retrieves a file from the Gitee repository
func (g *GiteeClient) GetFile(ctx context.Context, path, ref string) (string, error) {
	if err := validatePath(path); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", g.baseURL, url.QueryEscape(g.repo), url.QueryEscape(path), ref)

	resp, err := g.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get file (status %d)", resp.StatusCode)
	}

	var result struct {
		Content string `json:"content"`
		Encoding string `json:"encoding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode file response: %w", err)
	}

	// Gitee returns base64 encoded content
	if result.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(result.Content)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 content: %w", err)
		}
		return string(decoded), nil
	}

	return result.Content, nil
}

// GetPRInfo retrieves pull request information from Gitee
func (g *GiteeClient) GetPRInfo(ctx context.Context, prID int) (*PRInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d", g.baseURL, url.QueryEscape(g.repo), prID)

	resp, err := g.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get PR info (status %d)", resp.StatusCode)
	}

	var giteePR GiteePR
	if err := json.NewDecoder(resp.Body).Decode(&giteePR); err != nil {
		return nil, fmt.Errorf("failed to decode PR response: %w", err)
	}

	return &PRInfo{
		Number:      giteePR.Number,
		Title:       giteePR.Title,
		Description: giteePR.Body,
		Author:      giteePR.User.Login,
		SHA:         giteePR.Head.SHA,
		BaseBranch:  giteePR.Base.Ref,
		HeadBranch:  giteePR.Head.Ref,
		SourceRepo:  giteePR.Head.Repo.FullName,
	}, nil
}

// Health checks if the Gitee API is accessible
func (g *GiteeClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/repos/%s", g.baseURL, url.QueryEscape(g.repo))

	resp, err := g.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gitee: health check failed (status %d)", resp.StatusCode)
	}

	return nil
}

// IsGiteeEnv checks if running in Gitee environment
func IsGiteeEnv() bool {
	return os.Getenv("GITEE_API_URL") != "" || os.Getenv("GITEE_TOKEN") != ""
}

// ParseRepoFromGiteeEnv parses repo from Gitee environment variables
func ParseRepoFromGiteeEnv() (string, error) {
	// Gitee CI environment variables
	if repo := os.Getenv("GITEE_REPO"); repo != "" {
		return repo, nil
	}

	// Fallback to Gitea-compatible variables
	if owner := os.Getenv("GITEA_REPO_OWNER"); owner != "" {
		if name := os.Getenv("GITEA_REPO_NAME"); name != "" {
			return fmt.Sprintf("%s/%s", owner, name), nil
		}
	}

	return "", fmt.Errorf("could not parse repo from Gitee environment")
}

// ParsePRIDFromGiteeEnv parses PR ID from Gitee environment
func ParsePRIDFromGiteeEnv() (int, error) {
	// Gitee PR number
	if pr := os.Getenv("GITEE_PR_NUMBER"); pr != "" {
		var id int
		if _, err := fmt.Sscanf(pr, "%d", &id); err == nil {
			return id, nil
		}
	}

	// Fallback to Gitea-compatible variables
	if pr := os.Getenv("GITEA_PULL_REQUEST"); pr != "" {
		var id int
		if _, err := fmt.Sscanf(pr, "%d", &id); err == nil {
			return id, nil
		}
	}

	return 0, fmt.Errorf("could not parse PR ID from Gitee environment")
}
