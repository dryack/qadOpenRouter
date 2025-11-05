package openrouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// Cache validation errors
var (
	// ErrNilCache indicates the cached models pointer is nil
	ErrNilCache = errors.New("cached models is nil")
	// ErrNilModels indicates the models array is nil
	ErrNilModels = errors.New("models array is nil")
	// ErrZeroFetchedAt indicates the FetchedAt timestamp is zero/unset
	ErrZeroFetchedAt = errors.New("FetchedAt timestamp is zero")
	// ErrZeroExpiresAt indicates the ExpiresAt timestamp is zero/unset
	ErrZeroExpiresAt = errors.New("ExpiresAt timestamp is zero")
	// ErrInvalidExpiration indicates ExpiresAt is before FetchedAt
	ErrInvalidExpiration = errors.New("ExpiresAt is before FetchedAt")
)

// Cache provides thread-safe caching for model data
type Cache struct {
	mu       sync.RWMutex
	data     *CachedModels
	ttl      time.Duration
	fileMode os.FileMode // File permissions for SaveToFile (default: 0644)
}

// NewCache creates a new cache with the specified TTL and default file permissions (0644)
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl:      ttl,
		fileMode: 0644, // Default file permissions
	}
}

// NewCacheWithFileMode creates a new cache with custom file permissions.
// Use this to specify custom file permissions when saving cache to disk.
//
// Since: v2.0
//
// Example: NewCacheWithFileMode(1*time.Hour, 0600) for user-only read/write
//
// See also: WithCacheFileMode for using this with the client
func NewCacheWithFileMode(ttl time.Duration, fileMode os.FileMode) *Cache {
	return &Cache{
		ttl:      ttl,
		fileMode: fileMode,
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

	err = os.WriteFile(filepath, data, c.fileMode)
	if err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// ValidateCachedModels validates the structure of cached models.
// This function can be used to validate cache data before loading it into the cache.
// Returns specific error types (ErrNilCache, ErrNilModels, etc.) for programmatic error handling.
//
// Since: v2.0
//
// See also: LoadFromFile which calls this internally, and custom error types
// ErrNilCache, ErrNilModels, ErrZeroFetchedAt, ErrZeroExpiresAt, ErrInvalidExpiration
func ValidateCachedModels(cached *CachedModels) error {
	if cached == nil {
		return ErrNilCache
	}
	if cached.Models == nil {
		return ErrNilModels
	}
	if cached.FetchedAt.IsZero() {
		return ErrZeroFetchedAt
	}
	if cached.ExpiresAt.IsZero() {
		return ErrZeroExpiresAt
	}
	if cached.ExpiresAt.Before(cached.FetchedAt) {
		return fmt.Errorf("%w: ExpiresAt (%v) is before FetchedAt (%v)",
			ErrInvalidExpiration, cached.ExpiresAt, cached.FetchedAt)
	}
	return nil
}

// LoadFromFile loads cache from a JSON file.
// Returns an error if the file doesn't exist, is invalid, or the cache has expired.
//
// The cache structure is validated using ValidateCachedModels before loading.
//
// See also: ValidateCachedModels, SaveToFile
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

	// Validate the cache structure
	if err := ValidateCachedModels(&cached); err != nil {
		return fmt.Errorf("invalid cache structure: %w", err)
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
