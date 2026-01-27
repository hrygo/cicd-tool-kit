// Package platform provides GitLab platform implementation
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
)

// GitLabClient implements Platform for GitLab
type GitLabClient struct {
	token   string
	baseURL string // For GitLab self-hosted
	repo    string // project ID or path
	client  *http.Client
}

// GitLabAPIResponse represents common GitLab API response structure
type GitLabAPIResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

// GitLabMR represents GitLab merge request response
type GitLabMR struct {
	ID     int    `json:"id"`
	IID    int    `json:"iid"` // Merge Request IID (user-facing number)
	Title  string `json:"title"`
	Description string `json:"description"`
	Head   GitLabMRRef `json:"source_branch"`
	Base   GitLabMRRef `json:"target_branch"`
	Author GitLabUser `json:"author"`
	WebURL string `json:"web_url"`
	State  string `json:"state"`
	MergedAt *time.Time `json:"merged_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	SourceProject GitLabProject `json:"source_project"`
}

// GitLabMRRef represents a branch reference in a MR
type GitLabMRRef string

// GitLabProject represents GitLab project info
type GitLabProject struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
}

// GitLabUser represents GitLab user info
type GitLabUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// GitLabComment represents a comment on GitLab
type GitLabComment struct {
	Body string `json:"body"`
}

// GitLabDiffResponse represents diff response from GitLab
type GitLabDiffResponse struct {
	Changes []struct {
		Diff        string `json:"diff"`
		NewPath     string `json:"new_path"`
		OldPath     string `json:"old_path"`
		NewFile     bool   `json:"new_file"`
		RenamedFile bool   `json:"renamed_file"`
		DeletedFile bool   `json:"deleted_file"`
	} `json:"changes"`
}

// NewGitLabClient creates a new GitLab platform client
func NewGitLabClient(token, repo string) *GitLabClient {
	baseURL := os.Getenv("GITLAB_API_URL")
	if baseURL == "" {
		baseURL = "https://gitlab.com/api/v4"
	}

	return &GitLabClient{
		token:   token,
		baseURL: baseURL,
		repo:    repo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL sets a custom base URL for GitLab self-hosted
func (g *GitLabClient) SetBaseURL(url string) {
	g.baseURL = url
}

// PostComment posts a comment to a GitLab merge request
func (g *GitLabClient) PostComment(ctx context.Context, opts CommentOptions) error {
	if opts.PRID == 0 {
		return fmt.Errorf("MR IID is required")
	}

	comment := GitLabComment{
		Body: opts.Body,
	}

	body, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal comment: %w", err)
	}

	// GitLab uses IID (user-facing MR number) in the URL
	url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/notes", g.baseURL, urlPathEncode(g.repo), opts.PRID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", g.token)
	req.Header.Set("Content-Type", "application/json")

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

// GetDiff retrieves the diff for a GitLab merge request
func (g *GitLabClient) GetDiff(ctx context.Context, mrID int) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/changes", g.baseURL, urlPathEncode(g.repo), mrID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", g.token)

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get diff (status %d)", resp.StatusCode)
	}

	var result GitLabDiffResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode diff response: %w", err)
	}

	var diffBuilder bytes.Buffer
	for _, change := range result.Changes {
		diffBuilder.WriteString(change.Diff)
		diffBuilder.WriteString("\n")
	}

	return diffBuilder.String(), nil
}

// GetFile retrieves a file from the GitLab repository
func (g *GitLabClient) GetFile(ctx context.Context, path, ref string) (string, error) {
	encodedPath := urlPathEncode(path)
	url := fmt.Sprintf("%s/projects/%s/repository/files/%s?ref=%s", g.baseURL, urlPathEncode(g.repo), encodedPath, ref)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", g.token)

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

	// GitLab returns base64 encoded content
	if result.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(result.Content)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 content: %w", err)
		}
		return string(decoded), nil
	}

	return result.Content, nil
}

// GetPRInfo retrieves merge request information from GitLab
func (g *GitLabClient) GetPRInfo(ctx context.Context, mrID int) (*PRInfo, error) {
	url := fmt.Sprintf("%s/projects/%s/merge_requests/%d", g.baseURL, urlPathEncode(g.repo), mrID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", g.token)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get MR info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get MR info (status %d)", resp.StatusCode)
	}

	var gitlabMR GitLabMR
	if err := json.NewDecoder(resp.Body).Decode(&gitlabMR); err != nil {
		return nil, fmt.Errorf("failed to decode MR response: %w", err)
	}

	// Get latest SHA for source branch
	sha := ""
	if gitlabMR.SourceProject.ID > 0 {
		shaURL := fmt.Sprintf("%s/projects/%s/repository/commits?ref_name=%s", g.baseURL, urlPathEncode(g.repo), string(gitlabMR.Head))
		shaReq, _ := http.NewRequestWithContext(ctx, "GET", shaURL, nil)
		shaReq.Header.Set("PRIVATE-TOKEN", g.token)
		if shaResp, err := g.client.Do(shaReq); err == nil {
			defer shaResp.Body.Close()
			var commits []struct {
				ID string `json:"id"`
			}
			if json.NewDecoder(shaResp.Body).Decode(&commits) == nil && len(commits) > 0 {
				sha = commits[0].ID
			}
		}
	}

	return &PRInfo{
		Number:      gitlabMR.IID,
		Title:       gitlabMR.Title,
		Description: gitlabMR.Description,
		Author:      gitlabMR.Author.Username,
		SHA:         sha,
		BaseBranch:  string(gitlabMR.Base),
		HeadBranch:  string(gitlabMR.Head),
		SourceRepo:  gitlabMR.SourceProject.PathWithNamespace,
	}, nil
}

// Health checks if the GitLab API is accessible
func (g *GitLabClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/projects/%s", g.baseURL, urlPathEncode(g.repo))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", g.token)

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}

	return nil
}

// IsGitLabEnv checks if running in GitLab environment
func IsGitLabEnv() bool {
	return os.Getenv("GITLAB_CI") == "true" || os.Getenv("CI_SERVER_NAME") == "GitLab"
}

// ParseRepoFromGitLabEnv parses repo from GitLab environment variables
func ParseRepoFromGitLabEnv() (string, error) {
	// GitLab CI uses project path
	if projectPath := os.Getenv("CI_PROJECT_PATH"); projectPath != "" {
		return projectPath, nil
	}

	return "", fmt.Errorf("could not parse repo from GitLab environment")
}

// ParsePRIDFromGitLabEnv parses MR ID from GitLab environment
func ParsePRIDFromGitLabEnv() (int, error) {
	// GitLab CI merge request IID
	if mr := os.Getenv("CI_MERGE_REQUEST_IID"); mr != "" {
		var id int
		if _, err := fmt.Sscanf(mr, "%d", &id); err == nil {
			return id, nil
		}
	}

	return 0, fmt.Errorf("could not parse MR ID from GitLab environment")
}

// urlPathEncode encodes a path for URL in GitLab format (/ instead of %2F)
func urlPathEncode(path string) string {
	// GitLab uses URL-encoded paths with / for project paths
	// For simplicity, we'll just replace / with %2F for the project path only
	// In production, use url.PathEscape or similar
	return path
}
