// Package platform provides Gitee enterprise-specific features
package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// CodeOwnerEntry represents a CODEOWNERS entry
type CodeOwnerEntry struct {
	Pattern  string   `json:"pattern"`  // File pattern (e.g., "*.go", "/pkg/**")
	Owners   []string `json:"owners"`   // List of owner usernames
	Approved bool     `json:"approved"` // Whether approval is granted
}

// CodeOwnersFile represents the parsed CODEOWNERS file
type CodeOwnersFile struct {
	Entries []CodeOwnerEntry `json:"entries"`
}

// BranchProtectionRule represents a branch protection rule
type BranchProtectionRule struct {
	Name            string   `json:"name"`
	Pattern         string   `json:"pattern"` // Branch name pattern
	RequireApproval bool     `json:"require_approval"`
	ApprovalCount   int      `json:"approval_count"`
	RequireStatus   bool     `json:"require_status"`
	StatusContexts  []string `json:"status_contexts"`
	AllowForcePush  bool     `json:"allow_force_push"`
}

// SecurityScanResult represents a security scan result from GiteeScan
type SecurityScanResult struct {
	ID        int             `json:"id"`
	Tool      string          `json:"tool"`   // sast, license, duplication
	Status    string          `json:"status"` // passed, failed, running
	Summary   string          `json:"summary"`
	Issues    []SecurityIssue `json:"issues"`
	ReportURL string          `json:"report_url"`
	CreatedAt string          `json:"created_at"`
}

// SecurityIssue represents a single security issue
type SecurityIssue struct {
	Severity    string `json:"severity"` // critical, high, medium, low
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	CWE         string `json:"cwe,omitempty"` // CWE identifier if applicable
}

// CodeQualityMetrics represents code quality metrics
type CodeQualityMetrics struct {
	Coverage           float64 `json:"coverage"`    // Test coverage percentage
	Duplication        float64 `json:"duplication"` // Code duplication percentage
	Complexity         float64 `json:"complexity"`  // Cyclomatic complexity
	CodeSmellCount     int     `json:"code_smell_count"`
	BugCount           int     `json:"bug_count"`
	VulnerabilityCount int     `json:"vulnerability_count"`
}

// GetCodeOwners retrieves the CODEOWNERS file from the repository
func (g *GiteeClient) GetCodeOwners(ctx context.Context, ref string) (*CodeOwnersFile, error) {
	// Try common locations for CODEOWNERS file
	locations := []string{
		".gitee/CODEOWNERS",
		"CODEOWNERS",
		".github/CODEOWNERS",
		"docs/CODEOWNERS",
	}

	for _, loc := range locations {
		content, err := g.GetFile(ctx, loc, ref)
		if err == nil {
			return g.parseCodeOwners(content)
		}
	}

	return nil, fmt.Errorf("CODEOWNERS file not found in repository")
}

// parseCodeOwners parses CODEOWNERS file content
func (g *GiteeClient) parseCodeOwners(content string) (*CodeOwnersFile, error) {
	var file CodeOwnersFile
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse line: pattern @owner1 @owner2 ...
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pattern := parts[0]
		var owners []string
		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "@") {
				owners = append(owners, strings.TrimPrefix(part, "@"))
			}
		}

		if len(owners) > 0 {
			file.Entries = append(file.Entries, CodeOwnerEntry{
				Pattern: pattern,
				Owners:  owners,
			})
		}
	}

	return &file, nil
}

// GetRequiredOwnersForFile returns required owners for a given file path
func (g *GiteeClient) GetRequiredOwnersForFile(ctx context.Context, filePath, ref string) ([]string, error) {
	codeOwners, err := g.GetCodeOwners(ctx, ref)
	if err != nil {
		return nil, err
	}

	var requiredOwners []string

	// Find matching patterns (last match wins in CODEOWNERS)
	var lastMatch *CodeOwnerEntry
	for _, entry := range codeOwners.Entries {
		if g.patternMatches(entry.Pattern, filePath) {
			lastMatch = &entry
		}
	}

	if lastMatch != nil {
		requiredOwners = lastMatch.Owners
	}

	return requiredOwners, nil
}

