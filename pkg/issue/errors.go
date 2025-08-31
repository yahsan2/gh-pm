package issue

import (
	"fmt"
	"strings"
)

// ErrorType represents the type of error that occurred
type ErrorType int

const (
	// ErrorTypeValidation indicates a validation error
	ErrorTypeValidation ErrorType = iota
	// ErrorTypeConfiguration indicates a configuration error
	ErrorTypeConfiguration
	// ErrorTypePermission indicates a permission/authorization error
	ErrorTypePermission
	// ErrorTypeNetwork indicates a network connectivity error
	ErrorTypeNetwork
	// ErrorTypeRateLimit indicates GitHub API rate limit error
	ErrorTypeRateLimit
	// ErrorTypeNotFound indicates a resource was not found
	ErrorTypeNotFound
	// ErrorTypeAPI indicates a general API error
	ErrorTypeAPI
)

// IssueError represents a structured error with type and suggestion
type IssueError struct {
	Type       ErrorType
	Message    string
	Cause      error
	Suggestion string
}

// Error implements the error interface
func (e *IssueError) Error() string {
	var parts []string

	// Add main message
	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	// Add cause if present
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("caused by: %v", e.Cause))
	}

	// Add suggestion if present
	if e.Suggestion != "" {
		parts = append(parts, fmt.Sprintf("\n💡 %s", e.Suggestion))
	}

	return strings.Join(parts, ": ")
}

// Unwrap returns the underlying error
func (e *IssueError) Unwrap() error {
	return e.Cause
}

// Is checks if the error is of a specific type
func (e *IssueError) Is(target error) bool {
	t, ok := target.(*IssueError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// NewValidationError creates a new validation error
func NewValidationError(message string, cause error) *IssueError {
	return &IssueError{
		Type:       ErrorTypeValidation,
		Message:    message,
		Cause:      cause,
		Suggestion: "Check your input parameters and try again",
	}
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(message string, cause error) *IssueError {
	return &IssueError{
		Type:       ErrorTypeConfiguration,
		Message:    message,
		Cause:      cause,
		Suggestion: "Run 'gh pm init' to create or update your configuration",
	}
}

// NewPermissionError creates a new permission error
func NewPermissionError(message string, cause error) *IssueError {
	return &IssueError{
		Type:       ErrorTypePermission,
		Message:    message,
		Cause:      cause,
		Suggestion: "Check that you have write access to the repository and the required token scopes (repo, project, write:org)",
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, cause error) *IssueError {
	return &IssueError{
		Type:       ErrorTypeNetwork,
		Message:    message,
		Cause:      cause,
		Suggestion: "Check your internet connection and try again",
	}
}

// NewRateLimitError creates a new rate limit error with reset time
func NewRateLimitError(resetTime string) *IssueError {
	return &IssueError{
		Type:       ErrorTypeRateLimit,
		Message:    "GitHub API rate limit exceeded",
		Suggestion: fmt.Sprintf("Rate limit will reset at %s. You can wait or use a different token", resetTime),
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource string) *IssueError {
	return &IssueError{
		Type:       ErrorTypeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		Suggestion: fmt.Sprintf("Check that the %s exists and you have access to it", resource),
	}
}

// NewAPIError creates a new general API error
func NewAPIError(message string, cause error) *IssueError {
	return &IssueError{
		Type:       ErrorTypeAPI,
		Message:    message,
		Cause:      cause,
		Suggestion: "Check GitHub status at https://www.githubstatus.com/ and try again",
	}
}

// WrapError wraps an existing error with additional context
func WrapError(err error, message string) *IssueError {
	if issueErr, ok := err.(*IssueError); ok {
		// If it's already an IssueError, preserve the type and add context
		return &IssueError{
			Type:       issueErr.Type,
			Message:    message,
			Cause:      err,
			Suggestion: issueErr.Suggestion,
		}
	}

	// Otherwise, create a generic API error
	return NewAPIError(message, err)
}
