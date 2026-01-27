// Package main provides the CLI entry point
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/buildcontext"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/config"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/platform"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/runner"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "cicd-runner",
	Short: "AI-powered CI/CD toolkit using Claude Code",
	Long: `cicd-runner is an AI-powered CI/CD toolkit that uses Claude Code
to perform automated code reviews, change analysis, and test generation.`,
	Version: "1.0.0",
}

// reviewCmd runs code review
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Run code review on changes",
	Long:  "Run AI-powered code review on git changes or a specific PR",
	RunE:  runReview,
}

var reviewOpts struct {
	prID       int
	diff       string
	baseSHA    string
	headSHA    string
	skills     []string
	force      bool
	postComment bool
}

// analyzeCmd runs change analysis
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze code changes",
	Long:  "Run AI-powered analysis of code changes to generate summaries and risk assessments",
	RunE:  runAnalyze,
}

var analyzeOpts struct {
	prID   int
	diff   string
	skills []string
}

// testGenCmd generates tests
var testGenCmd = &cobra.Command{
	Use:   "test-gen",
	Short: "Generate tests from code changes",
	Long:  "Generate test files based on code changes using AI",
	RunE:  runTestGen,
}

var testGenOpts struct {
	diff          string
	targetFiles   []string
	testFramework string
	createFiles   bool
	outputDir     string
}

