// Package platform provides platform detection functionality
package platform

import (
	"context"
	"fmt"
	"os"
)

// DetectPlatform auto-detects the current CI/CD platform from environment variables
func DetectPlatform() string {
	// Check GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return "github"
	}

	// Check GitLab CI
	if os.Getenv("GITLAB_CI") == "true" {
		return "gitlab"
	}

	// Check Gitee
	if os.Getenv("GITEE_CI") == "true" || os.Getenv("GITEE_SERVER_URL") != "" {
		return "gitee"
	}

	// Check Jenkins
	if os.Getenv("JENKINS_HOME") != "" || os.Getenv("JENKINS_URL") != "" {
		return "jenkins"
	}

	// Check Azure Pipelines
	if os.Getenv("TF_BUILD") == "true" {
		return "azure"
	}

	// Check Bitbucket Pipelines
	if os.Getenv("BITBUCKET_BUILD_NUMBER") != "" {
		return "bitbucket"
	}

	// Check CircleCI
	if os.Getenv("CIRCLECI") == "true" {
		return "circleci"
	}

	// Check Travis CI
	if os.Getenv("TRAVIS") == "true" {
		return "travis"
	}

	// Check Drone CI
	if os.Getenv("DRONE") == "true" {
		return "drone"
	}

	// Default to local/unknown
	return "local"
}

// DetectFromEnvironment is an alias for DetectPlatform
func DetectFromEnvironment() string {
	return DetectPlatform()
}

// DetectPlatformWithContext detects platform with context for future expansion
func DetectPlatformWithContext(ctx context.Context) (string, error) {
	platform := DetectPlatform()
	return platform, nil
}

// IsRunningInCI returns true if running in any known CI environment
func IsRunningInCI() bool {
	platform := DetectPlatform()
	return platform != "local"
}

// GetPlatformFromConfig returns platform name from config, with auto-detection fallback
func GetPlatformFromConfig(configuredPlatform string) string {
	if configuredPlatform != "" && configuredPlatform != "auto" {
		return configuredPlatform
	}
	return DetectPlatform()
}

// PlatformInfo contains information about the detected platform
type PlatformInfo struct {
	Name     string
	IsCI     bool
	VarName  string // Name of the environment variable that was detected
	VarValue string // Value of the environment variable
}

// DetectPlatformInfo returns detailed platform detection information
func DetectPlatformInfo() *PlatformInfo {
	info := &PlatformInfo{}

	// Check each platform and return detailed info
	checks := []struct {
		name    string
		varName string
		detect  func() (bool, string, string)
	}{
		{"github", "GITHUB_ACTIONS", func() (bool, string, string) {
			val := os.Getenv("GITHUB_ACTIONS")
			return val == "true", "GITHUB_ACTIONS", val
		}},
		{"gitlab", "GITLAB_CI", func() (bool, string, string) {
			val := os.Getenv("GITLAB_CI")
			return val == "true", "GITLAB_CI", val
		}},
		{"gitee", "GITEE_CI", func() (bool, string, string) {
			val := os.Getenv("GITEE_CI")
			if val == "true" {
				return true, "GITEE_CI", val
			}
			val = os.Getenv("GITEE_SERVER_URL")
			return val != "", "GITEE_SERVER_URL", val
		}},
		{"jenkins", "JENKINS_HOME", func() (bool, string, string) {
			val := os.Getenv("JENKINS_HOME")
			if val != "" {
				return true, "JENKINS_HOME", val
			}
			val = os.Getenv("JENKINS_URL")
			return val != "", "JENKINS_URL", val
		}},
		{"azure", "TF_BUILD", func() (bool, string, string) {
			val := os.Getenv("TF_BUILD")
			return val == "true", "TF_BUILD", val
		}},
		{"bitbucket", "BITBUCKET_BUILD_NUMBER", func() (bool, string, string) {
			val := os.Getenv("BITBUCKET_BUILD_NUMBER")
			return val != "", "BITBUCKET_BUILD_NUMBER", val
		}},
		{"circleci", "CIRCLECI", func() (bool, string, string) {
			val := os.Getenv("CIRCLECI")
			return val == "true", "CIRCLECI", val
		}},
		{"travis", "TRAVIS", func() (bool, string, string) {
			val := os.Getenv("TRAVIS")
			return val == "true", "TRAVIS", val
		}},
		{"drone", "DRONE", func() (bool, string, string) {
			val := os.Getenv("DRONE")
			return val == "true", "DRONE", val
		}},
	}

	for _, check := range checks {
		if detected, varName, varValue := check.detect(); detected {
			info.Name = check.name
			info.IsCI = true
			info.VarName = varName
			info.VarValue = varValue
			return info
		}
	}

	// No CI detected
	info.Name = "local"
	info.IsCI = false
	return info
}

// GetSupportedPlatforms returns list of supported platform names
func GetSupportedPlatforms() []string {
	return []string{
		"github",
		"gitlab",
		"gitee",
		"jenkins",
		"azure",
		"bitbucket",
		"circleci",
		"travis",
		"drone",
		"local",
	}
}

// ValidatePlatform checks if a platform name is supported
func ValidatePlatform(platform string) error {
	supported := GetSupportedPlatforms()
	for _, name := range supported {
		if platform == name {
			return nil
		}
	}
	return fmt.Errorf("unsupported platform: %s (supported: %v)", platform, supported)
}