// patternMatches checks if a file pattern matches a file path
func (g *GiteeClient) patternMatches(pattern, filePath string) bool {
	// Simple glob matching
	if pattern == "*" {
		return true
	}

	// Handle /pkg/** style patterns (root anchored directory match)
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimPrefix(pattern, "/")
		prefix = strings.TrimSuffix(prefix, "/**")
		return strings.HasPrefix(filePath, prefix+"/") || filePath == prefix
	}

	// Exact match from root (e.g., /README.md)
	if strings.HasPrefix(pattern, "/") && !strings.Contains(pattern, "*") {
		return filePath == strings.TrimPrefix(pattern, "/")
	}

	// Directory match without root anchor (e.g., pkg/**)
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(filePath, prefix+"/") || filePath == prefix
	}

	if strings.Contains(pattern, "*") {
		// Simple wildcard
		patternParts := strings.Split(pattern, "*")
		if len(patternParts) == 2 {
			return strings.HasPrefix(filePath, patternParts[0]) &&
				strings.HasSuffix(filePath, patternParts[1])
		}
	}

	// Extension match
	if strings.HasPrefix(pattern, "*.") {
		ext := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(filePath, ext)
	}

	return filePath == pattern
}

// GetCodeOwnerApprovals retrieves approval status for a PR
func (g *GiteeClient) GetCodeOwnerApprovals(ctx context.Context, prID int) ([]CodeOwnerEntry, error) {
	// Get PR info to find the head ref
	prInfo, err := g.GetPRInfo(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR info: %w", err)
	}

	// Get CODEOWNERS file
	codeOwners, err := g.GetCodeOwners(ctx, prInfo.HeadBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get CODEOWNERS: %w", err)
	}

	// Get changed files in the PR
	diff, err := g.GetDiff(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Map required owners per changed file
	requiredOwnersMap := make(map[string][]string)
	for _, entry := range codeOwners.Entries {
		// Check which files match this pattern
		lines := strings.Split(diff, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "diff --git a/") {
				// Extract file path from diff line
				parts := strings.Fields(line)
				if len(parts) >= 4 {
					filePath := strings.TrimPrefix(parts[3], "b/")
					if g.patternMatches(entry.Pattern, filePath) {
						requiredOwnersMap[filePath] = append(requiredOwnersMap[filePath], entry.Owners...)
					}
				}
			}
		}
	}

	// Build results with approval status (simplified - in real implementation, check actual approvals)
	var results []CodeOwnerEntry
	for filePath, owners := range requiredOwnersMap {
		results = append(results, CodeOwnerEntry{
			Pattern:  filePath,
			Owners:   owners,
			Approved: false, // Would need actual approval API call
		})
	}

	return results, nil
}

// GetBranchProtection retrieves branch protection rules
func (g *GiteeClient) GetBranchProtection(ctx context.Context, branch string) (*BranchProtectionRule, error) {
	// Gitee API v5 doesn't expose branch protection directly
	// This is a placeholder for future API support
	// In enterprise deployments, this would call the appropriate endpoint

	apiURL := fmt.Sprintf("%s/repos/%s/branch_protection_rules/%s",
		g.baseURL, url.QueryEscape(g.repo), url.QueryEscape(branch))

	resp, err := g.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch protection: %w", err)
	}
	defer resp.Body.Close()

	// API may not be available, return default rule
	if resp.StatusCode == http.StatusNotFound {
		return &BranchProtectionRule{
			Name:            branch,
			Pattern:         branch,
			RequireApproval: false,
			AllowForcePush:  true,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get branch protection (status %d)", resp.StatusCode)
	}

	var rule BranchProtectionRule
	if err := json.NewDecoder(resp.Body).Decode(&rule); err != nil {
		return nil, fmt.Errorf("failed to decode branch protection: %w", err)
	}

	return &rule, nil
}

// GetSecurityScanResults retrieves security scan results for a commit
func (g *GiteeClient) GetSecurityScanResults(ctx context.Context, sha string) (*SecurityScanResult, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/code-check/%s",
		g.baseURL, url.QueryEscape(g.repo), url.QueryEscape(sha))

	resp, err := g.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get security scan results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get security scan results (status %d)", resp.StatusCode)
	}

	var result SecurityScanResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode security scan results: %w", err)
	}

	return &result, nil
}

