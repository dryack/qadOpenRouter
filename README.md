# OpenRouter Go Client

A comprehensive Go package for interacting with the OpenRouter API, featuring model pricing, chat completions, and cost tracking for A/B testing.

(100% pure AI slop to give me a tool I wanted immediately, with the absolute minimum effort.  Do NOT rely on this code without deeply examining it.)

## Features

### Model & Pricing
- 🚀 Fetch all available models from OpenRouter API
- 💾 Intelligent caching with configurable TTL (default: 1 hour)
- 💾 File-based cache persistence
- 🔒 Thread-safe cache operations
- 💰 Pricing calculation utilities
- 🎯 Model lookup and filtering functions

### Inference & Cost Tracking
- 🤖 OpenAI-compatible chat completions API
- 📊 Generation stats with actual token counts and costs
- 📈 Built-in cost tracking for A/B testing
- 🔬 Compare costs and latency across models
- 💸 Estimated vs actual cost comparison

### Configuration
- ⚙️ Configurable HTTP client and timeouts
- 🔑 API key support for authenticated requests

## Installation

```bash
go get github.com/dryack/openRouterPricing
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    openrouter "github.com/dryack/openRouterPricing"
)

func main() {
    // Create a new client
    client := openrouter.NewClient()

    // Get all models (uses cache when available)
    models, err := client.GetModels()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d models\n", len(models))
}
```

## Usage Examples

### Basic Client Setup

```go
// Default client (1 hour cache, 30s timeout)
client := openrouter.NewClient()

// Custom configuration
client := openrouter.NewClient(
    openrouter.WithCacheTTL(30 * time.Minute),
    openrouter.WithTimeout(10 * time.Second),
    openrouter.WithAPIKey("your-api-key"),
)
```

### Fetching Models

```go
// Get models (uses cache if available and not expired)
models, err := client.GetModels()

// Force fresh fetch from API
models, err := client.GetModelsFresh()
```

### Finding Specific Models

```go
// Find by ID
model, err := client.GetModelByID("openai/gpt-4")

// Get all models from a provider
anthropicModels, err := client.GetModelsByProvider("anthropic")
```

### Pricing Calculations

```go
model, _ := client.GetModelByID("openai/gpt-4")

// Calculate cost for specific token counts
promptTokens := 1000
completionTokens := 500
totalCost, err := openrouter.CalculateTotalCost(model, promptTokens, completionTokens)

// Get pricing info formatted per 1M tokens
pricingInfo, err := openrouter.GetPricingInfo(model)
fmt.Printf("Prompt: $%.2f per 1M tokens\n", pricingInfo.PromptPer1M)
fmt.Printf("Completion: $%.2f per 1M tokens\n", pricingInfo.CompletionPer1M)
```

### Cache Management

```go
// Check if cache is expired
if client.IsCacheExpired() {
    fmt.Println("Cache needs refresh")
}

// Check time remaining
remaining := client.CacheTimeRemaining()
fmt.Printf("Cache valid for: %v\n", remaining)

// Clear cache manually
client.ClearCache()
```

### File-Based Cache Persistence

Save and load cache to/from disk to persist across application restarts:

```go
const cacheFile = "openrouter_cache.json"

// Try to load existing cache
err := client.LoadCacheFromFile(cacheFile)
if err != nil {
    // Cache doesn't exist or expired, fetch fresh data
    models, err := client.GetModelsFresh()
    if err != nil {
        log.Fatal(err)
    }

    // Save for next time
    client.SaveCacheToFile(cacheFile)
} else {
    // Successfully loaded from file
    models, _ := client.GetModels()
    fmt.Printf("Using cached data with %d models\n", len(models))
}
```

Benefits of file-based caching:
- **Persistence** - Cache survives application restarts
- **Reduced API calls** - Especially useful for dev/testing
- **Offline capability** - Work with cached data without internet
- **Faster startup** - No need to fetch on every application start

See `example/file_cache/main.go` for a complete working example.

### Chat Completions (Inference)

Run chat completions and get actual cost data:

```go
// Requires API key
client := openrouter.NewClient(
    openrouter.WithAPIKey("your-api-key"),
)

// Simple inference
req := openrouter.ChatCompletionRequest{
    Model: "openai/gpt-3.5-turbo",
    Messages: []openrouter.Message{
        openrouter.NewUserMessage("What is the capital of France?"),
    },
}

resp, err := client.CreateChatCompletion(req)
if err != nil {
    log.Fatal(err)
}

// Access response
response := resp.Choices[0].Message.Content
fmt.Printf("Response: %v\n", response)
fmt.Printf("Tokens used: %d\n", resp.Usage.TotalTokens)

// Get actual cost information (with retry, as stats may not be immediately available)
stats, err := client.GetGenerationWithRetry(resp.ID, 5, 2*time.Second)
if err == nil {
    fmt.Printf("Actual cost: $%.6f\n", stats.TotalCost)
    fmt.Printf("Provider: %s\n", stats.ProviderName)
}
```

### A/B Testing & Cost Comparison

Track and compare costs across different models:

