package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GenerationStats represents detailed statistics for a generation
type GenerationStats struct {
	ID                string  `json:"id"`
	Model             string  `json:"model"`
	StreamingLatency  *int    `json:"streaming_latency,omitempty"`
	TotalLatency      *int    `json:"total_latency,omitempty"`
	CreatedAt         string  `json:"created_at"`
	ProviderName      string  `json:"provider_name"`
	TokensPrompt      int     `json:"tokens_prompt"`
	TokensCompletion  int     `json:"tokens_completion"`
	NativeCostsPrompt float64 `json:"native_costs_prompt"`
	NativeCostsCompletion float64 `json:"native_costs_completion"`
	TotalCost         float64 `json:"total_cost"`
	FinishReason      string  `json:"finish_reason,omitempty"`
	Usage             float64 `json:"usage"` // Can be fractional for cost per token
	AppID             *int    `json:"app_id,omitempty"`
	Moderation        interface{} `json:"moderation,omitempty"`
}

// GenerationResponse wraps the generation stats
type GenerationResponse struct {
	Data GenerationStats `json:"data"`
}

// GetGeneration retrieves detailed statistics for a specific generation with automatic retry
// Use the ID from the ChatCompletionResponse to get actual token counts and costs
func (c *Client) GetGeneration(ctx context.Context, generationID string) (*GenerationStats, error) {
	if generationID == "" {
		return nil, &ValidationError{
			Field:   "generationID",
			Message: "generation ID is required",
		}
	}

	// Execute with retry
	var stats *GenerationStats
	err := retryWithBackoff(ctx, c.retryConfig, func() error {
		var execErr error
		stats, execErr = c.executeGetGeneration(ctx, generationID)
		return execErr
	})
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// executeGetGeneration performs the actual generation stats request (internal, for retry)
func (c *Client) executeGetGeneration(ctx context.Context, generationID string) (*GenerationStats, error) {

	// Apply rate limiting if configured
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			// Check if context was canceled
			if ctx.Err() != nil {
				if errors.Is(ctx.Err(), context.Canceled) {
					return nil, ErrContextCanceled
				}
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return nil, ErrContextDeadlineExceeded
				}
			}
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	url := fmt.Sprintf("%s/generation?id=%s", c.baseURL, generationID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add context to request
	req = req.WithContext(ctx)

	// Add API key if provided
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	// Log request
	c.logRequest(req.Method, req.URL.String(), req.Header, nil)

	// Execute request and track duration
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		// Log error
		c.logError(err)

		// Check for context errors
		if ctx.Err() != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil, ErrContextCanceled
			}
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, ErrContextDeadlineExceeded
			}
		}
		// Network error
		return nil, &NetworkError{
			Message: "generation stats request failed",
			Err:     err,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logError(err)
		return nil, &NetworkError{
			Message: "failed to read response",
			Err:     err,
		}
	}

	// Log response
	c.logResponse(resp.StatusCode, resp.Header, body, duration)

	if resp.StatusCode != http.StatusOK {
		apiErr := newAPIError(resp.StatusCode, string(body))
		c.logError(apiErr)
		return nil, apiErr
	}

	var genResp GenerationResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to unmarshal response: %v", err),
			Err:        ErrInvalidRequest,
		}
	}

	return &genResp.Data, nil
}

// GetGenerationCost is a convenience method to get just the cost information
func (c *Client) GetGenerationCost(ctx context.Context, generationID string) (float64, error) {
	stats, err := c.GetGeneration(ctx, generationID)
	if err != nil {
		return 0, err
	}
	return stats.TotalCost, nil
}
