// Package main provides the cicd-runner CLI application.
package main

import (
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/version"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cicd-runner",
	Short: "CICD AI Toolkit Runner",
	Long: `CICD AI Toolkit Runner - An AI-powered CI/CD assistant.

The runner integrates with your CI/CD pipeline to provide intelligent
code analysis, automated testing, and more.`,
	Version: version.FullString(),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Here you can define flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cicd-runner.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
