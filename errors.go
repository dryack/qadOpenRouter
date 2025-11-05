package openrouter

import (
	"errors"
	"fmt"
	"net/http"
)

// Error types for structured error handling
var (
	// ErrInvalidRequest indicates invalid request parameters
	ErrInvalidRequest = errors.New("invalid request")

	// ErrAuthentication indicates API key issues
	ErrAuthentication = errors.New("authentication failed")

	// ErrRateLimit indicates rate limiting by the API
	ErrRateLimit = errors.New("rate limit exceeded")

	// ErrServerError indicates server-side errors (5xx)
	ErrServerError = errors.New("server error")

	// ErrNotFound indicates resource not found (404)
	ErrNotFound = errors.New("resource not found")

	// ErrNetworkError indicates network connectivity issues
	ErrNetworkError = errors.New("network error")

	// ErrContextCanceled indicates context was canceled
	ErrContextCanceled = errors.New("context canceled")

	// ErrContextDeadlineExceeded indicates context deadline exceeded
	ErrContextDeadlineExceeded = errors.New("context deadline exceeded")
)

// APIError represents a detailed API error with status code and message
type APIError struct {
	StatusCode int    // HTTP status code
	Message    string // Error message from API or description
	Err        error  // Underlying error type (e.g., ErrRateLimit)
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error: %s", e.Message)
}

// Unwrap returns the underlying error for errors.Is and errors.As
func (e *APIError) Unwrap() error {
	return e.Err
}

// Is allows error comparison with errors.Is
func (e *APIError) Is(target error) bool {
	if e.Err != nil {
		return errors.Is(e.Err, target)
	}
	return false
}

// IsRetryable returns true if the error is likely transient and worth retrying
func (e *APIError) IsRetryable() bool {
	// Retry on rate limits (429), server errors (5xx), and certain client errors
	if e.StatusCode == http.StatusTooManyRequests {
		return true
	}
	if e.StatusCode >= 500 && e.StatusCode < 600 {
		return true
	}
	// 408 Request Timeout is also retryable
	if e.StatusCode == http.StatusRequestTimeout {
		return true
	}
	return false
}

// NetworkError represents network connectivity errors
type NetworkError struct {
	Message string
	Err     error
}

// Error implements the error interface
func (e *NetworkError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("network error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("network error: %s", e.Message)
}

// Unwrap returns the underlying error
func (e *NetworkError) Unwrap() error {
	return e.Err
}

// Is allows error comparison
func (e *NetworkError) Is(target error) bool {
	return target == ErrNetworkError
}

// ValidationError represents request validation errors
type ValidationError struct {
	Field   string // Field that failed validation
	Message string // Validation error message
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// Is allows error comparison
func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidRequest
}

// newAPIError creates a new APIError from an HTTP response
func newAPIError(statusCode int, message string) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    message,
	}

	// Map status codes to error types
	switch statusCode {
	case http.StatusBadRequest:
		apiErr.Err = ErrInvalidRequest
	case http.StatusUnauthorized, http.StatusForbidden:
		apiErr.Err = ErrAuthentication
	case http.StatusNotFound:
		apiErr.Err = ErrNotFound
	case http.StatusTooManyRequests:
		apiErr.Err = ErrRateLimit
	default:
		if statusCode >= 500 {
			apiErr.Err = ErrServerError
		}
	}

	return apiErr
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRetryable()
	}

	// Network errors are generally retryable
	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return true
	}

	// Context errors are not retryable
	if errors.Is(err, ErrContextCanceled) || errors.Is(err, ErrContextDeadlineExceeded) {
		return false
	}

	return false
}
