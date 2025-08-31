package init

import (
	"fmt"
)

// ErrorType represents the type of initialization error
type ErrorType int

const (
	// ErrorTypeConfig indicates a configuration file error
	ErrorTypeConfig ErrorType = iota
	// ErrorTypeGitHub indicates a GitHub API error
	ErrorTypeGitHub
	// ErrorTypeFileSystem indicates a file system error
	ErrorTypeFileSystem
	// ErrorTypeValidation indicates a validation error
	ErrorTypeValidation
)

// InitError represents an initialization error with context
type InitError struct {
	Type    ErrorType
	Message string
	Cause   error
}

// Error implements the error interface
func (e *InitError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *InitError) Unwrap() error {
	return e.Cause
}

// NewConfigError creates a new configuration error
func NewConfigError(message string, cause error) *InitError {
	return &InitError{
		Type:    ErrorTypeConfig,
		Message: message,
		Cause:   cause,
	}
}

// NewGitHubError creates a new GitHub API error
func NewGitHubError(message string, cause error) *InitError {
	return &InitError{
		Type:    ErrorTypeGitHub,
		Message: message,
		Cause:   cause,
	}
}

// NewFileSystemError creates a new file system error
func NewFileSystemError(message string, cause error) *InitError {
	return &InitError{
		Type:    ErrorTypeFileSystem,
		Message: message,
		Cause:   cause,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *InitError {
	return &InitError{
		Type:    ErrorTypeValidation,
		Message: message,
		Cause:   nil,
	}
}

// HandleInitError provides appropriate error handling based on error type
func HandleInitError(err error) {
	if err == nil {
		return
	}

	switch e := err.(type) {
	case *InitError:
		switch e.Type {
		case ErrorTypeConfig:
			fmt.Printf("Configuration error: %v\n", e)
			fmt.Println("Please check your .gh-pm.yml file format and try again.")
		case ErrorTypeGitHub:
			fmt.Printf("GitHub API error: %v\n", e)
			fmt.Println("Please check your network connection and GitHub authentication:")
			fmt.Println("  Run: gh auth status")
		case ErrorTypeFileSystem:
			fmt.Printf("File system error: %v\n", e)
			fmt.Println("Please check file permissions and disk space.")
		case ErrorTypeValidation:
			fmt.Printf("Validation error: %v\n", e)
			fmt.Println("Please check your input values and try again.")
		default:
			fmt.Printf("Error: %v\n", e)
		}
	default:
		fmt.Printf("Unexpected error: %v\n", err)
		fmt.Println("Please report this issue at: https://github.com/yahsan2/gh-pm/issues")
	}
}
