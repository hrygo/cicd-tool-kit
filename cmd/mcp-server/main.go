// Package main is the entry point for MCP server
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"sync"

	"github.com/cicd-ai-toolkit/cicd-runner/pkg/config"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/mcp"
	"github.com/cicd-ai-toolkit/cicd-runner/pkg/platform"
)

func main() {
	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			slog.Error("PANIC", "error", r, "stack", string(debug.Stack()))
			os.Exit(2)
		}
	}()

	if err := run(); err != nil {
		slog.Error("Error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create platform client
	platformClient, err := createPlatform(cfg)
	if err != nil {
		return fmt.Errorf("failed to create platform: %w", err)
	}

	// Create MCP server
	server := mcp.NewServer(platformClient, logger)

	// Check transport mode (stdio is default for Claude Code)
	transport := os.Getenv("MCP_TRANSPORT")
	if transport == "http" {
		// HTTP mode - useful for testing
		return runHTTPServer(ctx, server)
	}

	// Stdio mode - for direct Claude Code integration
	return server.ServeStdio(ctx, os.Stdin, os.Stdout)
}

func runHTTPServer(ctx context.Context, server *mcp.Server) error {
	// HTTP server for testing
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", server.ServeHTTP)

	addr := os.Getenv("MCP_SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	slog.Info("MCP server listening", "address", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	wg.Wait()
	return nil
}

func loadConfig() (*config.Config, error) {
	if cfgFile := os.Getenv("CONFIG_FILE"); cfgFile != "" {
		return config.Load(cfgFile)
	}
	return config.LoadFromEnv()
}

func createPlatform(cfg *config.Config) (platform.Platform, error) {
	platformName := platform.DetectPlatform()

	switch platformName {
	case "github":
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
			if err := client.SetBaseURL(cfg.Platform.GitHub.APIURL); err != nil {
				return nil, fmt.Errorf("failed to set GitHub API URL: %w", err)
			}
		}
		return client, nil

	case "gitlab":
		token := os.Getenv("GITLAB_TOKEN")
		if token == "" {
			token = cfg.Platform.GitLab.Token
		}
		repo, err := platform.ParseRepoFromGitLabEnv()
		if err != nil {
			return nil, fmt.Errorf("failed to determine repository: %w", err)
		}
		client := platform.NewGitLabClient(token, repo)
		if cfg.Platform.GitLab.APIURL != "" {
			if err := client.SetBaseURL(cfg.Platform.GitLab.APIURL); err != nil {
				return nil, fmt.Errorf("failed to set GitLab API URL: %w", err)
			}
		}
		return client, nil

	case "gitee":
		token := os.Getenv("GITEE_TOKEN")
		if token == "" {
			token = cfg.Platform.Gitee.Token
		}
		repo, err := platform.ParseRepoFromGiteeEnv()
		if err != nil {
			return nil, fmt.Errorf("failed to determine repository: %w", err)
		}
		client := platform.NewGiteeClient(token, repo)
		if cfg.Platform.Gitee.APIURL != "" {
			if err := client.SetBaseURL(cfg.Platform.Gitee.APIURL); err != nil {
				return nil, fmt.Errorf("failed to set Gitee API URL: %w", err)
			}
		}
		return client, nil

	default:
		return nil, fmt.Errorf("unsupported platform: %s", platformName)
	}
}
