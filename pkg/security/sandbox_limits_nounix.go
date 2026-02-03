// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Use of this source code is governed by the Apache-2.0 license
// that can be found in the LICENSE file.

//go:build !unix

package security

// setResourceLimits is a no-op on non-Unix systems (Windows).
func (s *Sandbox) setResourceLimits() {
	// Resource limits via setrlimit are not available on Windows.
	// This function is intentionally left empty.
}
