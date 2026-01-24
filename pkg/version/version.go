// Package version provides version information for the cicd-runner.
// These variables are set via ldflags during the build process.
package version

// Version is the current version of the binary.
// Set via -ldflags "-X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.Version=..."
var Version = "dev"

// BuildDate is the date when the binary was built.
// Set via -ldflags "-X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.BuildDate=..."
var BuildDate = "unknown"

// GitCommit is the git commit hash used to build the binary.
// Set via -ldflags "-X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.GitCommit=..."
var GitCommit = "unknown"

// GoVersion is the Go version used to build the binary.
// Set via -ldflags "-X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.GoVersion=..."
var GoVersion = "unknown"

// String returns a formatted version string.
func String() string {
	return Version
}

// FullString returns a detailed version string including build info.
func FullString() string {
	if Version == "dev" {
		return "cicd-runner development version"
	}
	return "cicd-runner " + Version
}

// Info returns all version information as a map.
func Info() map[string]string {
	return map[string]string{
		"version":   Version,
		"buildDate": BuildDate,
		"gitCommit": GitCommit,
		"goVersion": GoVersion,
	}
}
