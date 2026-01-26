// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cicd-ai-toolkit/pkg/runner"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run AI analysis on CI/CD events",
	Long: `Execute AI-powered analysis on pull requests, pushes, and other CI/CD events.

This command will:
1. Detect the current platform (GitHub, GitLab, Gitee, etc.)
2. Gather context (git diff, commit messages, files changed)
3. Execute configured skills
4. Post results back to the platform`,
	RunE: runRun,
}

var (
	runSkills    string
	runConfig    string
	runPlatform  string
	runEventPath string
	runDryRun    bool
	runVerbose   bool
	runTimeout   time.Duration
)

func init() {
	runCmd.Flags().StringVarP(&runSkills, "skills", "s", "code-reviewer",
		"Comma-separated list of skills to run")
	runCmd.Flags().StringVarP(&runConfig, "config", "c", ".cicd-ai-toolkit.yaml",
		"Path to configuration file")
	runCmd.Flags().StringVarP(&runPlatform, "platform", "p", "",
		"Force platform (github, gitlab, gitee, jenkins)")
	runCmd.Flags().StringVar(&runEventPath, "event", "",
		"Path to event payload file")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false,
		"Run without posting comments")
	runCmd.Flags().BoolVarP(&runVerbose, "verbose", "v", false,
		"Verbose output")
	runCmd.Flags().DurationVar(&runTimeout, "timeout", 5*time.Minute,
		"Execution timeout")
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse skills list
	skills := strings.Split(runSkills, ",")
	for i := range skills {
		skills[i] = strings.TrimSpace(skills[i])
	}

	if runVerbose {
		fmt.Println("CICD AI Toolkit Runner")
		fmt.Println("=======================")
		fmt.Printf("Skills: %s\n", strings.Join(skills, ", "))
		fmt.Printf("Config: %s\n", runConfig)
		fmt.Printf("Platform: %s\n", runPlatform)
		fmt.Printf("Dry Run: %v\n", runDryRun)
		fmt.Printf("Timeout: %v\n", runTimeout)
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create runner with options
	r := runner.NewWithOptions(&runner.Options{
		ConfigPath:      runConfig,
		WorkDir:         workDir,
		SkillDirs:       []string{".skills", "skills"},
		PreWarmClaude:   false,
		GracefulTimeout: 5 * time.Second,
		Verbose:         runVerbose,
		DryRun:          runDryRun,
	})

	// Bootstrap the runner
	if err := r.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	defer r.Shutdown(ctx)

	if runVerbose {
		metrics := r.BootstrapMetrics()
		fmt.Printf("Bootstrap time: %v\n", metrics.TotalTime)
	}

	// Run each skill
	exitCode := 0
	for _, skillName := range skills {
		if runVerbose {
			fmt.Printf("\nRunning skill: %s\n", skillName)
		}

		result, err := r.Run(ctx, &runner.RunRequest{
			SkillName: skillName,
			Inputs:    buildInputs(ctx),
			Timeout:   runTimeout,
			DryRun:    runDryRun,
		})

		if err != nil && runVerbose {
			fmt.Printf("Error: %v\n", err)
		}

		if result != nil {
			if runVerbose {
				fmt.Printf("Exit code: %d\n", result.ExitCode)
				fmt.Printf("Duration: %v\n", result.Duration)
				if result.Retries > 0 {
					fmt.Printf("Retries: %d\n", result.Retries)
				}
			}

			if result.Output != "" {
				fmt.Println(result.Output)
			}

			if result.ExitCode != 0 && exitCode == 0 {
				exitCode = result.ExitCode
			}
		}
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}

// buildInputs builds input values for the skill from context.
func buildInputs(ctx context.Context) map[string]any {
	// TODO: Build inputs from platform event, git diff, etc.
	// For now, return empty map
	_ = ctx
	return map[string]any{}
}
