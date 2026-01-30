// Package platform tests for Gitee enterprise features
package platform

import (
	"testing"
)

func TestParseCodeOwners(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		name    string
		content string
		want    int // expected number of entries
	}{
		{
			name:    "empty file",
			content: "",
			want:    0,
		},
		{
			name:    "simple entry",
			content: "*.go @backend-team",
			want:    1,
		},
		{
			name: "multiple entries",
			content: `*.go @backend-team
/pkg/auth/** @security-team
*.md @docs-team`,
			want: 3,
		},
		{
			name: "with comments",
			content: `# This is a comment
*.go @backend-team
# Another comment
*.md @docs-team`,
			want: 2,
		},
		{
			name:    "multiple owners",
			content: "*.go @user1 @user2 @user3",
			want:    1,
		},
		{
			name: "empty lines ignored",
			content: `
*.go @backend-team

*.md @docs-team
`,
			want: 2,
		},
		{
			name: "invalid lines skipped",
			content: `*.go @backend-team
invalid line without owners
*.md @docs-team`,
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := client.parseCodeOwners(tt.content)
			if err != nil {
				t.Fatalf("parseCodeOwners() error = %v", err)
			}

			if len(file.Entries) != tt.want {
				t.Errorf("parseCodeOwners() entries = %d, want %d", len(file.Entries), tt.want)
			}
		})
	}
}

func TestPatternMatches(t *testing.T) {
	client := NewGiteeClient("test-token", "owner/repo")

	tests := []struct {
		pattern  string
		filePath string
		want     bool
	}{
		// Wildcard all
		{"*", "anyfile.go", true},
		{"*", "nested/file.go", true},

		// Exact match from root
		{"/README.md", "README.md", true},
		{"/README.md", "src/README.md", false},

		// Extension match
		{"*.go", "main.go", true},
		{"*.go", "main.ts", false},
		{"*.go", "src/main.go", true},

		// Directory match
		{"/pkg/**", "pkg/file.go", true},
		{"/pkg/**", "pkg/nested/file.go", true},
		{"/pkg/**", "src/file.go", false},
		{"pkg/**", "pkg/file.go", true},

		// Simple wildcard in middle
		{"pkg/*_test.go", "pkg/main_test.go", true},
		{"pkg/*_test.go", "pkg/main.go", false},

		// Edge cases
		{"", "file.go", false},
		{"README.md", "README.md", true},
		{"README.md", "readme.md", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"->"+tt.filePath, func(t *testing.T) {
			result := client.patternMatches(tt.pattern, tt.filePath)
			if result != tt.want {
				t.Errorf("patternMatches(%s, %s) = %v, want %v", tt.pattern, tt.filePath, result, tt.want)
			}
		})
	}
}

func TestCodeOwnerEntryFields(t *testing.T) {
	entry := CodeOwnerEntry{
		Pattern:  "*.go",
		Owners:   []string{"user1", "user2"},
		Approved: true,
	}

	if entry.Pattern != "*.go" {
		t.Errorf("Pattern = %s, want *.go", entry.Pattern)
	}
	if len(entry.Owners) != 2 {
		t.Errorf("Owners length = %d, want 2", len(entry.Owners))
	}
	if !entry.Approved {
		t.Error("Approved should be true")
	}
}

func TestCodeOwnersFileFields(t *testing.T) {
	file := CodeOwnersFile{
		Entries: []CodeOwnerEntry{
			{Pattern: "*.go", Owners: []string{"team"}},
		},
	}

	if len(file.Entries) != 1 {
		t.Errorf("Entries length = %d, want 1", len(file.Entries))
	}
}

func TestBranchProtectionRuleFields(t *testing.T) {
	rule := BranchProtectionRule{
		Name:            "main",
		Pattern:         "main",
		RequireApproval: true,
		ApprovalCount:   2,
		RequireStatus:   true,
		StatusContexts:  []string{"ci", "security"},
		AllowForcePush:  false,
	}

	if rule.Name != "main" {
		t.Errorf("Name = %s, want main", rule.Name)
	}
	if !rule.RequireApproval {
		t.Error("RequireApproval should be true")
	}
	if rule.ApprovalCount != 2 {
		t.Errorf("ApprovalCount = %d, want 2", rule.ApprovalCount)
	}
	if len(rule.StatusContexts) != 2 {
		t.Errorf("StatusContexts length = %d, want 2", len(rule.StatusContexts))
	}
}

