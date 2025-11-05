# OpenRouter Go Client

A comprehensive Go package for interacting with the OpenRouter API, featuring model pricing, chat completions, and cost tracking for A/B testing.

> ⚠️ **Production Readiness Update**: Recent improvements have addressed critical issues (concurrency deadlock, context support, rate limiting). See [Production Readiness](#production-readiness) for details.

## ⚠️ Breaking Changes in v2.0

**All API methods now require `context.Context` as the first parameter:**

```go
// OLD (v1.x)
models, err := client.GetModels()

// NEW (v2.0+)
ctx := context.Background()
models, err := client.GetModels(ctx)
```

**Migration Guide:**
1. Add `import "context"` to your code
2. Create a context: `ctx := context.Background()` or `ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)`
3. Pass context as first parameter to all API methods
4. See [examples/](example/) for complete updated examples

## Features

### Core Features
- 🚀 Fetch all available models from OpenRouter API
- 💾 Intelligent caching with configurable TTL (default: 1 hour)
- 💾 File-based cache persistence
- 🔒 Thread-safe cache operations (deadlock fixed)
- 💰 Pricing calculation utilities
- 🎯 Model lookup and filtering functions

### Inference & Cost Tracking
- 🤖 OpenAI-compatible chat completions API
- 📊 Generation stats with actual token counts and costs
- 📈 Built-in cost tracking for A/B testing
- 🔬 Compare costs and latency across models
- 💸 Estimated vs actual cost comparison

### Request Control & Safety
- ✅ **NEW** Context-based cancellation and timeouts
- ✅ **NEW** Optional rate limiting to prevent API abuse
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
    "context"
    "fmt"
    "log"

    openrouter "github.com/dryack/openRouterPricing"
)

