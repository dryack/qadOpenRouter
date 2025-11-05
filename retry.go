package openrouter

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds configuration for retry behavior with exponential backoff.
// The retry logic implements exponential backoff with jitter to avoid thundering herd problems.
// Jitter randomizes delays between 50%-100% of the calculated exponential delay.
//
// Example retry delays with default configuration (before jitter):
//   - Attempt 1: 1 second (actual: 0.5-1.0s with jitter)
//   - Attempt 2: 2 seconds (actual: 1.0-2.0s with jitter)
//   - Attempt 3: 4 seconds (actual: 2.0-4.0s with jitter)
//   - Attempt 4: 8 seconds (actual: 4.0-8.0s with jitter, capped at MaxDelay)
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

// ValidateRetryConfig validates the retry configuration.
// This function can be used to validate retry configurations before passing them to WithRetry.
// Returns ValidationError with specific field information for each validation failure.
//
// Example:
//
//	config := openrouter.RetryConfig{
//		MaxRetries:   3,
//		InitialDelay: 1 * time.Second,
//		MaxDelay:     30 * time.Second,
//		Multiplier:   2.0,
//	}
//	if err := openrouter.ValidateRetryConfig(&config); err != nil {
//		// Handle validation error
//	}
func ValidateRetryConfig(config *RetryConfig) error {
	if config.MaxRetries < 0 {
		return &ValidationError{
			Field:   "MaxRetries",
			Message: "must be non-negative",
		}
	}
	if config.InitialDelay <= 0 {
		return &ValidationError{
			Field:   "InitialDelay",
			Message: "must be positive",
		}
	}
	if config.MaxDelay < config.InitialDelay {
		return &ValidationError{
			Field:   "MaxDelay",
			Message: "must be greater than or equal to InitialDelay",
		}
	}
	if config.Multiplier <= 0 {
		return &ValidationError{
			Field:   "Multiplier",
			Message: "must be positive",
		}
	}
	return nil
}

// WithRetry is a client option to enable automatic retries.
// The configuration will be validated when first used.
//
// See also: ValidateRetryConfig for pre-validation, DefaultRetryConfig for sensible defaults
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

	// Validate configuration before use
	if err := ValidateRetryConfig(config); err != nil {
		return err
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

		// Apply jitter to prevent thundering herd (equal jitter: 50%-100% of calculated delay)
		// This randomizes retry timing across multiple clients
		jitter := delay/2 + time.Duration(rand.Int63n(int64(delay/2)))
		delay = jitter

		// Wait for the delay, respecting context cancellation
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			// Continue to next retry
		case <-ctx.Done():
			// Context canceled, return immediately
			timer.Stop()
			return ctx.Err()
		}
		timer.Stop()
	}

	// All retries exhausted
	return lastErr
}
