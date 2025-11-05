package openrouter

import (
	"errors"
	"net/http"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: 400,
		Message:    "Bad request",
		Err:        ErrInvalidRequest,
	}

	expected := "API error (status 400): Bad request"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	err := &APIError{
		StatusCode: 400,
		Message:    "Bad request",
		Err:        ErrInvalidRequest,
	}

	if !errors.Is(err, ErrInvalidRequest) {
		t.Error("Expected error to unwrap to ErrInvalidRequest")
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"429 Too Many Requests", http.StatusTooManyRequests, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
		{"502 Bad Gateway", http.StatusBadGateway, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
		{"504 Gateway Timeout", http.StatusGatewayTimeout, true},
		{"408 Request Timeout", http.StatusRequestTimeout, true},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"401 Unauthorized", http.StatusUnauthorized, false},
		{"403 Forbidden", http.StatusForbidden, false},
		{"404 Not Found", http.StatusNotFound, false},
		{"200 OK", http.StatusOK, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{
				StatusCode: tt.statusCode,
				Message:    "test",
				Err:        nil,
			}
			if got := err.IsRetryable(); got != tt.want {
				t.Errorf("APIError.IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNetworkError_Error(t *testing.T) {
	innerErr := errors.New("connection refused")
	err := &NetworkError{
		Message: "failed to connect",
		Err:     innerErr,
	}

	expected := "network error: failed to connect: connection refused"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	innerErr := errors.New("connection refused")
	err := &NetworkError{
		Message: "failed to connect",
		Err:     innerErr,
	}

	if !errors.Is(err, innerErr) {
		t.Error("Expected error to unwrap to inner error")
	}
}

func TestNetworkError_Is(t *testing.T) {
	err := &NetworkError{
		Message: "network error",
		Err:     errors.New("test"),
	}

	if !errors.Is(err, ErrNetworkError) {
		t.Error("NetworkError should match ErrNetworkError")
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "model",
		Message: "model is required",
	}

	expected := "validation error for field 'model': model is required"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestValidationError_Is(t *testing.T) {
	err := &ValidationError{
		Field:   "model",
		Message: "model is required",
	}

	if !errors.Is(err, ErrInvalidRequest) {
		t.Error("ValidationError should match ErrInvalidRequest")
	}
}

func TestNewAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
	}{
		{"400 Bad Request", http.StatusBadRequest, "invalid input", ErrInvalidRequest},
		{"401 Unauthorized", http.StatusUnauthorized, "unauthorized", ErrAuthentication},
		{"403 Forbidden", http.StatusForbidden, "forbidden", ErrAuthentication},
		{"404 Not Found", http.StatusNotFound, "not found", ErrNotFound},
		{"429 Too Many Requests", http.StatusTooManyRequests, "rate limit", ErrRateLimit},
		{"500 Internal Server Error", http.StatusInternalServerError, "internal error", ErrServerError},
		{"502 Bad Gateway", http.StatusBadGateway, "bad gateway", ErrServerError},
		{"503 Service Unavailable", http.StatusServiceUnavailable, "unavailable", ErrServerError},
		{"418 I'm a teapot", http.StatusTeapot, "teapot", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := newAPIError(tt.statusCode, tt.body)

			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, apiErr.StatusCode)
			}

			if tt.wantErr != nil && !errors.Is(apiErr, tt.wantErr) {
				t.Errorf("Expected error to wrap %v", tt.wantErr)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Retryable API error (500)",
			err: &APIError{
				StatusCode: http.StatusInternalServerError,
				Message:    "server error",
				Err:        ErrServerError,
			},
			want: true,
		},
		{
			name: "Retryable API error (429)",
			err: &APIError{
				StatusCode: http.StatusTooManyRequests,
				Message:    "rate limited",
				Err:        ErrRateLimit,
			},
			want: true,
		},
		{
			name: "Non-retryable API error (400)",
			err: &APIError{
				StatusCode: http.StatusBadRequest,
				Message:    "bad request",
				Err:        ErrInvalidRequest,
			},
			want: false,
		},
		{
			name: "Network error",
			err: &NetworkError{
				Message: "connection failed",
				Err:     errors.New("test"),
			},
			want: true,
		},
		{
			name: "Validation error",
			err: &ValidationError{
				Field:   "model",
				Message: "required",
			},
			want: false,
		},
		{
			name: "Context canceled",
			err:  ErrContextCanceled,
			want: false,
		},
		{
			name: "Context deadline exceeded",
			err:  ErrContextDeadlineExceeded,
			want: false,
		},
		{
			name: "Generic error",
			err:  errors.New("generic error"),
			want: false,
		},
		{
			name: "Nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.want {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}