func main() {
    // Create a new client
    client := openrouter.NewClient()

    // Create context for API calls
    ctx := context.Background()

    // Get all models (uses cache when available)
    models, err := client.GetModels(ctx)
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

// With rate limiting (requires golang.org/x/time/rate)
limiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 requests per second
client := openrouter.NewClient(
    openrouter.WithAPIKey("your-api-key"),
    openrouter.WithRateLimiter(limiter),
)
```

### Fetching Models

```go
ctx := context.Background()

// Get models (uses cache if available and not expired)
models, err := client.GetModels(ctx)

// Force fresh fetch from API
models, err := client.GetModelsFresh(ctx)
```

### Finding Specific Models

```go
ctx := context.Background()

// Find by ID
model, err := client.GetModelByID(ctx, "openai/gpt-4")

// Get all models from a provider
anthropicModels, err := client.GetModelsByProvider(ctx, "anthropic")
```

### Pricing Calculations

```go
ctx := context.Background()
model, _ := client.GetModelByID(ctx, "openai/gpt-4")

// Calculate cost for specific token counts
promptTokens := 1000
completionTokens := 500
totalCost, err := openrouter.CalculateTotalCost(*model, promptTokens, completionTokens)

// Get pricing info formatted per 1M tokens
pricingInfo, err := openrouter.GetPricingInfo(*model)
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
ctx := context.Background()

// Try to load existing cache
err := client.LoadCacheFromFile(cacheFile)
if err != nil {
    // Cache doesn't exist or expired, fetch fresh data
    models, err := client.GetModelsFresh(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Save for next time
    client.SaveCacheToFile(cacheFile)
} else {
    // Successfully loaded from file
    models, _ := client.GetModels(ctx)
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

ctx := context.Background()

// Simple inference
req := openrouter.ChatCompletionRequest{
    Model: "openai/gpt-3.5-turbo",
    Messages: []openrouter.Message{
        openrouter.NewUserMessage("What is the capital of France?"),
    },
}

resp, err := client.CreateChatCompletion(ctx, req)
if err != nil {
    log.Fatal(err)
}

// Access response
response := resp.Choices[0].Message.Content
fmt.Printf("Response: %v\n", response)
fmt.Printf("Tokens used: %d\n", resp.Usage.TotalTokens)

// Get actual cost information (with retry, as stats may not be immediately available)
stats, err := client.GetGenerationWithRetry(ctx, resp.ID, 5, 2*time.Second)
if err == nil {
    fmt.Printf("Actual cost: $%.6f\n", stats.TotalCost)
    fmt.Printf("Provider: %s\n", stats.ProviderName)
}
```

### A/B Testing & Cost Comparison

Track and compare costs across different models:

```go
ctx := context.Background()

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
    resp, err := client.CreateChatCompletion(ctx, req)
    latency := time.Since(start)

    if err != nil {
        continue
    }

    // Get actual cost (with retry)
    stats, _ := client.GetGenerationWithRetry(ctx, resp.ID, 5, 2*time.Second)

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
- `WithRateLimiter(limiter *rate.Limiter)` - Set rate limiter to prevent API abuse

### Client Methods

**Model & Pricing:**
- `GetModels(ctx context.Context) ([]Model, error)` - Get models (cached)
- `GetModelsFresh(ctx context.Context) ([]Model, error)` - Get models (fresh)
- `GetModelByID(ctx context.Context, id string) (*Model, error)` - Find model by ID
- `GetModelsByProvider(ctx context.Context, provider string) ([]Model, error)` - Filter by provider

**Cache Management:**
- `ClearCache()` - Clear cached data
- `IsCacheExpired() bool` - Check cache expiration
- `CacheTimeRemaining() time.Duration` - Get remaining cache time
- `SaveCacheToFile(filepath string) error` - Save cache to JSON file
- `LoadCacheFromFile(filepath string) error` - Load cache from JSON file

**Inference:**
- `CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)` - Run chat completion
- `GetGeneration(ctx context.Context, generationID string) (*GenerationStats, error)` - Get generation stats and actual costs
- `GetGenerationWithRetry(ctx context.Context, generationID string, maxRetries int, retryDelay time.Duration) (*GenerationStats, error)` - Get generation stats with automatic retry (recommended)
- `GetGenerationCost(ctx context.Context, generationID string) (float64, error)` - Get just the cost for a generation

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

1. **Use Context Properly**: Always pass appropriate context to API calls. Use `context.WithTimeout()` for time-bounded operations and `context.WithCancel()` when you need explicit cancellation control
2. **Reuse Client Instances**: Create one client and reuse it across your application
3. **Configure Appropriate TTL**: Set cache TTL based on your needs (pricing doesn't change frequently)
4. **Handle Errors**: Always check error returns, especially for network operations
5. **Use Cached Reads**: Use `GetModels()` instead of `GetModelsFresh()` when possible
6. **Rate Limiting**: Use `WithRateLimiter()` to prevent API abuse and respect rate limits
7. **Track Costs**: Use `CostTracker` for A/B testing to compare model performance and costs
8. **Get Actual Costs**: Use `GetGenerationWithRetry()` to retrieve actual costs after inference
9. **File Caching**: Save cache to disk for faster startup and reduced API calls

## Security Best Practices

### API Key Handling

Your OpenRouter API key provides access to paid services and should be protected:

**DO:**
- ✅ Store API keys in environment variables: `os.Getenv("OPENROUTER_API_KEY")`
- ✅ Use secure secret management services (AWS Secrets Manager, HashiCorp Vault, etc.) in production
- ✅ Restrict API key permissions to minimum required scope
- ✅ Rotate API keys periodically
- ✅ Use different API keys for development, staging, and production environments
- ✅ Add API keys to `.gitignore` to prevent accidental commits
- ✅ Implement rate limiting to prevent API abuse

**DON'T:**
- ❌ Hard-code API keys in source code
- ❌ Commit API keys to version control
- ❌ Share API keys via email or messaging apps
- ❌ Expose API keys in client-side code (web browsers, mobile apps)
- ❌ Log API keys in application logs
- ❌ Use production API keys for development/testing

### Example: Secure API Key Loading

```go
package main

import (
    "fmt"
    "log"
    "os"

    openrouter "github.com/dryack/openRouterPricing"
)

func main() {
    // Load API key from environment variable
    apiKey := os.Getenv("OPENROUTER_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENROUTER_API_KEY environment variable is required")
    }

    // Create client with API key
    client := openrouter.NewClient(
        openrouter.WithAPIKey(apiKey),
    )

    // Use client...
}
```

### Setting Environment Variables

**Linux/macOS:**
```bash
export OPENROUTER_API_KEY="your-api-key-here"
```

**Windows (Command Prompt):**
```cmd
set OPENROUTER_API_KEY=your-api-key-here
```

**Windows (PowerShell):**
```powershell
$env:OPENROUTER_API_KEY="your-api-key-here"
```

**Docker:**
```dockerfile
ENV OPENROUTER_API_KEY=""
```

Or pass at runtime:
```bash
docker run -e OPENROUTER_API_KEY="your-key" your-image
```

**Kubernetes Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openrouter-secret
type: Opaque
stringData:
  api-key: your-api-key-here
```

### Memory Security Note

API keys are currently stored in plain memory within the `Client` struct. For highly sensitive environments, consider:
- Using short-lived API keys with automatic rotation
- Implementing additional encryption at the application level
- Running in secure, isolated environments
- Monitoring API usage for anomalies

## Default Configuration

- **Base URL**: `https://openrouter.ai/api/v1`
- **Cache TTL**: 1 hour
- **HTTP Timeout**: 30 seconds

## Production Readiness

This library has undergone significant improvements to address production readiness concerns:

### Issues Resolved

**CRITICAL:**
- ✅ **Deadlock bug fixed** in `cost_tracking.go`: Removed unlock/relock pattern that could cause deadlocks during concurrent access to `GetAllModelStats()`

**HIGH Priority:**
- ✅ **Context support added**: All API methods now accept `context.Context` for proper cancellation and timeout control
- ✅ **Rate limiting support**: Optional rate limiting via `WithRateLimiter()` to prevent API abuse
- ✅ **Nil pointer protection**: Fixed `WithTimeout()` to handle nil `httpClient` safely

**MEDIUM Priority:**
- ✅ **Response body handling optimized**: Standardized to read body once per request for better efficiency
- ✅ **Test coverage expanded**: Added comprehensive tests for `inference.go` and `generation.go`
- ✅ **Security documentation added**: Comprehensive API key handling best practices and examples

**Testing:**
- ✅ **Comprehensive test coverage**: Added 402+ lines of tests across all critical paths
- ✅ **All tests passing**: 49 tests covering concurrency, context, rate limiting, inference, and generation APIs

### Breaking Changes

**v2.0** introduces breaking changes to add context support. See [Breaking Changes in v2.0](#️-breaking-changes-in-v20) for migration guide.

### Remaining Considerations

**MEDIUM Priority:**
- API keys are stored in plain memory - see [Security Best Practices](#security-best-practices) for guidance
- No built-in retry logic for transient network errors (implement at application level if needed)

**LOW Priority:**
- Additional test coverage could be added for edge cases and error scenarios

The library is now production-ready with proper error handling, context management, rate limiting, comprehensive test coverage, and security best practices documentation.

## License

MIT
