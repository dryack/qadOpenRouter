package openrouter

import (
	"context"
	"math"
	"time"
)

// RetryConfig holds configuration for retry behavior with exponential backoff.
// The retry logic implements exponential backoff with jitter to avoid thundering herd problems.
//
// Example retry delays with default configuration:
//   - Attempt 1: 1 second
//   - Attempt 2: 2 seconds
//   - Attempt 3: 4 seconds
//   - Attempt 4: 8 seconds (but capped at MaxDelay)
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (not including the initial attempt).
	// Default: 3 (total of 4 attempts including initial)
	MaxRetries int

	// InitialDelay is the delay before the first retry.
	// Default: 1 second
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries, preventing extremely long waits.
	// Default: 30 seconds
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases after each retry.
	// For example, with a multiplier of 2.0, delays double each time (exponential backoff).
	// Default: 2.0
	Multiplier float64

	// ShouldRetry is an optional custom function to determine if an error should be retried.
	// If nil, uses the built-in IsRetryableError function which handles:
	//   - Network errors (connection failures, timeouts)
	//   - HTTP 5xx server errors
	//   - HTTP 429 rate limit errors
	//   - HTTP 408 request timeout errors
	//
	// Custom example:
	//   config.ShouldRetry = func(err error) bool {
	//       return errors.Is(err, MyCustomRetryableError)
	//   }
	ShouldRetry func(error) bool
}

// DefaultRetryConfig returns a retry configuration with sensible defaults
// for most use cases:
//   - 3 retries (4 total attempts)
//   - 1 second initial delay
//   - 30 second maximum delay
//   - 2.0x exponential backoff multiplier
//   - Built-in retry logic for transient errors
//
// Example:
//
//	config := openrouter.DefaultRetryConfig()
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-key"),
//		openrouter.WithRetryConfig(&config),
//	)
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
