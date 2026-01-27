// Package errors provides typed errors for cicd-ai-toolkit
package errors

import (
	"errors"
	"fmt"
)

// ErrorType represents the category of error
type ErrorType int

const (
	// ErrConfig indicates a configuration error
	ErrConfig ErrorType = iota
	// ErrPlatform indicates a platform API error
	ErrPlatform
	// ErrClaude indicates a Claude Code execution error
	ErrClaude
	// ErrSkill indicates a skill loading/execution error
	ErrSkill
	// ErrValidation indicates an input validation error
	ErrValidation
	// ErrTimeout indicates a timeout occurred
	ErrTimeout
	// ErrBudget indicates budget limit exceeded
	ErrBudget
)

// CICDError is the base error type for all cicd-ai-toolkit errors
type CICDError struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error returns the error message
func (e *CICDError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", errorTypeString(e.Type), e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", errorTypeString(e.Type), e.Message)
}

// Unwrap returns the underlying cause
func (e *CICDError) Unwrap() error {
	return e.Cause
}

// New creates a new CICDError
func New(errType ErrorType, message string, cause error) *CICDError {
	return &CICDError{
		Type:    errType,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *CICDError) WithContext(key string, value interface{}) *CICDError {
	e.Context[key] = value
	return e
}

// IsType checks if an error is of a specific type
func IsType(err error, errType ErrorType) bool {
	var cicdErr *CICDError
	if err == nil {
		return false
	}
	if errors.As(err, &cicdErr) {
		return cicdErr.Type == errType
	}
	return false
}

// IsRetryable returns true if the error is transient and retryable
func IsRetryable(err error) bool {
	var cicdErr *CICDError
	if !errors.As(err, &cicdErr) {
		return false
	}

	switch cicdErr.Type {
	case ErrPlatform, ErrTimeout:
		return true
	case ErrClaude:
		// Retry only for rate limits and timeouts
		return cicdErr.Message == "rate_limit_exceeded" || cicdErr.Message == "timeout"
	default:
		return false
	}
}

// ShouldBlockCI returns true if the error should block the CI pipeline
func ShouldBlockCI(err error) bool {
	var cicdErr *CICDError
	if !errors.As(err, &cicdErr) {
		return false
	}

	// Most errors should NOT block CI (per PRD requirement)
	switch cicdErr.Type {
	case ErrBudget, ErrClaude:
		// Don't block CI for Claude issues
		return false
	case ErrConfig, ErrValidation:
		// Configuration errors should block (user needs to fix)
		return true
	default:
		return false
	}
}

func errorTypeString(et ErrorType) string {
	switch et {
	case ErrConfig:
		return "CONFIG"
	case ErrPlatform:
		return "PLATFORM"
	case ErrClaude:
		return "CLAUDE"
	case ErrSkill:
		return "SKILL"
	case ErrValidation:
		return "VALIDATION"
	case ErrTimeout:
		return "TIMEOUT"
	case ErrBudget:
		return "BUDGET"
	default:
		return "UNKNOWN"
	}
}

// Convenience functions for common errors

// ConfigError creates a configuration error
func ConfigError(message string, cause error) *CICDError {
	return New(ErrConfig, message, cause)
}

// PlatformError creates a platform error
func PlatformError(message string, cause error) *CICDError {
	return New(ErrPlatform, message, cause)
}

// ClaudeError creates a Claude execution error
func ClaudeError(message string, cause error) *CICDError {
	return New(ErrClaude, message, cause)
}

// SkillError creates a skill error
func SkillError(message string, cause error) *CICDError {
	return New(ErrSkill, message, cause)
}

// ValidationError creates a validation error
func ValidationError(message string, cause error) *CICDError {
	return New(ErrValidation, message, cause)
}

// TimeoutError creates a timeout error
func TimeoutError(message string, cause error) *CICDError {
	return New(ErrTimeout, message, cause)
}

// BudgetError creates a budget exceeded error
func BudgetError(message string, cause error) *CICDError {
	return New(ErrBudget, message, cause)
}
