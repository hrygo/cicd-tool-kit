// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package runner provides the core execution engine for CICD AI Toolkit.
//
// Runner is responsible for:
//   - Process management: Starting, monitoring, and terminating Claude CLI subprocesses
//   - IO redirection: Capturing stdin/stdout/stderr for context injection and result capture
//   - Lifecycle management: Init, Execute, Cleanup phases
//   - Signal handling: Graceful shutdown on SIGINT/SIGTERM
//   - Watchdog: Retry mechanism with exponential backoff
//   - Fallback: Graceful degradation when Claude API is unavailable
//
// Example usage:
//
//	runner := runner.New()
//	if err := runner.Bootstrap(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer runner.Shutdown(ctx)
//
//	result, err := runner.Run(ctx, &runner.RunRequest{
//	    SkillName: "code-reviewer",
//	    Timeout:   5 * time.Minute,
//	})
package runner
