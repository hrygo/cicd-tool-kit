// Package main provides the cicd-runner CLI application.
package main

import (
	"fmt"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display detailed version information including build date, git commit, and Go version.`,
	Run: func(cmd *cobra.Command, args []string) {
		info := version.Info()
		fmt.Printf("cicd-runner version: %s\n", info["version"])
		fmt.Printf("  build date: %s\n", info["buildDate"])
		fmt.Printf("  git commit: %s\n", info["gitCommit"])
		fmt.Printf("  go version: %s\n", info["goVersion"])
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
