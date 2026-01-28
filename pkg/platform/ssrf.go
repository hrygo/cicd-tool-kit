// Package platform provides security utilities for SSRF protection
package platform

import (
	"fmt"
	"net/url"
	"regexp"
)

// validURLPattern matches safe URL schemes (http/https only)
var validURLPattern = regexp.MustCompile(`^https?://`)

// privateIPPatterns matches private/internal network IP addresses to prevent SSRF
// Focused on blocking cloud metadata endpoints and internal network ranges
// Note: localhost is explicitly allowed for local development/testing
var privateIPPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(^|\.)10\.`),                          // 10.0.0.0/8 (private network)
	regexp.MustCompile(`(^|\.)172\.(1[6-9]|2[0-9]|3[0-1])\.`), // 172.16.0.0/12 (private network)
	regexp.MustCompile(`(^|\.)192\.168\.`),                    // 192.168.0.0/16 (private network)
	// Block cloud metadata endpoints specifically
	regexp.MustCompile(`(^|\.)169\.254\.169\.254$`), // AWS/GCP/Azure metadata endpoint
	regexp.MustCompile(`(^|\.)fc00:`),                // fc00::/7 (IPv6 private)
	regexp.MustCompile(`^fe80:`),                     // fe80::/10 (IPv6 link-local) - prefix match to prevent bypass
	regexp.MustCompile(`^::1`),                       // IPv6 loopback - prefix match to catch ::1, ::1%1, etc.
}

// validateBaseURL validates the baseURL to prevent SSRF attacks
// Ensures only http/https schemes are used and blocks private/internal networks
func validateBaseURL(baseURL string) error {
	// Check URL scheme - only allow http and https
	if !validURLPattern.MatchString(baseURL) {
		return fmt.Errorf("invalid URL scheme: only http and https are allowed")
	}

	// Parse URL to extract hostname
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL has no hostname")
	}

	// Check against private/internal network patterns
	for _, pattern := range privateIPPatterns {
		if pattern.MatchString(hostname) {
			return fmt.Errorf("SSRF protection: cannot connect to private/internal network: %s", hostname)
		}
	}

	return nil
}
