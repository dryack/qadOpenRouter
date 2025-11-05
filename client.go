package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultBaseURL is the default OpenRouter API base URL
	DefaultBaseURL = "https://openrouter.ai/api/v1"

	// DefaultCacheTTL is the default cache time-to-live (1 hour)
	DefaultCacheTTL = 1 * time.Hour

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
)

// Client is the main OpenRouter API client
type Client struct {
	baseURL     string
	httpClient  *http.Client
	cache       *Cache
	apiKey      string        // Optional API key for authenticated requests
	rateLimiter *rate.Limiter // Optional rate limiter for API calls
	retryConfig *RetryConfig  // Optional retry configuration for transient errors
	logger      Logger        // Optional logger for request/response logging
}

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL for the API
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithCacheTTL sets a custom cache TTL
func WithCacheTTL(ttl time.Duration) ClientOption {
	return func(c *Client) {
		c.cache = NewCache(ttl)
	}
}

// WithAPIKey sets an API key for authenticated requests
func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

// WithTimeout sets a custom HTTP timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		if c.httpClient == nil {
			c.httpClient = &http.Client{}
		}
		c.httpClient.Timeout = timeout
	}
}

// WithRateLimiter sets a rate limiter for API calls to control request throughput.
// Rate limiting helps prevent hitting API rate limits and manages request concurrency.
//
// Example:
//
//	// Allow 10 requests per second with burst of 10
//	limiter := rate.NewLimiter(rate.Every(time.Second), 10)
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-key"),
//		openrouter.WithRateLimiter(limiter),
//	)
func WithRateLimiter(limiter *rate.Limiter) ClientOption {
	return func(c *Client) {
		c.rateLimiter = limiter
	}
}

// WithRetryConfig sets custom retry behavior for transient errors.
// If not set, retry logic is disabled by default. Use DefaultRetryConfig()
// for sensible defaults, or customize for your needs.
//
// The retry logic handles:
//   - Network errors (connection failures, timeouts)
//   - HTTP 5xx server errors
//   - HTTP 429 rate limit errors
//   - HTTP 408 request timeout errors
//
// Example:
//
//	config := openrouter.DefaultRetryConfig()
//	config.MaxRetries = 5
//	config.InitialDelay = 2 * time.Second
//	config.MaxDelay = 60 * time.Second
//
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-key"),
//		openrouter.WithRetryConfig(&config),
//	)
func WithRetryConfig(config *RetryConfig) ClientOption {
	return func(c *Client) {
		c.retryConfig = config
	}
}

// NewClient creates a new OpenRouter API client with the specified options.
// Returns a client configured with sensible defaults that can be overridden
// via functional options.
//
// Default configuration:
//   - Base URL: https://openrouter.ai/api/v1
//   - HTTP Timeout: 30 seconds
//   - Cache TTL: 1 hour
//   - No rate limiting
//   - No retry logic
//   - No logging
//
// Example:
//
//	client := openrouter.NewClient(
//		openrouter.WithAPIKey("your-api-key"),
//		openrouter.WithTimeout(60 * time.Second),
//		openrouter.WithStandardLogger(true, true),
//	)
func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		cache: NewCache(DefaultCacheTTL),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// GetModels retrieves all available models from OpenRouter
// Returns cached data if available and not expired
func (c *Client) GetModels(ctx context.Context) ([]Model, error) {
	// Check cache first
	if cached, ok := c.cache.Get(); ok {
		return cached.Models, nil
	}

	// Fetch from API with retry
	var models []Model
	err := retryWithBackoff(ctx, c.retryConfig, func() error {
		var fetchErr error
		models, fetchErr = c.fetchModels(ctx)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}

	// Update cache
	c.cache.Set(models)

	return models, nil
}

// GetModelsFresh forces a fresh fetch from the API, bypassing cache
func (c *Client) GetModelsFresh(ctx context.Context) ([]Model, error) {
	// Fetch from API with retry
	var models []Model
	err := retryWithBackoff(ctx, c.retryConfig, func() error {
		var fetchErr error
		models, fetchErr = c.fetchModels(ctx)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}

	// Update cache with fresh data
	c.cache.Set(models)

	return models, nil
}

// fetchModels performs the actual HTTP request to fetch models
func (c *Client) fetchModels(ctx context.Context) ([]Model, error) {
	// Apply rate limiting if configured
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			// Check if context was canceled
			if ctx.Err() != nil {
				if errors.Is(ctx.Err(), context.Canceled) {
					return nil, ErrContextCanceled
				}
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return nil, ErrContextDeadlineExceeded
				}
			}
			return nil, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	url := fmt.Sprintf("%s/models", c.baseURL)

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

	// Log request
	c.logRequest(req.Method, req.URL.String(), req.Header, nil)

	// Execute request and track duration
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		// Log error
		c.logError(err)

		// Check for context errors
		if ctx.Err() != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil, ErrContextCanceled
			}
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, ErrContextDeadlineExceeded
			}
		}
		// Network error
		return nil, &NetworkError{
			Message: "failed to fetch models",
			Err:     err,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logError(err)
		return nil, &NetworkError{
			Message: "failed to read response body",
			Err:     err,
		}
	}

	// Log response
	c.logResponse(resp.StatusCode, resp.Header, body, duration)

	if resp.StatusCode != http.StatusOK {
		apiErr := newAPIError(resp.StatusCode, string(body))
		c.logError(apiErr)
		return nil, apiErr
	}

	var modelsResp ModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to unmarshal response: %v", err),
			Err:        ErrInvalidRequest,
		}
	}

	return modelsResp.Data, nil
}

// ClearCache clears the cached models data
func (c *Client) ClearCache() {
	c.cache.Clear()
}

// IsCacheExpired checks if the cache has expired
func (c *Client) IsCacheExpired() bool {
	return c.cache.IsExpired()
}

// CacheTimeRemaining returns the duration until cache expiration
func (c *Client) CacheTimeRemaining() time.Duration {
	return c.cache.TimeUntilExpiration()
}

// SaveCacheToFile saves the current cache to a file
func (c *Client) SaveCacheToFile(filepath string) error {
	return c.cache.SaveToFile(filepath)
}

// LoadCacheFromFile loads cache from a file
// If the cache file exists and is valid, it will be loaded into memory
// Returns an error if the file doesn't exist, is invalid, or has expired
func (c *Client) LoadCacheFromFile(filepath string) error {
	return c.cache.LoadFromFile(filepath)
}
