package main

import (
	"fmt"
	"log"
	"os"
	"time"

	openrouter "github.com/dryack/openRouterPricing"
)

// displayRequest shows the request being sent to the API
func displayRequest(req openrouter.ChatCompletionRequest) {
	fmt.Println("\n📤 Request:")
	fmt.Printf("  Model: %s\n", req.Model)

	if req.Temperature != nil {
		fmt.Printf("  Temperature: %.1f\n", *req.Temperature)
	}
	if req.MaxTokens != nil {
		fmt.Printf("  Max Tokens: %d\n", *req.MaxTokens)
	}

	fmt.Println("  Messages:")
	for i, msg := range req.Messages {
		content := ""
		if str, ok := msg.Content.(string); ok {
			content = str
			// Truncate if too long
			if len(content) > 100 {
				content = content[:97] + "..."
			}
		}
		fmt.Printf("    [%d] %s: %s\n", i+1, msg.Role, content)
	}
	fmt.Println()
}

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}

	fmt.Println("=== Simple Inference Example ===")
	fmt.Println()

	// Create client with API key
	client := openrouter.NewClient(
		openrouter.WithAPIKey(apiKey),
	)

	// Example 1: Simple chat completion
	fmt.Println("Example 1: Simple Question")
	fmt.Println("----------------------------")

	req := openrouter.ChatCompletionRequest{
		Model: "openai/gpt-3.5-turbo",
		Messages: []openrouter.Message{
			openrouter.NewUserMessage("What is the capital of France?"),
		},
	}

	displayRequest(req)

	resp, err := client.CreateChatCompletion(req)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Display response
	if len(resp.Choices) > 0 {
		fmt.Printf("📥 Response: %v\n", resp.Choices[0].Message.Content)
	}

	fmt.Printf("\nUsage:\n")
	fmt.Printf("  Prompt tokens: %d\n", resp.Usage.PromptTokens)
	fmt.Printf("  Completion tokens: %d\n", resp.Usage.CompletionTokens)
	fmt.Printf("  Total tokens: %d\n", resp.Usage.TotalTokens)

	// Get actual cost from generation stats
	// Note: Stats may take a few seconds to become available
	fmt.Println("\nFetching actual cost data (this may take a few seconds)...")

	stats, err := client.GetGenerationWithRetry(resp.ID, 5, 2*time.Second)
	if err != nil {
		log.Printf("Warning: Could not fetch generation stats: %v", err)
		log.Printf("This is normal - generation stats may not be available immediately.")
	} else {
		fmt.Printf("\nCost Information:\n")
		fmt.Printf("  Prompt cost: $%.6f\n", stats.NativeCostsPrompt)
		fmt.Printf("  Completion cost: $%.6f\n", stats.NativeCostsCompletion)
		fmt.Printf("  Total cost: $%.6f\n", stats.TotalCost)
		fmt.Printf("  Provider: %s\n", stats.ProviderName)
	}

	// Example 2: Multi-turn conversation
	fmt.Println("\n\nExample 2: Multi-turn Conversation")
	fmt.Println("-----------------------------------")

	conversationReq := openrouter.ChatCompletionRequest{
		Model: "anthropic/claude-3-haiku",
		Messages: []openrouter.Message{
			openrouter.NewSystemMessage("You are a helpful assistant that responds concisely."),
			openrouter.NewUserMessage("Tell me a fun fact about space."),
			openrouter.NewAssistantMessage("A day on Venus is longer than its year! Venus takes 243 Earth days to rotate once, but only 225 Earth days to orbit the Sun."),
			openrouter.NewUserMessage("That's interesting! Tell me another one."),
		},
	}

	displayRequest(conversationReq)

	convResp, err := client.CreateChatCompletion(conversationReq)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if len(convResp.Choices) > 0 {
		fmt.Printf("📥 Response: %v\n", convResp.Choices[0].Message.Content)
	}

	fmt.Printf("\nUsage:\n")
	fmt.Printf("  Prompt tokens: %d\n", convResp.Usage.PromptTokens)
	fmt.Printf("  Completion tokens: %d\n", convResp.Usage.CompletionTokens)
	fmt.Printf("  Total tokens: %d\n", convResp.Usage.TotalTokens)

	// Get actual cost
	if convStats, err := client.GetGenerationWithRetry(convResp.ID, 5, 2*time.Second); err == nil {
		fmt.Printf("\nCost: $%.6f (Provider: %s)\n", convStats.TotalCost, convStats.ProviderName)
	}

	// Example 3: With parameters
	fmt.Println("\n\nExample 3: With Custom Parameters")
	fmt.Println("----------------------------------")

	temperature := 0.7
	maxTokens := 100

	paramReq := openrouter.ChatCompletionRequest{
		Model:       "meta-llama/llama-3-8b-instruct",
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Messages: []openrouter.Message{
			openrouter.NewUserMessage("Write a creative tagline for a coffee shop."),
		},
	}

	displayRequest(paramReq)

	paramResp, err := client.CreateChatCompletion(paramReq)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if len(paramResp.Choices) > 0 {
		fmt.Printf("📥 Response: %v\n", paramResp.Choices[0].Message.Content)
	}

	fmt.Printf("\nUsage:\n")
	fmt.Printf("  Prompt tokens: %d\n", paramResp.Usage.PromptTokens)
	fmt.Printf("  Completion tokens: %d\n", paramResp.Usage.CompletionTokens)
	fmt.Printf("  Total tokens: %d (max tokens: %d)\n",
		paramResp.Usage.TotalTokens, maxTokens)

	// Get actual cost
	if paramStats, err := client.GetGenerationWithRetry(paramResp.ID, 5, 2*time.Second); err == nil {
		fmt.Printf("\nCost: $%.6f (Provider: %s)\n", paramStats.TotalCost, paramStats.ProviderName)
	}

	// Example 4: Compare estimated vs actual costs
	fmt.Println("\n\nExample 4: Cost Estimation vs Actual")
	fmt.Println("-------------------------------------")

	testModel := "openai/gpt-4"
	testReq := openrouter.ChatCompletionRequest{
		Model: testModel,
		Messages: []openrouter.Message{
			openrouter.NewUserMessage("Explain AI in one sentence."),
		},
	}

	displayRequest(testReq)

	testResp, err := client.CreateChatCompletion(testReq)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if len(testResp.Choices) > 0 {
		fmt.Printf("📥 Response: %v\n", testResp.Choices[0].Message.Content)
	}

	// Calculate estimated cost
	var estimatedCost float64
	modelInfo, err := client.GetModelByID(testModel)
	if err == nil {
		promptCost, _ := openrouter.CalculatePromptCost(*modelInfo, testResp.Usage.PromptTokens)
		completionCost, _ := openrouter.CalculateCompletionCost(*modelInfo, testResp.Usage.CompletionTokens)
		estimatedCost = promptCost + completionCost

		fmt.Printf("Estimated Cost: $%.6f\n", estimatedCost)
	}

	// Get actual cost
	if testStats, err := client.GetGenerationWithRetry(testResp.ID, 5, 2*time.Second); err == nil {
		fmt.Printf("Actual Cost: $%.6f\n", testStats.TotalCost)
		if estimatedCost > 0 {
			diff := testStats.TotalCost - estimatedCost
			fmt.Printf("Difference: $%.6f", diff)
			if diff > 0 {
				fmt.Printf(" (%.1f%% higher)\n", (diff/estimatedCost)*100)
			} else if diff < 0 {
				fmt.Printf(" (%.1f%% lower)\n", ((-diff)/estimatedCost)*100)
			} else {
				fmt.Println(" (exact match)")
			}
		}
	}

	fmt.Println("\n✓ All examples completed!")
}
