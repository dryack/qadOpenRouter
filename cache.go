package openrouter

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Cache provides thread-safe caching for model data
type Cache struct {
	mu    sync.RWMutex
	data  *CachedModels
	ttl   time.Duration
}

// NewCache creates a new cache with the specified TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl: ttl,
	}
}

// Get retrieves cached models if they exist and haven't expired
func (c *Cache) Get() (*CachedModels, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return nil, false
	}

	if time.Now().After(c.data.ExpiresAt) {
		return nil, false
	}

	return c.data, true
}

// Set stores models in the cache with expiration time
func (c *Cache) Set(models []Model) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.data = &CachedModels{
		Models:    models,
		FetchedAt: now,
		ExpiresAt: now.Add(c.ttl),
	}
}

// Clear removes all cached data
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = nil
}

// IsExpired checks if the cache has expired
func (c *Cache) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return true
	}

	return time.Now().After(c.data.ExpiresAt)
}

// TimeUntilExpiration returns the duration until cache expiration
func (c *Cache) TimeUntilExpiration() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return 0
	}

	remaining := time.Until(c.data.ExpiresAt)
	if remaining < 0 {
		return 0
	}

	return remaining
}

// SaveToFile saves the current cache to a JSON file
// Returns nil if cache is empty or expired
func (c *Cache) SaveToFile(filepath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return fmt.Errorf("no cache data to save")
	}

	// Don't save expired cache
	if time.Now().After(c.data.ExpiresAt) {
		return fmt.Errorf("cache expired, not saving")
	}

	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// LoadFromFile loads cache from a JSON file
// Returns an error if the file doesn't exist, is invalid, or the cache has expired
func (c *Cache) LoadFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var cached CachedModels
	err = json.Unmarshal(data, &cached)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	// Check if loaded cache has expired
	if time.Now().After(cached.ExpiresAt) {
		return fmt.Errorf("loaded cache has expired")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = &cached

	return nil
}
