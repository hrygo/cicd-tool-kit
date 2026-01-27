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
	"time"
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

// NewGiteeClient creates a new Gitee platform client
func NewGiteeClient(token, repo string) *GiteeClient {
	baseURL := os.Getenv("GITEE_API_URL")
	if baseURL == "" {
		baseURL = "https://gitee.com/api/v5"
	}

	return &GiteeClient{
		token:   token,
		baseURL: baseURL,
		repo:    repo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL sets a custom base URL for Gitee Enterprise
func (g *GiteeClient) SetBaseURL(url string) {
	g.baseURL = url
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
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "cicd-ai-toolkit/1.0")

	resp, err := g.client.Do(req)
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

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	req.Header.Set("User-Agent", "cicd-ai-toolkit/1.0")

	resp, err := g.client.Do(req)
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
			diffBuilder.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file.Filename, file.Filename))
			diffBuilder.WriteString(file.Patch)
			diffBuilder.WriteString("\n")
		}
	}

	return diffBuilder.String(), nil
}

// GetFile retrieves a file from the Gitee repository
func (g *GiteeClient) GetFile(ctx context.Context, path, ref string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", g.baseURL, url.QueryEscape(g.repo), url.QueryEscape(path), ref)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	req.Header.Set("User-Agent", "cicd-ai-toolkit/1.0")

	resp, err := g.client.Do(req)
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

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	req.Header.Set("User-Agent", "cicd-ai-toolkit/1.0")

	resp, err := g.client.Do(req)
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

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	req.Header.Set("User-Agent", "cicd-ai-toolkit/1.0")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Gitee API returned status %d", resp.StatusCode)
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
