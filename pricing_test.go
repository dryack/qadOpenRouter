package openrouter

import (
	"testing"
)

func TestCalculatePromptCost(t *testing.T) {
	model := Model{
		Pricing: Pricing{
			Prompt: "0.000001", // $1 per 1M tokens
		},
	}

	cost, err := CalculatePromptCost(model, 1000)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := 0.001 // 1000 tokens at $0.000001 per token
	if cost != expected {
		t.Errorf("Expected cost %f, got %f", expected, cost)
	}
}

func TestCalculateCompletionCost(t *testing.T) {
	model := Model{
		Pricing: Pricing{
			Completion: "0.000002", // $2 per 1M tokens
		},
	}

	cost, err := CalculateCompletionCost(model, 1000)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := 0.002 // 1000 tokens at $0.000002 per token
	if cost != expected {
		t.Errorf("Expected cost %f, got %f", expected, cost)
	}
}

func TestCalculateTotalCost(t *testing.T) {
	model := Model{
		Pricing: Pricing{
			Prompt:     "0.000001", // $1 per 1M tokens
			Completion: "0.000002", // $2 per 1M tokens
		},
	}

	cost, err := CalculateTotalCost(model, 1000, 500)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// (1000 * 0.000001) + (500 * 0.000002) = 0.001 + 0.001 = 0.002
	expected := 0.002
	if cost != expected {
		t.Errorf("Expected total cost %f, got %f", expected, cost)
	}
}

func TestGetRequestCost(t *testing.T) {
	model := Model{
		Pricing: Pricing{
			Request: "0.05",
		},
	}

	cost, err := GetRequestCost(model)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := 0.05
	if cost != expected {
		t.Errorf("Expected request cost %f, got %f", expected, cost)
	}
}

func TestGetImageCost(t *testing.T) {
	model := Model{
		Pricing: Pricing{
			Image: "0.01",
		},
	}

	cost, err := GetImageCost(model)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := 0.01
	if cost != expected {
		t.Errorf("Expected image cost %f, got %f", expected, cost)
	}
}

func TestGetPricingInfo(t *testing.T) {
	model := Model{
		Pricing: Pricing{
			Prompt:     "0.000001", // $1 per 1M tokens
			Completion: "0.000002", // $2 per 1M tokens
			Request:    "0.05",
			Image:      "0.01",
		},
	}

	info, err := GetPricingInfo(model)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if info.PromptPer1M != 1.0 {
		t.Errorf("Expected PromptPer1M to be 1.0, got %f", info.PromptPer1M)
	}

	if info.CompletionPer1M != 2.0 {
		t.Errorf("Expected CompletionPer1M to be 2.0, got %f", info.CompletionPer1M)
	}

	if info.PerRequest != 0.05 {
		t.Errorf("Expected PerRequest to be 0.05, got %f", info.PerRequest)
	}

	if info.PerImage != 0.01 {
		t.Errorf("Expected PerImage to be 0.01, got %f", info.PerImage)
	}
}

func TestCalculatePromptCost_InvalidPrice(t *testing.T) {
	model := Model{
		Pricing: Pricing{
			Prompt: "invalid",
		},
	}

	_, err := CalculatePromptCost(model, 1000)
	if err == nil {
		t.Error("Expected error for invalid price")
	}
}

func TestGetPricingInfo_InvalidPrice(t *testing.T) {
	model := Model{
		Pricing: Pricing{
			Prompt:     "invalid",
			Completion: "0.000002",
			Request:    "0",
			Image:      "0",
		},
	}

	_, err := GetPricingInfo(model)
	if err == nil {
		t.Error("Expected error for invalid prompt price")
	}
}