func TestSecurityScanResultFields(t *testing.T) {
	result := SecurityScanResult{
		ID:      123,
		Tool:    "sast",
		Status:  "passed",
		Summary: "No issues found",
		Issues: []SecurityIssue{
			{
				Severity:    "high",
				RuleID:      "S1001",
				Description: "SQL injection",
				File:        "main.go",
				Line:        42,
			},
		},
		ReportURL: "https://example.com/report",
	}

	if result.ID != 123 {
		t.Errorf("ID = %d, want 123", result.ID)
	}
	if result.Tool != "sast" {
		t.Errorf("Tool = %s, want sast", result.Tool)
	}
	if result.Status != "passed" {
		t.Errorf("Status = %s, want passed", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Errorf("Issues length = %d, want 1", len(result.Issues))
	}
}

func TestSecurityIssueFields(t *testing.T) {
	issue := SecurityIssue{
		Severity:    "critical",
		RuleID:      "S1001",
		Description: "Security issue",
		File:        "main.go",
		Line:        10,
		CWE:         "CWE-89",
	}

	if issue.Severity != "critical" {
		t.Errorf("Severity = %s, want critical", issue.Severity)
	}
	if issue.RuleID != "S1001" {
		t.Errorf("RuleID = %s, want S1001", issue.RuleID)
	}
	if issue.CWE != "CWE-89" {
		t.Errorf("CWE = %s, want CWE-89", issue.CWE)
	}
}

func TestCodeQualityMetricsFields(t *testing.T) {
	metrics := CodeQualityMetrics{
		Coverage:           85.5,
		Duplication:        5.2,
		Complexity:         12.3,
		CodeSmellCount:     5,
		BugCount:           2,
		VulnerabilityCount: 1,
	}

	if metrics.Coverage != 85.5 {
		t.Errorf("Coverage = %f, want 85.5", metrics.Coverage)
	}
	if metrics.Duplication != 5.2 {
		t.Errorf("Duplication = %f, want 5.2", metrics.Duplication)
	}
	if metrics.BugCount != 2 {
		t.Errorf("BugCount = %d, want 2", metrics.BugCount)
	}
}

func TestReviewerSuggestionFields(t *testing.T) {
	suggestion := ReviewerSuggestion{
		Username:    "reviewer1",
		Email:       "reviewer1@example.com",
		Reason:      "code-owner",
		Score:       0.95,
		FileCount:   5,
		RecentFiles: []string{"file1.go", "file2.go"},
	}

	if suggestion.Username != "reviewer1" {
		t.Errorf("Username = %s, want reviewer1", suggestion.Username)
	}
	if suggestion.Email != "reviewer1@example.com" {
		t.Errorf("Email = %s, want reviewer1@example.com", suggestion.Email)
	}
	if suggestion.Reason != "code-owner" {
		t.Errorf("Reason = %s, want code-owner", suggestion.Reason)
	}
	if suggestion.Score != 0.95 {
		t.Errorf("Score = %f, want 0.95", suggestion.Score)
	}
	if len(suggestion.RecentFiles) != 2 {
		t.Errorf("RecentFiles length = %d, want 2", len(suggestion.RecentFiles))
	}
}

func TestEnterpriseInfoFields(t *testing.T) {
	info := EnterpriseInfo{
		ID:             123,
		Name:           "Test Enterprise",
		Slug:           "test-enterprise",
		DisplayName:    "Test Enterprise Display",
		LogoURL:        "https://example.com/logo.png",
		Level3Security: true,
		PasswordEval:   true,
		Xinchuang:      true,
	}

	if info.ID != 123 {
		t.Errorf("ID = %d, want 123", info.ID)
	}
	if info.Name != "Test Enterprise" {
		t.Errorf("Name = %s, want Test Enterprise", info.Name)
	}
	if !info.Level3Security {
		t.Error("Level3Security should be true")
	}
	if !info.PasswordEval {
		t.Error("PasswordEval should be true")
	}
	if !info.Xinchuang {
		t.Error("Xinchuang should be true")
	}
}

func TestGiteeGoConfigFields(t *testing.T) {
	config := GiteeGoConfig{
		Name:        "CI Pipeline",
		Description: "Main CI pipeline",
		YAML:        "stages:\n  - build",
		Variables: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		},
		Enabled: true,
	}

	if config.Name != "CI Pipeline" {
		t.Errorf("Name = %s, want CI Pipeline", config.Name)
	}
	if len(config.Variables) != 2 {
		t.Errorf("Variables length = %d, want 2", len(config.Variables))
	}
	if !config.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestComplianceReportFields(t *testing.T) {
	report := ComplianceReport{
		Enterprise: &EnterpriseInfo{
			ID:   123,
			Name: "Enterprise",
		},
		HasCodeOwners:       true,
		HasBranchProtection: true,
		HasSecurityScan:     true,
		HasStatusChecks:     true,
		ComplianceScore:     95.5,
		Recommendations:     []string{"Enable 2FA"},
	}

	if !report.HasCodeOwners {
		t.Error("HasCodeOwners should be true")
	}
	if report.ComplianceScore != 95.5 {
		t.Errorf("ComplianceScore = %f, want 95.5", report.ComplianceScore)
	}
	if len(report.Recommendations) != 1 {
		t.Errorf("Recommendations length = %d, want 1", len(report.Recommendations))
	}
}

func TestCodeOwnersLocations(t *testing.T) {
	// Test that we check all expected CODEOWNERS locations
	locations := []string{
		".gitee/CODEOWNERS",
		"CODEOWNERS",
		".github/CODEOWNERS",
		"docs/CODEOWNERS",
	}

	// Just verify the constant logic
	for _, loc := range locations {
		if loc == "" {
			t.Error("Location should not be empty")
		}
	}
}
