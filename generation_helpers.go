package openrouter

import (
	"fmt"
	"time"
)

// GetGenerationWithRetry attempts to fetch generation stats with retries
// Generation stats may not be available immediately after completion
func (c *Client) GetGenerationWithRetry(generationID string, maxRetries int, retryDelay time.Duration) (*GenerationStats, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		stats, err := c.GetGeneration(generationID)
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
			time.Sleep(retryDelay)
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
	return contains(errMsg, "404") || contains(errMsg, "not found")
}

// contains checks if a string contains a substring (case-insensitive check would be better)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
