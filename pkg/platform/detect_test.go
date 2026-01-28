// Package platform provides platform detection functionality
package platform

import (
	"os"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	// Save original env vars
	origEnv := make(map[string]string)
	envVars := []string{
		"GITHUB_ACTIONS", "GITLAB_CI", "GITEE_CI", "GITEE_SERVER_URL",
		"JENKINS_HOME", "JENKINS_URL", "TF_BUILD", "BITBUCKET_BUILD_NUMBER",
		"CIRCLECI", "TRAVIS", "DRONE",
	}
	for _, v := range envVars {
		origEnv[v] = os.Getenv(v)
	}

	// Clean env for testing
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	// Default should be local
	if got := DetectPlatform(); got != "local" {
		t.Errorf("expected local, got %s", got)
	}

	// Test GitHub
	os.Setenv("GITHUB_ACTIONS", "true")
	if got := DetectPlatform(); got != "github" {
		t.Errorf("expected github, got %s", got)
	}
	os.Unsetenv("GITHUB_ACTIONS")

	// Test GitLab
	os.Setenv("GITLAB_CI", "true")
	if got := DetectPlatform(); got != "gitlab" {
		t.Errorf("expected gitlab, got %s", got)
	}
	os.Unsetenv("GITLAB_CI")

	// Test Gitee
	os.Setenv("GITEE_CI", "true")
	if got := DetectPlatform(); got != "gitee" {
		t.Errorf("expected gitee, got %s", got)
	}
	os.Unsetenv("GITEE_CI")

	os.Setenv("GITEE_SERVER_URL", "https://gitee.com")
	if got := DetectPlatform(); got != "gitee" {
		t.Errorf("expected gitee, got %s", got)
	}
	os.Unsetenv("GITEE_SERVER_URL")

	// Test Jenkins
	os.Setenv("JENKINS_HOME", "/var/jenkins")
	if got := DetectPlatform(); got != "jenkins" {
		t.Errorf("expected jenkins, got %s", got)
	}
	os.Unsetenv("JENKINS_HOME")

	// Restore env
	for _, v := range envVars {
		if val, ok := origEnv[v]; ok {
			os.Setenv(v, val)
		}
	}
}

func TestDetectPlatformInfo(t *testing.T) {
	// Save original env vars
	origEnv := make(map[string]string)
	envVars := []string{"GITHUB_ACTIONS", "GITLAB_CI"}
	for _, v := range envVars {
		origEnv[v] = os.Getenv(v)
		os.Unsetenv(v)
	}

	// Test no CI
	info := DetectPlatformInfo()
	if info.Name != "local" {
		t.Errorf("expected local, got %s", info.Name)
	}
	if info.IsCI {
		t.Error("expected IsCI=false for local")
	}

	// Test GitHub
	os.Setenv("GITHUB_ACTIONS", "true")
	info = DetectPlatformInfo()
	if info.Name != "github" {
		t.Errorf("expected github, got %s", info.Name)
	}
	if !info.IsCI {
		t.Error("expected IsCI=true for github")
	}
	if info.VarName != "GITHUB_ACTIONS" {
		t.Errorf("expected GITHUB_ACTIONS, got %s", info.VarName)
	}
	if info.VarValue != "true" {
		t.Errorf("expected true, got %s", info.VarValue)
	}

	os.Unsetenv("GITHUB_ACTIONS")

	// Restore env
	for _, v := range envVars {
		if val, ok := origEnv[v]; ok {
			os.Setenv(v, val)
		}
	}
}

func TestIsRunningInCI(t *testing.T) {
	// Save original env
	origVal := os.Getenv("GITHUB_ACTIONS")
	defer func() {
		if origVal != "" {
			os.Setenv("GITHUB_ACTIONS", origVal)
		} else {
			os.Unsetenv("GITHUB_ACTIONS")
		}
	}()

	// Not in CI
	os.Unsetenv("GITHUB_ACTIONS")
	if IsRunningInCI() {
		t.Error("expected false when not in CI")
	}

	// In CI
	os.Setenv("GITHUB_ACTIONS", "true")
	if !IsRunningInCI() {
		t.Error("expected true when in CI")
	}
}

func TestGetPlatformFromConfig(t *testing.T) {
	// Save original env
	origVal := os.Getenv("GITHUB_ACTIONS")
	defer func() {
		if origVal != "" {
			os.Setenv("GITHUB_ACTIONS", origVal)
		} else {
			os.Unsetenv("GITHUB_ACTIONS")
		}
	}()

	os.Unsetenv("GITHUB_ACTIONS")

	// Auto mode should detect
	if got := GetPlatformFromConfig("auto"); got != "local" {
		t.Errorf("auto mode: expected local, got %s", got)
	}

	// Explicit config should be returned
	if got := GetPlatformFromConfig("github"); got != "github" {
		t.Errorf("explicit: expected github, got %s", got)
	}
}

func TestValidatePlatform(t *testing.T) {
	// Valid platforms
	validPlatforms := []string{"github", "gitlab", "gitee", "jenkins", "azure", "bitbucket", "circleci", "travis", "drone", "local"}
	for _, p := range validPlatforms {
		if err := ValidatePlatform(p); err != nil {
			t.Errorf("platform %s should be valid: %v", p, err)
		}
	}

	// Invalid platform
	if err := ValidatePlatform("invalid"); err == nil {
		t.Error("invalid platform should return error")
	}
}

func TestGetSupportedPlatforms(t *testing.T) {
	platforms := GetSupportedPlatforms()

	if len(platforms) < 9 {
		t.Errorf("expected at least 9 platforms, got %d", len(platforms))
	}

	// Check for essential platforms
	essential := []string{"github", "gitlab", "gitee", "jenkins", "local"}
	for _, p := range essential {
		found := false
		for _, supported := range platforms {
			if supported == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("essential platform %s not found in %v", p, platforms)
		}
	}
}
