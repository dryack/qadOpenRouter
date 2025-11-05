package openrouter

import (
	"context"
	"math"
	"time"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// InitialDelay is the initial delay before the first retry
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases after each retry
	Multiplier float64

	// ShouldRetry is an optional custom function to determine if an error should be retried
	// If nil, uses IsRetryableError
	ShouldRetry func(error) bool
}

// DefaultRetryConfig returns a retry configuration with sensible defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		ShouldRetry:  nil, // Use default IsRetryableError
	}
}

// WithRetry is a client option to enable automatic retries
func WithRetry(config RetryConfig) ClientOption {
	return func(c *Client) {
		c.retryConfig = &config
	}
}

// retryWithBackoff executes a function with exponential backoff retry logic
func retryWithBackoff(ctx context.Context, config *RetryConfig, operation func() error) error {
	if config == nil || config.MaxRetries == 0 {
		// No retry configured, execute once
		return operation()
	}

	var lastErr error
	shouldRetry := config.ShouldRetry
	if shouldRetry == nil {
		shouldRetry = IsRetryableError
	}

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the operation
		err := operation()

		// Success!
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !shouldRetry(err) {
			return err
		}

		// Don't retry on the last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Calculate backoff delay with exponential increase
		delay := config.InitialDelay
		if attempt > 0 {
			delay = time.Duration(float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt)))
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}

		// Wait for the delay, respecting context cancellation
		select {
		case <-time.After(delay):
			// Continue to next retry
		case <-ctx.Done():
			// Context canceled, return immediately
			return ctx.Err()
		}
	}

	// All retries exhausted
	return lastErr
}
