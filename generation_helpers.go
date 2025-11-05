package openrouter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// GetGenerationWithRetry attempts to fetch generation stats with retries
// Generation stats may not be available immediately after completion
func (c *Client) GetGenerationWithRetry(ctx context.Context, generationID string, maxRetries int, retryDelay time.Duration) (*GenerationStats, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		stats, err := c.GetGeneration(ctx, generationID)
		if err == nil {
			return stats, nil
		}

		lastErr = err

		// If it's a 404, the stats aren't ready yet - wait and retry
		// For other errors, return immediately
		if !isNotFoundError(err) {
			return nil, err
		}

		// Don't sleep after the last attempt
		if i < maxRetries-1 {
			// Use context-aware sleep to allow cancellation
			timer := time.NewTimer(retryDelay)
			select {
			case <-timer.C:
				// Continue to next retry
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			}
			timer.Stop()
		}
	}

	return nil, fmt.Errorf("generation stats not available after %d retries: %w", maxRetries, lastErr)
}

// isNotFoundError checks if the error is a 404 not found error using proper error type checking
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's an APIError with 404 status code
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}

	// Check if it's wrapped ErrNotFound
	return errors.Is(err, ErrNotFound)
}
