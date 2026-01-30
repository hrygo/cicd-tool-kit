// Package platform provides Jenkins integration for legacy CI environments
package platform

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// validJobNamePattern matches safe job names: alphanumeric, hyphen, underscore, dot
// Rejects path traversal attempts (..) and special characters
var validJobNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// sanitizeJobPath validates and sanitizes a job path to prevent directory traversal attacks.
// Returns the sanitized path and an error if the input contains unsafe characters.
func sanitizeJobPath(input string) (string, error) {
	// Reject empty input
	if input == "" {
		return "", fmt.Errorf("job path cannot be empty")
	}

	// Reject path traversal attempts explicitly
	if strings.Contains(input, "..") {
		return "", fmt.Errorf("job path cannot contain '..'")
	}

	// Reject URL-encoded traversal attempts
	if strings.Contains(input, "%2e") || strings.Contains(input, "%2E") {
		return "", fmt.Errorf("job path cannot contain URL-encoded dots")
	}

	// Reject absolute paths
	if strings.HasPrefix(input, "/") || strings.HasPrefix(input, "\\") {
		return "", fmt.Errorf("job path cannot be absolute")
	}

	// Validate against safe pattern
	if !validJobNamePattern.MatchString(input) {
		return "", fmt.Errorf("job path contains invalid characters")
	}

	// Use filepath.Base to extract just the filename, preventing any path components
	safe := filepath.Base(input)

	// Final safety check
	if safe == "." || safe == ".." || safe == "" {
		return "", fmt.Errorf("job path is invalid")
	}

	return safe, nil
}

// sanitizeFilePath validates a file path within a workspace, allowing subdirectories
// but preventing directory traversal attacks.
func sanitizeFilePath(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Check for null byte
	if strings.Contains(input, "\x00") {
		return "", fmt.Errorf("file path cannot contain null byte")
	}

	// Reject path traversal attempts explicitly
	if strings.Contains(input, "..") {
		return "", fmt.Errorf("file path cannot contain '..'")
	}

	// Reject URL-encoded traversal attempts (case-insensitive for %2e, %2E, %5c, %5C)
	lowerInput := strings.ToLower(input)
	if strings.Contains(lowerInput, "%2e") || strings.Contains(lowerInput, "%5c") {
		return "", fmt.Errorf("file path cannot contain URL-encoded dots or backslashes")
	}

	// Reject absolute paths
	if strings.HasPrefix(input, "/") || strings.HasPrefix(input, "\\") {
		return "", fmt.Errorf("file path cannot be absolute")
	}

	// Clean the path but don't use Base - we want to allow subdirectories
	clean := filepath.Clean(input)

	// Final safety check
	if clean == "." || clean == ".." || clean == "" {
		return "", fmt.Errorf("file path is invalid")
	}

	return clean, nil
}

// JenkinsClient provides integration with Jenkins CI/CD server
// Jenkins is a build automation server that can work with various Git platforms
type JenkinsClient struct {
	baseURL    string
	username   string
	apiToken   string
	jobName    string
	httpClient *http.Client
}

// JenkinsBuildInfo represents information about a Jenkins build
type JenkinsBuildInfo struct {
	Number          int    `json:"number"`
	URL             string `json:"url"`
	Building        bool   `json:"building"`
	Result          string `json:"result"`
	Timestamp       int64  `json:"timestamp"`
	Duration        int64  `json:"duration"`
	DisplayName     string `json:"displayName"`
	FullDisplayName string `json:"fullDisplayName"`
}

// JenkinsChange represents a change in a build
type JenkinsChange struct {
	CommitID string `json:"commitId"`
	Msg      string `json:"msg"`
	Author   string `json:"author"`
}

