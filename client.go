package openrouter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
	baseURL    string
	httpClient *http.Client
	cache      *Cache
	apiKey     string // Optional API key for authenticated requests
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
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new OpenRouter API client
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
func (c *Client) GetModels() ([]Model, error) {
	// Check cache first
	if cached, ok := c.cache.Get(); ok {
		return cached.Models, nil
	}

	// Fetch from API
	models, err := c.fetchModels()
	if err != nil {
		return nil, err
	}

	// Update cache
	c.cache.Set(models)

	return models, nil
}

// GetModelsFresh forces a fresh fetch from the API, bypassing cache
func (c *Client) GetModelsFresh() ([]Model, error) {
	models, err := c.fetchModels()
	if err != nil {
		return nil, err
	}

	// Update cache with fresh data
	c.cache.Set(models)

	return models, nil
}

// fetchModels performs the actual HTTP request to fetch models
func (c *Client) fetchModels() ([]Model, error) {
	url := fmt.Sprintf("%s/models", c.baseURL)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key if provided
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var modelsResp ModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
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