// initCommands initializes all commands
func initCommands() {
	// Review flags
	reviewCmd.Flags().IntVarP(&reviewOpts.prID, "pr", "p", 0, "Pull request ID")
	reviewCmd.Flags().StringVarP(&reviewOpts.diff, "diff", "d", "", "Diff string to review")
	reviewCmd.Flags().StringVar(&reviewOpts.baseSHA, "base", "", "Base commit SHA")
	reviewCmd.Flags().StringVar(&reviewOpts.headSHA, "head", "", "Head commit SHA")
	reviewCmd.Flags().StringSliceVarP(&reviewOpts.skills, "skills", "s", nil, "Skills to run")
	reviewCmd.Flags().BoolVarP(&reviewOpts.force, "force", "f", false, "Skip cache")
	reviewCmd.Flags().BoolVarP(&reviewOpts.postComment, "post", "o", false, "Post comment to platform")

	// Analyze flags
	analyzeCmd.Flags().IntVarP(&analyzeOpts.prID, "pr", "p", 0, "Pull request ID")
	analyzeCmd.Flags().StringVarP(&analyzeOpts.diff, "diff", "d", "", "Diff string to analyze")
	analyzeCmd.Flags().StringSliceVarP(&analyzeOpts.skills, "skills", "s", nil, "Skills to run")

	// Test generation flags
	testGenCmd.Flags().StringVarP(&testGenOpts.diff, "diff", "d", "", "Diff string")
	testGenCmd.Flags().StringSliceVarP(&testGenOpts.targetFiles, "files", "f", nil, "Target files")
	testGenCmd.Flags().StringVarP(&testGenOpts.testFramework, "framework", "F", "", "Test framework")
	testGenCmd.Flags().BoolVarP(&testGenOpts.createFiles, "write", "w", false, "Write test files")
	testGenCmd.Flags().StringVarP(&testGenOpts.outputDir, "output", "o", "", "Output directory")

	// Add subcommands
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(testGenCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

// runReview executes the review command
func runReview(cmd *cobra.Command, args []string) error {
	ctx, cancel := signalContext()
	defer cancel()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	platformClient, err := createPlatform(cfg)
	if err != nil {
		return fmt.Errorf("failed to create platform: %w", err)
	}

	r, err := runner.NewRunner(cfg, platformClient, baseDir)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Build review options
	opts := runner.ReviewOptions{
		PRID:    reviewOpts.prID,
		Force:   reviewOpts.force,
		Skills:  reviewOpts.skills,
	}

	// Get diff if not provided
	if reviewOpts.diff == "" {
		builder := buildcontext.NewBuilder(baseDir, cfg.Global.DiffContext, cfg.Global.Exclude)
		diff, err := builder.BuildDiff(ctx, buildcontext.DiffOptions{
			TargetRef: reviewOpts.baseSHA,
			SourceRef: reviewOpts.headSHA,
		})
		if err != nil {
			return fmt.Errorf("failed to get diff: %w", err)
		}
		opts.Diff = diff
	} else {
		opts.Diff = reviewOpts.diff
	}

	// Run review
	if verbose {
		fmt.Println("Running code review...")
	}

	result, err := r.Review(ctx, opts)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Print results
	fmt.Println(result.PlatformComment)

	// Post comment if requested
	if reviewOpts.postComment {
		if err := platformClient.PostComment(ctx, platform.CommentOptions{
			PRID: opts.PRID,
			Body: result.PlatformComment,
		}); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}
		fmt.Println("\nComment posted to platform.")
	}

	return nil
}

// runAnalyze executes the analyze command
func runAnalyze(cmd *cobra.Command, args []string) error {
	ctx, cancel := signalContext()
	defer cancel()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	platformClient, err := createPlatform(cfg)
	if err != nil {
		return fmt.Errorf("failed to create platform: %w", err)
	}

	r, err := runner.NewRunner(cfg, platformClient, baseDir)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Build analyze options
	opts := runner.AnalyzeOptions{
		PRID:   analyzeOpts.prID,
		Skills: analyzeOpts.skills,
	}

	// Run analysis
	if verbose {
		fmt.Println("Running change analysis...")
	}

	result, err := r.Analyze(ctx, opts)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Print results
	fmt.Printf("Analysis complete in %v\n", result.Duration)
	fmt.Printf("Risk Score: %d/10\n", result.Risk.Score)

	return nil
}

// runTestGen executes the test generation command
func runTestGen(cmd *cobra.Command, args []string) error {
	ctx, cancel := signalContext()
	defer cancel()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	platformClient, err := createPlatform(cfg)
	if err != nil {
		return fmt.Errorf("failed to create platform: %w", err)
	}

	r, err := runner.NewRunner(cfg, platformClient, baseDir)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Build test generation options
	opts := runner.TestGenOptions{
		Diff:          testGenOpts.diff,
		TargetFiles:   testGenOpts.targetFiles,
		TestFramework: testGenOpts.testFramework,
		CreateFiles:   testGenOpts.createFiles,
	}

	// Run test generation
	if verbose {
		fmt.Println("Generating tests...")
	}

	result, err := r.GenerateTests(ctx, opts)
	if err != nil {
		return fmt.Errorf("test generation failed: %w", err)
	}

	// Print results
	fmt.Printf("Generated %d test files with %d tests\n", result.Summary.FilesCreated, result.Summary.TotalTests)
	fmt.Printf("Estimated coverage: %s\n", result.Summary.CoverageEst)

	return nil
}

// loadConfig loads the configuration
func loadConfig() (*config.Config, error) {
	if cfgFile != "" {
		return config.Load(cfgFile)
	}
	return config.LoadFromEnv()
}

// createPlatform creates the appropriate platform client
func createPlatform(cfg *config.Config) (platform.Platform, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = cfg.Platform.GitHub.Token
	}

	repo, err := platform.ParseRepoFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to determine repository: %w", err)
	}

	client := platform.NewGitHubClient(token, repo)
	if cfg.Platform.GitHub.APIURL != "" {
		client.SetBaseURL(cfg.Platform.GitHub.APIURL)
	}

	return client, nil
}

// signalContext creates a context that cancels on SIGINT/SIGTERM
func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
			// Exit goroutine when parent context is cancelled
			// Prevents goroutine leak
		}
		signal.Stop(sigChan)
		close(sigChan)
	}()

	return ctx, cancel
}

// Execute runs the CLI
func Execute() error {
	initCommands()
	return rootCmd.Execute()
}
