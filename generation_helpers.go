package openrouter

import (
	"context"
	"fmt"
	"strings"
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
			select {
			case <-time.After(retryDelay):
				// Continue to next retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("generation stats not available after %d retries: %w", maxRetries, lastErr)
}

// isNotFoundError checks if the error is a 404 not found error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains "404" or "not found"
	errMsg := err.Error()
	return strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found")
}