// TriggerSecurityScan triggers a security scan for a commit
func (g *GiteeClient) TriggerSecurityScan(ctx context.Context, sha string, scanTypes []string) error {
	// GiteeScan scan types: sast, license, duplication
	payload := map[string]interface{}{
		"sha":        sha,
		"scan_types": scanTypes,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal scan request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/code-check",
		g.baseURL, url.QueryEscape(g.repo))

	resp, err := g.doRequest(ctx, "POST", apiURL, body)
	if err != nil {
		return fmt.Errorf("failed to trigger security scan: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to trigger security scan (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetCodeQualityMetrics retrieves code quality metrics for a commit
func (g *GiteeClient) GetCodeQualityMetrics(ctx context.Context, sha string) (*CodeQualityMetrics, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/code-metrics/%s",
		g.baseURL, url.QueryEscape(g.repo), url.QueryEscape(sha))

	resp, err := g.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get code quality metrics: %w", err)
	}
	defer resp.Body.Close()

	// Metrics may not be available
	if resp.StatusCode == http.StatusNotFound {
		return &CodeQualityMetrics{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get code quality metrics (status %d)", resp.StatusCode)
	}

	var metrics CodeQualityMetrics
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return nil, fmt.Errorf("failed to decode code quality metrics: %w", err)
	}

	return &metrics, nil
}

// ReviewerSuggestion represents a suggested code reviewer
type ReviewerSuggestion struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Reason      string   `json:"reason"`       // code-owner, recent-author, domain-expert
	Score       float64  `json:"score"`        // Confidence score 0-1
	FileCount   int      `json:"file_count"`   // Number of files they own
	RecentFiles []string `json:"recent_files"` // Recently modified files
}

// GetReviewerSuggestions suggests reviewers for a PR based on CODEOWNERS and history
func (g *GiteeClient) GetReviewerSuggestions(ctx context.Context, prID int) ([]ReviewerSuggestion, error) {
	prInfo, err := g.GetPRInfo(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR info: %w", err)
	}

	// Get CODEOWNERS
	codeOwners, err := g.GetCodeOwners(ctx, prInfo.BaseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get CODEOWNERS: %w", err)
	}

	// Get diff to find changed files
	diff, err := g.GetDiff(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Build suggestions based on CODEOWNERS
	suggestionsMap := make(map[string]*ReviewerSuggestion)

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git a/") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				filePath := strings.TrimPrefix(parts[3], "b/")

				// Find matching code owners
				for _, entry := range codeOwners.Entries {
					if g.patternMatches(entry.Pattern, filePath) {
						for _, owner := range entry.Owners {
							if _, exists := suggestionsMap[owner]; !exists {
								suggestionsMap[owner] = &ReviewerSuggestion{
									Username: owner,
									Reason:   "code-owner",
									Score:    1.0,
								}
							}
							s := suggestionsMap[owner]
							s.FileCount++
							s.RecentFiles = append(s.RecentFiles, filePath)
						}
					}
				}
			}
		}
	}

	// Convert map to slice
	var suggestions []ReviewerSuggestion
	for _, s := range suggestionsMap {
		suggestions = append(suggestions, *s)
	}

	return suggestions, nil
}

// EnterpriseInfo represents enterprise information
type EnterpriseInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	LogoURL     string `json:"logo_url"`
	// Compliance features
	Level3Security bool `json:"level3_security"` // 等保三级
	PasswordEval   bool `json:"password_eval"`   // 密评
	Xinchuang      bool `json:"xinchuang"`       // 信创
}

