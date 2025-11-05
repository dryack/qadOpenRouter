package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	openrouter "github.com/dryack/openRouterPricing"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}

	fmt.Println("=== OpenRouter A/B Testing Demo ===\n")

	// Create client with API key
	client := openrouter.NewClient(
		openrouter.WithAPIKey(apiKey),
	)

	// Create cost tracker
	tracker := openrouter.NewCostTracker()

	// Define test prompts
	testPrompts := []string{
		"What is the capital of France?",
		"Explain quantum computing in simple terms.",
		"Write a haiku about programming.",
	}

	// Define models to test
	modelsToTest := []string{
		"openai/gpt-3.5-turbo",
		"anthropic/claude-3-haiku",
		"meta-llama/llama-3-8b-instruct",
	}

	fmt.Printf("Testing %d models with %d prompts each...\n\n", len(modelsToTest), len(testPrompts))

	// Run tests for each model
	for _, model := range modelsToTest {
		fmt.Printf("Testing %s...\n", model)

		for i, prompt := range testPrompts {
			fmt.Printf("  Prompt %d/%d: ", i+1, len(testPrompts))

			result := runInference(client, model, prompt)
			tracker.Record(result)

			if result.Error != nil {
				fmt.Printf("❌ Error: %v\n", result.Error)
			} else {
				fmt.Printf("✓ Tokens: %d, Cost: $%.6f, Latency: %v\n",
					result.TotalTokens, result.ActualCost, result.Latency.Round(time.Millisecond))
			}

			// Small delay between requests
			time.Sleep(500 * time.Millisecond)
		}
		fmt.Println()
	}

	// Generate comparison report
	fmt.Println("\n" + strings.Repeat("=", 60))
	report := tracker.Compare()
	fmt.Print(report.FormatReport())

	// Show individual model details
	fmt.Println("=== Detailed Model Analysis ===\n")
	for _, model := range modelsToTest {
		stats := tracker.GetModelStats(model)
		if stats == nil {
			continue
		}

		fmt.Printf("%s:\n", model)
		fmt.Printf("  Success Rate: %.1f%%\n", stats.SuccessRate())
		fmt.Printf("  Average Cost per Request: $%.6f\n", stats.AvgCostPerRequest())

		if stats.SuccessCount > 0 {
			costPer1kTokens := (stats.TotalActualCost / float64(stats.TotalTokens)) * 1000
			fmt.Printf("  Cost per 1K tokens: $%.6f\n", costPer1kTokens)
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Total Cost: $%.6f\n", tracker.GetTotalCost())
	fmt.Printf("Recommendation: Use %s for best cost-efficiency\n", report.BestCostModel)
	fmt.Printf("                Use %s for best latency\n", report.BestLatencyModel)
}

// runInference performs a single inference and tracks costs
func runInference(client *openrouter.Client, model, prompt string) openrouter.InferenceResult {
	result := openrouter.InferenceResult{
		Model:     model,
		Prompt:    prompt,
		Timestamp: time.Now(),
	}

	// Start timing
	start := time.Now()

	// Create chat completion request
	req := openrouter.ChatCompletionRequest{
		Model: model,
		Messages: []openrouter.Message{
			openrouter.NewUserMessage(prompt),
		},
	}

	// Execute inference
	resp, err := client.CreateChatCompletion(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.Error = err
		return result
	}

	// Extract response
	if len(resp.Choices) > 0 {
		if content, ok := resp.Choices[0].Message.Content.(string); ok {
			result.Response = content
		}
	}

	// Get token counts from response
	result.PromptTokens = resp.Usage.PromptTokens
	result.CompletionTokens = resp.Usage.CompletionTokens
	result.TotalTokens = resp.Usage.TotalTokens
	result.GenerationID = resp.ID

	// Calculate estimated cost from pricing data
	modelInfo, err := client.GetModelByID(model)
	if err == nil {
		promptCost, _ := openrouter.CalculatePromptCost(*modelInfo, result.PromptTokens)
		completionCost, _ := openrouter.CalculateCompletionCost(*modelInfo, result.CompletionTokens)
		result.EstimatedCost = promptCost + completionCost
	}

	// Get actual cost from generation stats
	// Note: Stats may take a few seconds to become available, using retry logic
	stats, err := client.GetGenerationWithRetry(resp.ID, 3, 1*time.Second)
	if err == nil {
		result.ActualCost = stats.TotalCost
	} else {
		// Fallback to estimated cost if stats not available yet
		result.ActualCost = result.EstimatedCost
	}

	return result
}
