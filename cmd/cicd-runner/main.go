// Package main is the entry point for the cicd-runner CLI.
package main

import (
	"fmt"
	"os"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/version"
	"github.com/spf13/cobra"
)

func main() {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "cicd-runner",
		Short: "CICD AI Toolkit Runner",
		Long: `CICD AI Toolkit Runner - An AI-powered CI/CD assistant.

The runner integrates with your CI/CD pipeline to provide intelligent
code analysis, automated testing, and more.`,
		Version: version.FullString(),
	}

	// Version command with detailed output
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.Info()
			fmt.Printf("cicd-runner version: %s\n", info["version"])
			fmt.Printf("  build date: %s\n", info["buildDate"])
			fmt.Printf("  git commit: %s\n", info["gitCommit"])
			fmt.Printf("  go version: %s\n", info["goVersion"])
		},
	}

	// Run command (placeholder for now)
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the CICD AI Toolkit",
		Long: `Run the CICD AI Toolkit to analyze your codebase.

This command will analyze the current git diff and provide
AI-powered insights and suggestions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("CICD AI Toolkit Runner v" + version.String())
			fmt.Println("This is a placeholder implementation.")
			fmt.Println("Full implementation coming in Phase 2 (CORE-01).")
			return nil
		},
	}

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
