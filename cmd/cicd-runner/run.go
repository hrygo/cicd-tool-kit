// Package main provides the cicd-runner CLI application.
package main

import (
	"fmt"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/version"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
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

// runFlags holds the flags for the run command
type runFlags struct {
	skills    string
	config    string
	verbose   bool
	dryRun    bool
}

var runOpts runFlags

func init() {
	rootCmd.AddCommand(runCmd)

	// Local flags for the run command
	runCmd.Flags().StringVarP(&runOpts.skills, "skills", "s", "code-reviewer", "Comma-separated list of skills to run")
	runCmd.Flags().StringVarP(&runOpts.config, "config", "c", "", "Path to configuration file")
	runCmd.Flags().BoolVarP(&runOpts.verbose, "verbose", "v", false, "Verbose output")
	runCmd.Flags().BoolVar(&runOpts.dryRun, "dry-run", false, "Show what would be done without doing it")
}
