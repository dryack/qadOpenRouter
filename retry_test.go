package openrouter

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.MaxRetries)
	}

	if config.InitialDelay != 1*time.Second {
		t.Errorf("Expected InitialDelay 1s, got %v", config.InitialDelay)
	}

	if config.MaxDelay != 30*time.Second {
		t.Errorf("Expected MaxDelay 30s, got %v", config.MaxDelay)
	}

	if config.Multiplier != 2.0 {
		t.Errorf("Expected Multiplier 2.0, got %f", config.Multiplier)
	}

	if config.ShouldRetry != nil {
		t.Error("Expected ShouldRetry to be nil")
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	callCount := 0
	operation := func() error {
		callCount++
		return nil
	}

	err := retryWithBackoff(ctx, &config, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 100 * time.Millisecond

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return &NetworkError{
				Message: "temporary failure",
				Err:     errors.New("network error"),
			}
		}
		return nil
	}

	err := retryWithBackoff(ctx, &config, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 100 * time.Millisecond

	callCount := 0
	operation := func() error {
		callCount++
		return &NetworkError{
			Message: "persistent failure",
			Err:     errors.New("network error"),
		}
	}

	err := retryWithBackoff(ctx, &config, operation)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Should be called MaxRetries+1 times (initial + 3 retries)
	if callCount != 4 {
		t.Errorf("Expected 4 calls, got %d", callCount)
	}
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	callCount := 0
	operation := func() error {
		callCount++
		return &ValidationError{
			Field:   "model",
			Message: "model is required",
		}
	}

	err := retryWithBackoff(ctx, &config, operation)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Should only be called once since validation errors are not retryable
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultRetryConfig()
	config.InitialDelay = 100 * time.Millisecond

	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 1 {
			// Cancel context after first call
			cancel()
			return &NetworkError{
				Message: "network error",
				Err:     errors.New("temporary"),
			}
		}
		return nil
	}

	err := retryWithBackoff(ctx, &config, operation)
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	// Should only be called once since context is canceled
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestRetryWithBackoff_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	config := DefaultRetryConfig()
	config.InitialDelay = 100 * time.Millisecond

	callCount := 0
	operation := func() error {
		callCount++
		return &NetworkError{
			Message: "network error",
			Err:     errors.New("temporary"),
		}
	}

	err := retryWithBackoff(ctx, &config, operation)
	if err == nil {
		t.Error("Expected error due to context timeout")
	}

	// Should only be called once or twice before timeout
	if callCount > 2 {
		t.Errorf("Expected at most 2 calls, got %d", callCount)
	}
}

func TestRetryWithBackoff_CustomShouldRetry(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	// Custom retry predicate that only retries on specific error
	customErr := errors.New("custom retryable error")
	config.ShouldRetry = func(err error) bool {
		return errors.Is(err, customErr)
	}

	t.Run("Retry custom error", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			if callCount < 2 {
				return customErr
			}
			return nil
		}

		err := retryWithBackoff(ctx, &config, operation)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if callCount != 2 {
			t.Errorf("Expected 2 calls, got %d", callCount)
		}
	})

	t.Run("Don't retry different error", func(t *testing.T) {
		callCount := 0
		operation := func() error {
			callCount++
			return errors.New("different error")
		}

		err := retryWithBackoff(ctx, &config, operation)
		if err == nil {
			t.Error("Expected error, got nil")
		}

		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}
	})
}

func TestRetryWithBackoff_NilConfig(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		return nil
	}

	err := retryWithBackoff(ctx, nil, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should use default config
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestRetryWithBackoff_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond
	config.MaxDelay = 100 * time.Millisecond
	config.Multiplier = 2.0

	delays := []time.Duration{}
	lastCall := time.Now()

	callCount := 0
	operation := func() error {
		callCount++
		if callCount > 1 {
			delay := time.Since(lastCall)
			delays = append(delays, delay)
		}
		lastCall = time.Now()

		if callCount < 4 {
			return &APIError{
				StatusCode: http.StatusInternalServerError,
				Message:    "server error",
				Err:        ErrServerError,
			}
		}
		return nil
	}

	err := retryWithBackoff(ctx, &config, operation)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify exponential backoff with jitter
	if len(delays) < 2 {
		t.Fatal("Expected at least 2 delays")
	}

	// First delay should be between 50%-100% of InitialDelay (due to jitter)
	minFirstDelay := config.InitialDelay / 2
	maxFirstDelay := config.InitialDelay
	if delays[0] < minFirstDelay || delays[0] > maxFirstDelay {
		t.Errorf("First delay %v is outside jittered range [%v, %v]", delays[0], minFirstDelay, maxFirstDelay)
	}

	// Second delay should be between 50%-100% of (2 * InitialDelay) due to exponential backoff + jitter
	// This means between 1.0 * InitialDelay and 2.0 * InitialDelay
	minSecondDelay := config.InitialDelay
	maxSecondDelay := config.InitialDelay * 2
	if delays[1] < minSecondDelay || delays[1] > maxSecondDelay {
		t.Errorf("Second delay %v is outside jittered range [%v, %v]", delays[1], minSecondDelay, maxSecondDelay)
	}
}

func TestRetryWithBackoff_APIError429(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return &APIError{
				StatusCode: http.StatusTooManyRequests,
				Message:    "rate limited",
				Err:        ErrRateLimit,
			}
		}
		return nil
	}

	err := retryWithBackoff(ctx, &config, operation)
	if err != nil {
		t.Errorf("Expected no error after retries, got %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestValidateRetryConfig_ValidConfig(t *testing.T) {
	config := DefaultRetryConfig()
	err := ValidateRetryConfig(&config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
}

func TestValidateRetryConfig_NegativeMaxRetries(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = -1
	err := ValidateRetryConfig(&config)
	if err == nil {
		t.Error("Expected error for negative MaxRetries")
	}
}

func TestValidateRetryConfig_ZeroInitialDelay(t *testing.T) {
	config := DefaultRetryConfig()
	config.InitialDelay = 0
	err := ValidateRetryConfig(&config)
	if err == nil {
		t.Error("Expected error for zero InitialDelay")
	}
}

func TestValidateRetryConfig_NegativeInitialDelay(t *testing.T) {
	config := DefaultRetryConfig()
	config.InitialDelay = -1 * time.Second
	err := ValidateRetryConfig(&config)
	if err == nil {
		t.Error("Expected error for negative InitialDelay")
	}
}

func TestValidateRetryConfig_MaxDelayLessThanInitial(t *testing.T) {
	config := DefaultRetryConfig()
	config.InitialDelay = 10 * time.Second
	config.MaxDelay = 5 * time.Second
	err := ValidateRetryConfig(&config)
	if err == nil {
		t.Error("Expected error for MaxDelay < InitialDelay")
	}
}

func TestValidateRetryConfig_ZeroMultiplier(t *testing.T) {
	config := DefaultRetryConfig()
	config.Multiplier = 0
	err := ValidateRetryConfig(&config)
	if err == nil {
		t.Error("Expected error for zero Multiplier")
	}
}

func TestValidateRetryConfig_NegativeMultiplier(t *testing.T) {
	config := DefaultRetryConfig()
	config.Multiplier = -1.0
	err := ValidateRetryConfig(&config)
	if err == nil {
		t.Error("Expected error for negative Multiplier")
	}
}

func TestRetryWithBackoff_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.MaxRetries = -1 // Invalid

	operation := func() error {
		return nil
	}

	err := retryWithBackoff(ctx, &config, operation)
	if err == nil {
		t.Error("Expected validation error for invalid config")
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Expected ValidationError, got %T", err)
	}
}
