package main

import (
	"fmt"
	"log"
	"time"

	openrouter "github.com/dryack/openRouterPricing"
)

func main() {
	// Create a new client with default settings (1 hour cache TTL)
	client := openrouter.NewClient()

	// Or create a client with custom options
	// client := openrouter.NewClient(
	// 	openrouter.WithCacheTTL(30 * time.Minute),
	// 	openrouter.WithTimeout(10 * time.Second),
	// 	openrouter.WithAPIKey("your-api-key"), // Optional
	// )

	fmt.Println("Fetching OpenRouter models...")

	// Get all models (uses cache if available)
	models, err := client.GetModels()
	if err != nil {
		log.Fatalf("Error fetching models: %v", err)
	}

	fmt.Printf("Retrieved %d models\n\n", len(models))

	// Display cache information
	if !client.IsCacheExpired() {
		fmt.Printf("Cache valid for: %v\n\n", client.CacheTimeRemaining().Round(time.Second))
	}

	// Example 1: Find a specific model
	fmt.Println("=== Example 1: Find specific model ===")
	model, err := client.GetModelByID("openai/gpt-4")
	if err != nil {
		log.Printf("Model not found: %v", err)
	} else {
		displayModelInfo(model)
	}

	// Example 2: Get all models from a provider
	fmt.Println("\n=== Example 2: Models from Anthropic ===")
	anthropicModels, err := client.GetModelsByProvider("anthropic")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Found %d Anthropic models:\n", len(anthropicModels))
	for _, m := range anthropicModels {
		fmt.Printf("  - %s\n", m.Name)
	}

	// Example 3: Calculate costs
	if len(models) > 0 {
		fmt.Println("\n=== Example 3: Calculate costs ===")
		exampleModel := models[0]

		promptTokens := 1000
		completionTokens := 500

		totalCost, err := openrouter.CalculateTotalCost(exampleModel, promptTokens, completionTokens)
		if err != nil {
			log.Printf("Error calculating cost: %v", err)
		} else {
			fmt.Printf("Model: %s\n", exampleModel.Name)
			fmt.Printf("Cost for %d prompt + %d completion tokens: $%.6f\n",
				promptTokens, completionTokens, totalCost)
		}

		// Get pricing info per 1M tokens
		pricingInfo, err := openrouter.GetPricingInfo(exampleModel)
		if err != nil {
			log.Printf("Error getting pricing info: %v", err)
		} else {
			fmt.Printf("Pricing per 1M tokens:\n")
			fmt.Printf("  Prompt: $%.2f\n", pricingInfo.PromptPer1M)
			fmt.Printf("  Completion: $%.2f\n", pricingInfo.CompletionPer1M)
			if pricingInfo.PerRequest > 0 {
				fmt.Printf("  Per Request: $%.4f\n", pricingInfo.PerRequest)
			}
		}
	}

	// Example 4: Force fresh fetch
	fmt.Println("\n=== Example 4: Force refresh ===")
	fmt.Println("Fetching fresh data from API...")
	freshModels, err := client.GetModelsFresh()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Retrieved %d models (fresh)\n", len(freshModels))
	fmt.Printf("New cache valid for: %v\n", client.CacheTimeRemaining().Round(time.Second))

	// Example 5: Clear cache manually
	fmt.Println("\n=== Example 5: Clear cache ===")
	client.ClearCache()
	fmt.Println("Cache cleared")
	fmt.Printf("Cache expired: %v\n", client.IsCacheExpired())
}

func displayModelInfo(model *openrouter.Model) {
	fmt.Printf("Model ID: %s\n", model.ID)
	fmt.Printf("Name: %s\n", model.Name)
	fmt.Printf("Context Length: %d\n", model.ContextLength)
	fmt.Printf("Description: %s\n", model.Description)
	fmt.Printf("\nPricing:\n")
	fmt.Printf("  Prompt: $%s per token\n", model.Pricing.Prompt)
	fmt.Printf("  Completion: $%s per token\n", model.Pricing.Completion)

	if model.Pricing.Request != "0" {
		fmt.Printf("  Request: $%s per request\n", model.Pricing.Request)
	}
	if model.Pricing.Image != "0" {
		fmt.Printf("  Image: $%s per image\n", model.Pricing.Image)
	}
}