// JenkinsAction represents a build action (may contain PR info)
type JenkinsAction struct {
	LastBuiltRevision struct {
		SHA1 string `json:"SHA1"`
	} `json:"lastBuiltRevision"`
	Parameters []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"parameters"`
	Causes []struct {
		ShortDescription string `json:"shortDescription"`
	} `json:"causes"`
}

// JenkinsJob represents a Jenkins job
type JenkinsJob struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	URL         string `json:"url"`
	Builds      []struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	} `json:"builds"`
	LastBuild struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	} `json:"lastBuild"`
}

// NewJenkinsClient creates a new Jenkins client
func NewJenkinsClient(baseURL, username, apiToken, jobName string) (*JenkinsClient, error) {
	// SECURITY: Validate baseURL to prevent SSRF attacks
	if err := validateBaseURL(baseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Validate and sanitize jobName to prevent path traversal attacks
	cleanJobName, err := sanitizeJobPath(jobName)
	if err != nil {
		return nil, fmt.Errorf("invalid job name: %w", err)
	}

	return &JenkinsClient{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		username: username,
		apiToken: apiToken,
		jobName:  cleanJobName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Name returns the platform name
func (j *JenkinsClient) Name() string {
	return "jenkins"
}

// PostComment posts a comment to a Jenkins build
// For Jenkins, this posts a console note or adds to the build description
func (j *JenkinsClient) PostComment(ctx context.Context, opts CommentOptions) error {
	// In Jenkins, "comments" are typically:
	// 1. Console notes (added during build)
	// 2. Build description (added after build)
	// 3. Build log notes

	// We'll add a console note to the build
	// This requires the "Console Note" script or REST API

	// For now, we'll update the build description
	endpoint := fmt.Sprintf("%s/job/%s/%d/submitDescription", j.baseURL, j.jobName, opts.PRID)

	// SECURITY: Validate opts.Body for CRLF sequences before URL encoding
	// to prevent HTTP Response Splitting attacks
	if strings.Contains(opts.Body, "\r\n") || strings.Contains(opts.Body, "\n\r") ||
		strings.Contains(opts.Body, "\r") || strings.Contains(opts.Body, "\n") {
		return fmt.Errorf("comment body contains CRLF sequences which are not allowed")
	}

	// Properly URL-encode the form data to prevent injection
	data := fmt.Sprintf("description=%s", url.QueryEscape(opts.Body))
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(j.username, j.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetDiff retrieves the diff for a Jenkins build
// For Jenkins, we need to get the SCM change information
func (j *JenkinsClient) GetDiff(ctx context.Context, prID int) (string, error) {
	// Get the change set for this build
	url := fmt.Sprintf("%s/job/%s/%d/changeset", j.baseURL, j.jobName, prID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(j.username, j.apiToken)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get changeset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no changeset found for build %d", prID)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Jenkins changeset API returns HTML, we need to parse or use the API
	// For simplicity, return the change information
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Try to parse as JSON (some Jenkins APIs return JSON)
	var changeset struct {
		Changes []struct {
			Commit struct {
				Msg string `json:"msg"`
			} `json:"commit"`
		} `json:"changes"`
	}

	if err := json.Unmarshal(body, &changeset); err == nil {
		var result strings.Builder
		for _, change := range changeset.Changes {
			result.WriteString(change.Commit.Msg)
			result.WriteString("\n\n")
		}
		return result.String(), nil
	}

	// Return raw HTML as fallback
	return string(body), nil
}

// GetFile retrieves a file's content from the workspace of a build
func (j *JenkinsClient) GetFile(ctx context.Context, path, ref string) (string, error) {
	// Sanitize path to prevent directory traversal attacks
	cleanPath, err := sanitizeFilePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	// In Jenkins, we get the file from the workspace
	// The ref parameter is interpreted as a build number
	buildNumber := "lastSuccessfulBuild"
	if ref != "" {
		// Validate ref parameter - only allow specific build references or positive integers
		// This prevents path traversal or injection via the ref parameter
		validRefs := map[string]bool{
			"lastSuccessfulBuild": true,
			"lastCompletedBuild":  true,
			"lastFailedBuild":     true,
			"lastStableBuild":     true,
			"lastUnstableBuild":   true,
		}
		if !validRefs[ref] {
			// Check if it's a valid positive integer
			num := 0
			_, err := fmt.Sscanf(ref, "%d", &num)
			// SECURITY: Add upper bound check to prevent DoS via extremely large build numbers
			// Max int32 is a reasonable upper limit for Jenkins build numbers
			const maxBuildNumber = 2147483647 // max int32
			if err != nil || num <= 0 || num > maxBuildNumber {
				return "", fmt.Errorf("invalid build reference: %s (must be between 1 and %d)", ref, maxBuildNumber)
			}
		}
		buildNumber = ref
	}

	url := fmt.Sprintf("%s/job/%s/%s/ws/%s", j.baseURL, j.jobName, buildNumber, cleanPath)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(j.username, j.apiToken)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(content), nil
}

// GetPRInfo retrieves information about a Jenkins build
// Note: Jenkins uses "build number" instead of "PR ID"
func (j *JenkinsClient) GetPRInfo(ctx context.Context, prID int) (*PRInfo, error) {
	buildInfo, err := j.getBuildInfo(ctx, prID)
	if err != nil {
		return nil, err
	}

	// Extract PR information from build parameters or causes
	prInfo := &PRInfo{
		Number: prID,
		Title:  buildInfo.DisplayName,
		SHA:    j.getBuildSHA(buildInfo),
	}

	// Try to get PR info from parameters
	for _, action := range buildInfo.Actions {
		for _, param := range action.Parameters {
			switch param.Name {
			case "ghprbPullTitle", "pull_request_title":
				prInfo.Title = param.Value
			case "ghprbPullId", "pull_request_id":
				// Override build number with actual PR ID if available
				if param.Value != "" {
					fmt.Sscanf(param.Value, "%d", &prInfo.Number)
				}
			case "ghprbSourceBranch", "source_branch":
				prInfo.HeadBranch = param.Value
			case "ghprbTargetBranch", "target_branch":
				prInfo.BaseBranch = param.Value
			case "ghprbPullAuthor", "pull_request_author":
				prInfo.Author = param.Value
			}
		}

		// Get description from cause if available
		for _, cause := range action.Causes {
			if cause.ShortDescription != "" {
				if prInfo.Description == "" {
					prInfo.Description = cause.ShortDescription
				}
			}
		}
	}

	return prInfo, nil
}

// Health checks if the Jenkins API is accessible
func (j *JenkinsClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/json", j.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(j.username, j.apiToken)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Jenkins: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jenkins health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// getBuildInfo retrieves detailed information about a specific build
func (j *JenkinsClient) getBuildInfo(ctx context.Context, buildNumber int) (*jenkinsBuildInfoFull, error) {
	url := fmt.Sprintf("%s/job/%s/%d/api/json", j.baseURL, j.jobName, buildNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(j.username, j.apiToken)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get build info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("build %d not found", buildNumber)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var buildInfo jenkinsBuildInfoFull
	if err := json.NewDecoder(resp.Body).Decode(&buildInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &buildInfo, nil
}

// jenkinsBuildInfoFull represents the full build info with actions
type jenkinsBuildInfoFull struct {
	JenkinsBuildInfo
	Actions []struct {
		LastBuiltRevision struct {
			SHA1 string `json:"SHA1"`
		} `json:"lastBuiltRevision"`
		Parameters []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"parameters"`
		Causes []struct {
			ShortDescription string `json:"shortDescription"`
		} `json:"causes"`
	} `json:"actions"`
}