// GetEnterpriseInfo retrieves enterprise information
func (g *GiteeClient) GetEnterpriseInfo(ctx context.Context) (*EnterpriseInfo, error) {
	// Extract enterprise slug from baseURL if using enterprise
	if !strings.Contains(g.baseURL, "gitee.com") {
		// Self-hosted or enterprise instance
		parts := strings.Split(g.baseURL, ".")
		if len(parts) > 1 {
			slug := strings.Split(parts[0], "://")[1]
			return &EnterpriseInfo{
				Slug: slug,
				Name: slug,
			}, nil
		}
	}

	// For public Gitee, try to get enterprise info from repo
	apiURL := fmt.Sprintf("%s/repos/%s", g.baseURL, url.QueryEscape(g.repo))

	resp, err := g.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get enterprise info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get enterprise info (status %d)", resp.StatusCode)
	}

	var result struct {
		Enterprise *EnterpriseInfo `json:"enterprise"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode enterprise info: %w", err)
	}

	if result.Enterprise == nil {
		return &EnterpriseInfo{}, nil
	}

	return result.Enterprise, nil
}

// GiteeGoConfig represents Gitee Go pipeline configuration
type GiteeGoConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	YAML        string            `json:"yaml"`      // Pipeline YAML content
	Variables   map[string]string `json:"variables"` // Environment variables
	Enabled     bool              `json:"enabled"`
}

// CreatePipeline creates a new Gitee Go pipeline
func (g *GiteeClient) CreatePipeline(ctx context.Context, config GiteeGoConfig) error {
	payload := map[string]interface{}{
		"name":        config.Name,
		"description": config.Description,
		"yaml":        config.YAML,
		"variables":   config.Variables,
		"enabled":     config.Enabled,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline config: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/pipelines",
		g.baseURL, url.QueryEscape(g.repo))

	resp, err := g.doRequest(ctx, "POST", apiURL, body)
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create pipeline (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetPipelineStatus retrieves the status of a pipeline run
func (g *GiteeClient) GetPipelineStatus(ctx context.Context, pipelineID, runID int) (string, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/pipelines/%d/runs/%d",
		g.baseURL, url.QueryEscape(g.repo), pipelineID, runID)

	resp, err := g.doRequest(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get pipeline status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get pipeline status (status %d)", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"` // pending, running, success, failed
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode pipeline status: %w", err)
	}

	return result.Status, nil
}

// TriggerPipeline manually triggers a pipeline run
func (g *GiteeClient) TriggerPipeline(ctx context.Context, pipelineID int, branch string) (int, error) {
	payload := map[string]string{
		"ref": branch,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal trigger request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/repos/%s/pipelines/%d/runs",
		g.baseURL, url.QueryEscape(g.repo), pipelineID)

	resp, err := g.doRequest(ctx, "POST", apiURL, body)
	if err != nil {
		return 0, fmt.Errorf("failed to trigger pipeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to trigger pipeline (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode trigger response: %w", err)
	}

	return result.ID, nil
}

// GetComplianceReport generates a compliance report for the repository
func (g *GiteeClient) GetComplianceReport(ctx context.Context) (*ComplianceReport, error) {
	// Get enterprise info
	enterprise, err := g.GetEnterpriseInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get enterprise info: %w", err)
	}

	report := &ComplianceReport{
		Enterprise:          enterprise,
		HasCodeOwners:       false,
		HasBranchProtection: false,
		HasSecurityScan:     false,
		HasStatusChecks:     false,
	}

	// Check for CODEOWNERS
	if _, err := g.GetCodeOwners(ctx, "main"); err == nil {
		report.HasCodeOwners = true
	}

	// Check for branch protection
	if _, err := g.GetBranchProtection(ctx, "main"); err == nil {
		report.HasBranchProtection = true
	}

	// Check security scan capability
	if enterprise.Level3Security || enterprise.PasswordEval {
		report.HasSecurityScan = true
	}

	// Status checks are available if status API works
	report.HasStatusChecks = true

	return report, nil
}

// ComplianceReport represents a compliance status report
type ComplianceReport struct {
	Enterprise          *EnterpriseInfo `json:"enterprise"`
	HasCodeOwners       bool            `json:"has_code_owners"`
	HasBranchProtection bool            `json:"has_branch_protection"`
	HasSecurityScan     bool            `json:"has_security_scan"`
	HasStatusChecks     bool            `json:"has_status_checks"`
	ComplianceScore     float64         `json:"compliance_score"`
	Recommendations     []string        `json:"recommendations"`
}
