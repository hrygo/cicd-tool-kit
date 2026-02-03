// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Use of this source code is governed by the Apache-2.0 license
// that can be found in the LICENSE file.

//go:build unix

package security

import (
	"runtime"
	"syscall"
)

// setResourceLimits sets resource limits for the process (Unix systems).
func (s *Sandbox) setResourceLimits() {
	rl := s.resourceLimits

	// Memory limit (Linux only, via setrlimit RLIMIT_AS)
	//nolint:errcheck // Intentionally ignore - best-effort resource limits
	if rl.MaxMemory > 0 && runtime.GOOS == "linux" {
		_ = syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
			Cur: uint64(rl.MaxMemory),
			Max: uint64(rl.MaxMemory),
		})
	}

	// CPU time limit (Linux only)
	//nolint:errcheck // Intentionally ignore - best-effort resource limits
	if rl.MaxWallTime > 0 && runtime.GOOS == "linux" {
		_ = syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{
			Cur: uint64(rl.MaxWallTime.Seconds()),
			Max: uint64(rl.MaxWallTime.Seconds()),
		})
	}

	// Max processes (Linux only)
	//nolint:errcheck // Intentionally ignore - best-effort resource limits
	if rl.MaxProcesses > 0 && runtime.GOOS == "linux" {
		// RLIMIT_NPROC is Linux-specific
		const RLIMIT_NPROC = 6
		_ = syscall.Setrlimit(RLIMIT_NPROC, &syscall.Rlimit{
			Cur: uint64(rl.MaxProcesses),
			Max: uint64(rl.MaxProcesses),
		})
	}

	// Max open files (Linux/Darwin only)
	//nolint:errcheck // Intentionally ignore - best-effort resource limits
	if rl.MaxFiles > 0 && (runtime.GOOS == "linux" || runtime.GOOS == "darwin") {
		_ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{
			Cur: uint64(rl.MaxFiles),
			Max: uint64(rl.MaxFiles),
		})
	}
}
