// Package openrouter provides a Go client library for the OpenRouter API.
//
// OpenRouter is a unified interface for LLMs (Large Language Models) that provides
// access to multiple AI models through a single API. This package simplifies
// interaction with OpenRouter's endpoints, providing:
//
// - Model discovery and pricing information
// - Chat completion requests with full parameter support
// - Generation statistics and cost tracking
// - Automatic retry logic with exponential backoff
// - Request/response logging
// - Rate limiting support
// - Context-based cancellation and timeouts
//
// # Basic Usage
//
// Create a client and make a simple chat completion request:
//
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-api-key"),
//	)
//
//	ctx := context.Background()
//	req := openrouter.ChatCompletionRequest{
//		Model: "anthropic/claude-3.5-sonnet",
//		Messages: []openrouter.Message{
//			openrouter.NewUserMessage("Hello, how are you?"),
//		},
//	}
//
//	resp, err := client.CreateChatCompletion(ctx, req)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Println(resp.Choices[0].Message.Content)
//
// # Client Configuration
//
// The client supports extensive configuration through functional options:
//
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-api-key"),
//		openrouter.WithTimeout(60 * time.Second),
//		openrouter.WithCacheTTL(30 * time.Minute),
//		openrouter.WithRateLimiter(rate.NewLimiter(rate.Every(time.Second), 10)),
//		openrouter.WithStandardLogger(true, true),  // Log bodies and headers
//	)
//
// # Error Handling
//
// The package provides structured error types for better error handling:
//
//	resp, err := client.CreateChatCompletion(ctx, req)
//	if err != nil {
//		var apiErr *openrouter.APIError
//		if errors.As(err, &apiErr) {
//			fmt.Printf("API error (status %d): %s\n", apiErr.StatusCode, apiErr.Message)
//			if apiErr.IsRetryable() {
//				// Retry logic here
//			}
//		}
//
//		var netErr *openrouter.NetworkError
//		if errors.As(err, &netErr) {
//			fmt.Printf("Network error: %s\n", netErr.Message)
//		}
//
//		var valErr *openrouter.ValidationError
//		if errors.As(err, &valErr) {
//			fmt.Printf("Validation error for %s: %s\n", valErr.Field, valErr.Message)
//		}
//	}
//
// Cache validation errors for programmatic error handling:
//
//	err := client.LoadCacheFromFile("cache.json")
//	if err != nil {
//		if errors.Is(err, openrouter.ErrNilCache) {
//			fmt.Println("Cache is nil")
//		} else if errors.Is(err, openrouter.ErrInvalidExpiration) {
//			fmt.Println("Cache has invalid expiration time")
//		}
//	}
//
// Available cache validation errors:
//   - ErrNilCache: cached models pointer is nil
//   - ErrNilModels: models array is nil
//   - ErrZeroFetchedAt: FetchedAt timestamp is zero/unset
//   - ErrZeroExpiresAt: ExpiresAt timestamp is zero/unset
//   - ErrInvalidExpiration: ExpiresAt is before FetchedAt
//
// # Automatic Retry
//
// The client automatically retries transient errors (network errors, 5xx responses,
// 429 rate limits) using exponential backoff:
//
//	retryConfig := openrouter.DefaultRetryConfig()
//	retryConfig.MaxRetries = 5
//	retryConfig.InitialDelay = 2 * time.Second
//
//	// Validate configuration before use (since v2.0)
//	if err := openrouter.ValidateRetryConfig(&retryConfig); err != nil {
//		log.Fatal(err)
//	}
//
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-api-key"),
//		openrouter.WithRetry(retryConfig),
//	)
//
// # Logging
//
// Enable request/response logging for debugging:
//
//	// Standard logger to stderr (default 500 char truncation)
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-api-key"),
//		openrouter.WithStandardLogger(true, true),  // Log bodies and headers
//	)
//
//	// Custom truncation for large payloads (since v2.0)
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-api-key"),
//		openrouter.WithStandardLoggerTruncate(true, true, 2000),  // 2000 char limit
//	)
//
//	// Unlimited logging for debugging (not recommended for production)
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-api-key"),
//		openrouter.WithStandardLoggerTruncate(true, true, 0),  // No truncation
//	)
//
//	// Custom logger
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-api-key"),
//		openrouter.WithCustomLogger(func(format string, args ...interface{}) {
//			log.Printf(format, args...)
//		}),
//	)
//
// # Model Discovery
//
// Fetch available models and their pricing:
//
//	models, err := client.GetModels(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for _, model := range models {
//		fmt.Printf("%s: $%.4f per 1K tokens\n",
//			model.ID,
//			model.Pricing.Prompt,
//		)
//	}
//
// # Cost Tracking
//
// Track costs across multiple models:
//
//	tracker := openrouter.NewCostTracker()
//
//	resp, err := client.CreateChatCompletion(ctx, req)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Get detailed generation stats
//	stats, err := client.GetGeneration(ctx, resp.ID)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Record the result for cost tracking
//	tracker.Record(openrouter.InferenceResult{
//		Model:            req.Model,
//		GenerationID:     resp.ID,
//		PromptTokens:     stats.TokensPrompt,
//		CompletionTokens: stats.TokensCompletion,
//		TotalTokens:      stats.TokensPrompt + stats.TokensCompletion,
//		ActualCost:       stats.TotalCost,
//		Timestamp:        time.Now(),
//	})
//
// # Memory-Bounded Cost Tracking
//
// Prevent unbounded memory growth in production (since v2.0):
//
//	// Unlimited tracking (default, backward compatible)
//	tracker := openrouter.NewCostTracker()
//
//	// Bounded tracking (recommended for production)
//	// Keeps only the last 1000 results per model
//	tracker := openrouter.NewCostTrackerWithLimit(1000)
//
// The bounded tracker automatically removes oldest results when the limit is reached,
// using a sliding window approach to maintain recent data while preventing memory leaks.
//
// # Context Support
//
// All API calls support context for cancellation and timeouts:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	resp, err := client.CreateChatCompletion(ctx, req)
//	if errors.Is(err, openrouter.ErrContextDeadlineExceeded) {
//		fmt.Println("Request timed out")
//	}
//
// # Caching
//
// Model data is automatically cached to reduce API calls:
//
//	// Uses cache if available
//	models, _ := client.GetModels(ctx)
//
//	// Force fresh fetch
//	models, _ := client.GetModelsFresh(ctx)
//
//	// Save cache to file
//	client.SaveCacheToFile("cache.json")
//
//	// Load cache from file
//	client.LoadCacheFromFile("cache.json")
//
//	// Validate cache data before loading (since v2.0)
//	cached := &openrouter.CachedModels{...}
//	if err := openrouter.ValidateCachedModels(cached); err != nil {
//		log.Printf("Invalid cache: %v", err)
//	}
//
// # Advanced Cache Configuration
//
// Configure custom file permissions for sensitive cache data (since v2.0):
//
//	// Default permissions (0644 - readable by all users)
//	client := openrouter.NewClient(
//		openrouter.WithCacheTTL(1 * time.Hour),
//	)
//
//	// Secure permissions (0600 - user-only read/write)
//	client := openrouter.NewClient(
//		openrouter.WithCacheFileMode(1*time.Hour, 0600),
//	)
//
// # Security Best Practices
//
// - Store API keys in environment variables, not in code
// - Use HTTPS for all API communication (default)
// - Enable logging only in development environments
// - Authorization headers are automatically redacted in logs
// - Validate all user inputs before making API calls
// - Use context timeouts to prevent hanging requests
//
// For more information, visit: https://openrouter.ai/docs
package openrouter
