// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package version provides version information.
package version

const (
	// Version is the application version.
	Version = "0.1.0"

	// GitCommit is the git commit hash.
	GitCommit = "unknown"

	// BuildDate is the build date.
	BuildDate = "unknown"
)

// Info returns version information.
func Info() string {
	return Version
}

// FullInfo returns full version information.
func FullInfo() map[string]string {
	return map[string]string{
		"version":   Version,
		"commit":    GitCommit,
		"buildDate": BuildDate,
	}
}
