package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// GetGeneration retrieves detailed statistics for a specific generation
// Use the ID from the ChatCompletionResponse to get actual token counts and costs
func (c *Client) GetGeneration(ctx context.Context, generationID string) (*GenerationStats, error) {
	if generationID == "" {
		return nil, fmt.Errorf("generation ID is required")
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var genResp GenerationResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
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
