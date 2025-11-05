package openrouter

import (
	"context"
	"fmt"
	"strconv"
)

// GetModelByID finds a model by its ID
func (c *Client) GetModelByID(ctx context.Context, id string) (*Model, error) {
	models, err := c.GetModels(ctx)
	if err != nil {
		return nil, err
	}

	for _, model := range models {
		if model.ID == id {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", id)
}

// GetModelsByProvider returns all models from a specific provider
func (c *Client) GetModelsByProvider(ctx context.Context, provider string) ([]Model, error) {
	models, err := c.GetModels(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []Model
	for _, model := range models {
		// Provider is the first part of the ID before the "/"
		if len(model.ID) > len(provider) && model.ID[:len(provider)] == provider {
			filtered = append(filtered, model)
		}
	}

	return filtered, nil
}

// CalculatePromptCost calculates the cost for a given number of prompt tokens
func CalculatePromptCost(model Model, tokens int) (float64, error) {
	pricePerToken, err := strconv.ParseFloat(model.Pricing.Prompt, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid prompt price: %w", err)
	}
	return pricePerToken * float64(tokens), nil
}

// CalculateCompletionCost calculates the cost for a given number of completion tokens
func CalculateCompletionCost(model Model, tokens int) (float64, error) {
	pricePerToken, err := strconv.ParseFloat(model.Pricing.Completion, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid completion price: %w", err)
	}
	return pricePerToken * float64(tokens), nil
}

// CalculateTotalCost calculates the total cost for prompt and completion tokens
func CalculateTotalCost(model Model, promptTokens, completionTokens int) (float64, error) {
	promptCost, err := CalculatePromptCost(model, promptTokens)
	if err != nil {
		return 0, err
	}

	completionCost, err := CalculateCompletionCost(model, completionTokens)
	if err != nil {
		return 0, err
	}

	return promptCost + completionCost, nil
}

// GetRequestCost returns the per-request cost if applicable
func GetRequestCost(model Model) (float64, error) {
	cost, err := strconv.ParseFloat(model.Pricing.Request, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid request price: %w", err)
	}
	return cost, nil
}

// GetImageCost returns the per-image cost if applicable
func GetImageCost(model Model) (float64, error) {
	cost, err := strconv.ParseFloat(model.Pricing.Image, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid image price: %w", err)
	}
	return cost, nil
}

// PricingInfo provides a human-readable summary of model pricing
type PricingInfo struct {
	PromptPer1M     float64
	CompletionPer1M float64
	PerRequest      float64
	PerImage        float64
}

// GetPricingInfo returns pricing information formatted per 1M tokens
func GetPricingInfo(model Model) (*PricingInfo, error) {
	promptPerToken, err := strconv.ParseFloat(model.Pricing.Prompt, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid prompt price: %w", err)
	}

	completionPerToken, err := strconv.ParseFloat(model.Pricing.Completion, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid completion price: %w", err)
	}

	requestCost, err := strconv.ParseFloat(model.Pricing.Request, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid request price: %w", err)
	}

	imageCost, err := strconv.ParseFloat(model.Pricing.Image, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid image price: %w", err)
	}

	return &PricingInfo{
		PromptPer1M:     promptPerToken * 1_000_000,
		CompletionPer1M: completionPerToken * 1_000_000,
		PerRequest:      requestCost,
		PerImage:        imageCost,
	}, nil
}
