// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cicd-runner",
	Short: "CICD AI Toolkit - AI-powered code review and analysis runner",
	Long: `CICD AI Toolkit Runner

An AI-powered tool for CI/CD pipelines that provides code review,
test generation, change analysis, and more using Claude AI.`,
	Version: getVersion(),
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(skillCmd)
	rootCmd.AddCommand(versionCmd)
}

func getVersion() string {
	return "0.1.0-dev"
}

// Execute runs the root command
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
	return nil
}
