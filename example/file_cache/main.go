package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	openrouter "github.com/dryack/qadOpenRouter"
)

const cacheFilePath = "openrouter_cache.json"

func main() {
	fmt.Println("=== File-Based Cache Example ===")

	// Create context for API calls
	ctx := context.Background()

	// Create a new client
	client := openrouter.NewClient(
		openrouter.WithCacheTTL(24 * time.Hour), // Cache for 24 hours
	)

	// Try to load cache from file first
	fmt.Println("Attempting to load cache from file...")
	err := client.LoadCacheFromFile(cacheFilePath)
	if err != nil {
		fmt.Printf("Could not load cache from file: %v\n", err)
		fmt.Println("Fetching fresh data from API...")

		// Fetch fresh data if cache file doesn't exist or is expired
		models, err := client.GetModelsFresh(ctx)
		if err != nil {
			log.Fatalf("Error fetching models: %v", err)
		}

		fmt.Printf("✓ Retrieved %d models from API\n", len(models))

		// Save to file for next time
		fmt.Println("Saving cache to file...")
		err = client.SaveCacheToFile(cacheFilePath)
		if err != nil {
			log.Printf("Warning: Failed to save cache: %v", err)
		} else {
			fmt.Printf("✓ Cache saved to %s\n", cacheFilePath)
		}
	} else {
		fmt.Println("✓ Successfully loaded cache from file!")

		// Get models from in-memory cache (loaded from file)
		models, err := client.GetModels(ctx)
		if err != nil {
			log.Fatalf("Error getting models: %v", err)
		}

		fmt.Printf("✓ Using cached data with %d models\n", len(models))
		fmt.Printf("Cache valid for: %v\n", client.CacheTimeRemaining().Round(time.Minute))
	}

	// Display some model information
	fmt.Println("\n=== Sample Models ===")
	models, _ := client.GetModels(ctx)

	// Show first 5 models
	count := 5
	if len(models) < count {
		count = len(models)
	}

	for i := 0; i < count; i++ {
		model := models[i]
		pricingInfo, _ := openrouter.GetPricingInfo(model)
		fmt.Printf("\n%d. %s\n", i+1, model.Name)
		fmt.Printf("   ID: %s\n", model.ID)
		fmt.Printf("   Context: %d tokens\n", model.ContextLength)
		fmt.Printf("   Pricing: $%.2f/$%.2f per 1M tokens (prompt/completion)\n",
			pricingInfo.PromptPer1M, pricingInfo.CompletionPer1M)
	}

	// Example: Force refresh and save
	fmt.Println("\n=== Force Refresh ===")
	fmt.Println("To force a refresh, delete the cache file and run again, or:")
	fmt.Println("  1. Call client.GetModelsFresh() to fetch fresh data")
	fmt.Println("  2. Call client.SaveCacheToFile(path) to save updated cache")

	// Show file info
	if info, err := os.Stat(cacheFilePath); err == nil {
		fmt.Printf("\nCache file size: %.2f KB\n", float64(info.Size())/1024)
		fmt.Printf("Last modified: %v\n", info.ModTime().Format(time.RFC3339))
	}

	fmt.Println("\n✓ Done! Cache will be reused on next run if still valid.")
}