// getBuildSHA extracts the commit SHA from build info
func (j *JenkinsClient) getBuildSHA(buildInfo *jenkinsBuildInfoFull) string {
	for _, action := range buildInfo.Actions {
		if action.LastBuiltRevision.SHA1 != "" {
			return action.LastBuiltRevision.SHA1
		}
	}
	return ""
}

// GetJob retrieves information about the Jenkins job
func (j *JenkinsClient) GetJob(ctx context.Context) (*JenkinsJob, error) {
	url := fmt.Sprintf("%s/job/%s/api/json", j.baseURL, j.jobName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(j.username, j.apiToken)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get job info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("job %s not found", j.jobName)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var job JenkinsJob
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &job, nil
}

// TriggerBuild triggers a new build of the Jenkins job
func (j *JenkinsClient) TriggerBuild(ctx context.Context, parameters map[string]string) (int, error) {
	endpoint := fmt.Sprintf("%s/job/%s/buildWithParameters", j.baseURL, j.jobName)

	// Block CI system reserved prefixes to prevent environment variable injection
	blockedPrefixes := []string{
		"GITHUB_", "GITLAB_", "BITBUCKET_", "JENKINS_",
		"CI_", "BUILD_", "JOB_", "EXECUTOR_", "NODE_",
		"WORKSPACE_", "SVN_", "CVS_",
	}

	// Limit maximum number of parameters to prevent DoS
	const maxParams = 50
	if len(parameters) > maxParams {
		return 0, fmt.Errorf("too many parameters: maximum %d allowed", maxParams)
	}

	// Validate parameters to prevent injection
	for key, value := range parameters {
		// Check for blocked prefixes
		for _, prefix := range blockedPrefixes {
			if strings.HasPrefix(key, prefix) {
				return 0, fmt.Errorf("parameter key '%s' has blocked prefix '%s'", key, prefix)
			}
		}

		// Check for dangerous characters in parameter keys and values
		dangerousChars := []string{"\n", "\r", "\t", "\x00"}
		for _, ch := range dangerousChars {
			if strings.Contains(key, ch) || strings.Contains(value, ch) {
				return 0, fmt.Errorf("parameter contains dangerous character")
			}
		}
		// Limit parameter length to prevent DoS
		if len(key) > 256 || len(value) > 4096 {
			return 0, fmt.Errorf("parameter too large")
		}
	}

	// Create form data with parameters, properly URL-encoded to prevent injection
	var formData bytes.Buffer
	for key, value := range parameters {
		if formData.Len() > 0 {
			formData.WriteString("&")
		}
		fmt.Fprintf(&formData, "%s=%s", url.QueryEscape(key), url.QueryEscape(value))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, &formData)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(j.username, j.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to trigger build: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, fmt.Errorf("job %s not found or doesn't support parameters", j.jobName)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Get the queue location from response headers
	location := resp.Header.Get("Location")
	if location == "" {
		// Try to get the last build number
		job, err := j.GetJob(ctx)
		if err != nil {
			return 0, fmt.Errorf("build triggered but couldn't get build number: %w", err)
		}
		return job.LastBuild.Number, nil
	}

	// Extract build number from location URL
	// Location is typically: /job/{jobName}/{buildNumber}/
	// Search from the end to handle cases where jobName appears multiple times
	parts := strings.Split(location, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == j.jobName && i+1 < len(parts) {
			var buildNum int
			n, err := fmt.Sscanf(parts[i+1], "%d", &buildNum)
			if n != 1 || err != nil || buildNum <= 0 {
				return 0, fmt.Errorf("invalid build number from location: %s", parts[i+1])
			}
			return buildNum, nil
		}
	}

	return 0, fmt.Errorf("couldn't extract build number from location")
}

// GetBuildStatus retrieves the current status of a build
func (j *JenkinsClient) GetBuildStatus(ctx context.Context, buildNumber int) (string, bool, error) {
	buildInfo, err := j.getBuildInfo(ctx, buildNumber)
	if err != nil {
		return "", false, err
	}

	return buildInfo.Result, buildInfo.Building, nil
}

// GetBuildLog retrieves the console output for a build
func (j *JenkinsClient) GetBuildLog(ctx context.Context, buildNumber int) (string, error) {
	url := fmt.Sprintf("%s/job/%s/%d/consoleText", j.baseURL, j.jobName, buildNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(j.username, j.apiToken)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get build log: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(log), nil
}

// SetBuildResult sets the result of a build (for build scripts)
// This is typically done by the build itself, but can be used for external status updates
func (j *JenkinsClient) SetBuildResult(ctx context.Context, buildNumber int, result string, message string) error {
	// Jenkins doesn't have a direct API to set build results from outside
	// This would typically use the Jenkins "build description" or "console note" APIs
	return j.PostComment(ctx, CommentOptions{
		PRID: buildNumber,
		Body: fmt.Sprintf("Build Result: %s\n\n%s", result, message),
	})
}

// CreateCrumb creates a CSRF crumb for Jenkins API requests
// Jenkins requires CSRF crumbs for POST requests in newer versions
func (j *JenkinsClient) CreateCrumb(ctx context.Context) (string, string, error) {
	url := fmt.Sprintf("%s/crumbIssuer/api/json", j.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(j.username, j.apiToken)

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to get crumb: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// CSRF might be disabled
		return "", "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var crumb struct {
		Crumb             string `json:"crumb"`
		CrumbRequestField string `json:"crumbRequestField"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&crumb); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	return crumb.CrumbRequestField, crumb.Crumb, nil
}

// JenkinsWebhook represents a Jenkins webhook payload
type JenkinsWebhook struct {
	BuildName   string `json:"name"`
	BuildURL    string `json:"url"`
	BuildNumber int    `json:"number"`
	Phase       string `json:"phase"`
	Status      string `json:"status"`
	URL         string `json:"buildUrl"`
}

// ParseWebhook parses a Jenkins webhook notification
func ParseJenkinsWebhook(authToken string, handler func(ctx context.Context, webhook *JenkinsWebhook) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify auth token if provided (using constant-time comparison to prevent timing attacks)
		if authToken != "" {
			receivedToken := r.Header.Get("Authorization")
			if receivedToken == "" {
				receivedToken = r.URL.Query().Get("token")
			}
			// Strip "Bearer " prefix if present (common auth header format)
			receivedToken = strings.TrimPrefix(receivedToken, "Bearer ")
			// SECURITY: Use only subtle.ConstantTimeCompare for timing-attack-safe comparison
			// Do NOT check length first - subtle.ConstantTimeCompare handles this internally
			// and a separate length check would leak timing information
			if subtle.ConstantTimeCompare([]byte(receivedToken), []byte(authToken)) != 1 {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Limit request body size to prevent memory exhaustion attacks (max 1MB)
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var webhook JenkinsWebhook
		if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		if err := handler(ctx, &webhook); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// JenkinsBasicAuth creates a basic auth header value
func JenkinsBasicAuth(username, apiToken string) string {
	auth := username + ":" + apiToken
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