```go
// Create cost tracker
tracker := openrouter.NewCostTracker()

// Test different models
models := []string{
    "openai/gpt-3.5-turbo",
    "anthropic/claude-3-haiku",
    "meta-llama/llama-3-8b-instruct",
}

for _, model := range models {
    req := openrouter.ChatCompletionRequest{
        Model: model,
        Messages: []openrouter.Message{
            openrouter.NewUserMessage("Explain AI in one sentence."),
        },
    }

    start := time.Now()
    resp, err := client.CreateChatCompletion(req)
    latency := time.Since(start)

    if err != nil {
        continue
    }

    // Get actual cost (with retry)
    stats, _ := client.GetGenerationWithRetry(resp.ID, 5, 2*time.Second)

    // Record result
    tracker.Record(openrouter.InferenceResult{
        Model:            model,
        PromptTokens:     resp.Usage.PromptTokens,
        CompletionTokens: resp.Usage.CompletionTokens,
        TotalTokens:      resp.Usage.TotalTokens,
        ActualCost:       stats.TotalCost,
        Latency:          latency,
        Timestamp:        time.Now(),
    })
}

// Generate comparison report
report := tracker.Compare()
fmt.Printf("Best cost: %s\n", report.BestCostModel)
fmt.Printf("Best latency: %s\n", report.BestLatencyModel)

// Get detailed stats for a model
stats := tracker.GetModelStats("openai/gpt-3.5-turbo")
fmt.Printf("Average cost per request: $%.6f\n", stats.AvgCostPerRequest())
fmt.Printf("Success rate: %.1f%%\n", stats.SuccessRate())
```

See `example/ab_testing/main.go` for a complete A/B testing example.

## API Reference

### Client Options

- `WithCacheTTL(ttl time.Duration)` - Set cache time-to-live
- `WithTimeout(timeout time.Duration)` - Set HTTP request timeout
- `WithAPIKey(apiKey string)` - Set OpenRouter API key
- `WithBaseURL(baseURL string)` - Set custom API base URL
- `WithHTTPClient(client *http.Client)` - Use custom HTTP client

### Client Methods

**Model & Pricing:**
- `GetModels() ([]Model, error)` - Get models (cached)
- `GetModelsFresh() ([]Model, error)` - Get models (fresh)
- `GetModelByID(id string) (*Model, error)` - Find model by ID
- `GetModelsByProvider(provider string) ([]Model, error)` - Filter by provider

**Cache Management:**
- `ClearCache()` - Clear cached data
- `IsCacheExpired() bool` - Check cache expiration
- `CacheTimeRemaining() time.Duration` - Get remaining cache time
- `SaveCacheToFile(filepath string) error` - Save cache to JSON file
- `LoadCacheFromFile(filepath string) error` - Load cache from JSON file

**Inference:**
- `CreateChatCompletion(req ChatCompletionRequest) (*ChatCompletionResponse, error)` - Run chat completion
- `GetGeneration(generationID string) (*GenerationStats, error)` - Get generation stats and actual costs
- `GetGenerationWithRetry(generationID string, maxRetries int, retryDelay time.Duration) (*GenerationStats, error)` - Get generation stats with automatic retry (recommended)
- `GetGenerationCost(generationID string) (float64, error)` - Get just the cost for a generation

### Pricing Functions

- `CalculatePromptCost(model Model, tokens int) (float64, error)`
- `CalculateCompletionCost(model Model, tokens int) (float64, error)`
- `CalculateTotalCost(model Model, promptTokens, completionTokens int) (float64, error)`
- `GetRequestCost(model Model) (float64, error)`
- `GetImageCost(model Model) (float64, error)`
- `GetPricingInfo(model Model) (*PricingInfo, error)`

### Cost Tracking (A/B Testing)

**CostTracker:**
- `NewCostTracker() *CostTracker` - Create a new cost tracker
- `Record(result InferenceResult)` - Record an inference result
- `GetModelStats(model string) *ModelStats` - Get stats for a specific model
- `GetAllModelStats() map[string]*ModelStats` - Get stats for all models
- `Compare() *ComparisonReport` - Generate comparison report
- `GetTotalCost() float64` - Get total cost across all models
- `Clear()` - Clear all tracked results

**Helper Functions:**
- `NewMessage(role, content string) Message` - Create a message
- `NewUserMessage(content string) Message` - Create user message
- `NewAssistantMessage(content string) Message` - Create assistant message
- `NewSystemMessage(content string) Message` - Create system message

## Data Structures

### Model

```go
type Model struct {
    ID              string
    Name            string
    Description     string
    ContextLength   int
    Pricing         Pricing
    Architecture    Architecture
    // ... additional fields
}
```

### Pricing

```go
type Pricing struct {
    Prompt            string  // USD per prompt token
    Completion        string  // USD per completion token
    Request           string  // USD per request
    Image             string  // USD per image
    WebSearch         string  // USD per web search
    InternalReasoning string  // USD per reasoning token
    InputCacheRead    string  // USD per cached input read
}
```

## Running the Examples

### Basic Examples

```bash
# Pricing and model information
cd example
go run main.go

# File-based caching
cd example/file_cache
go run main.go
```

### Inference Examples (Requires API Key)

```bash
# Set your API key
export OPENROUTER_API_KEY="your-key-here"

# Simple inference
cd example/simple_inference
go run main.go

# A/B testing with cost tracking
cd example/ab_testing
go run main.go
```

## Best Practices

1. **Reuse Client Instances**: Create one client and reuse it across your application
2. **Configure Appropriate TTL**: Set cache TTL based on your needs (pricing doesn't change frequently)
3. **Handle Errors**: Always check error returns, especially for network operations
4. **Use Cached Reads**: Use `GetModels()` instead of `GetModelsFresh()` when possible
5. **Track Costs**: Use `CostTracker` for A/B testing to compare model performance and costs
6. **Get Actual Costs**: Use `GetGeneration()` to retrieve actual costs after inference
7. **File Caching**: Save cache to disk for faster startup and reduced API calls

## Default Configuration

- **Base URL**: `https://openrouter.ai/api/v1`
- **Cache TTL**: 1 hour
- **HTTP Timeout**: 30 seconds

## License

MIT
