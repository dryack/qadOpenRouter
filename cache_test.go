package openrouter

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	models := []Model{
		{ID: "test/model-1", Name: "Test Model 1"},
		{ID: "test/model-2", Name: "Test Model 2"},
	}

	cache.Set(models)

	cached, ok := cache.Get()
	if !ok {
		t.Fatal("Expected to get cached data")
	}

	if len(cached.Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(cached.Models))
	}

	if cached.Models[0].ID != "test/model-1" {
		t.Errorf("Expected first model ID to be 'test/model-1', got %s", cached.Models[0].ID)
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := NewCache(100 * time.Millisecond)

	models := []Model{
		{ID: "test/model-1", Name: "Test Model 1"},
	}

	cache.Set(models)

	// Should be valid immediately
	if cache.IsExpired() {
		t.Error("Cache should not be expired immediately after setting")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	if !cache.IsExpired() {
		t.Error("Cache should be expired after TTL")
	}

	// Get should return false for expired cache
	_, ok := cache.Get()
	if ok {
		t.Error("Get should return false for expired cache")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	models := []Model{
		{ID: "test/model-1", Name: "Test Model 1"},
	}

	cache.Set(models)

	// Verify it's set
	if cache.IsExpired() {
		t.Error("Cache should not be expired")
	}

	// Clear the cache
	cache.Clear()

	// Should be expired after clear
	if !cache.IsExpired() {
		t.Error("Cache should be expired after clear")
	}

	// Get should return false
	_, ok := cache.Get()
	if ok {
		t.Error("Get should return false after clear")
	}
}

func TestCache_TimeUntilExpiration(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	models := []Model{
		{ID: "test/model-1", Name: "Test Model 1"},
	}

	cache.Set(models)

	remaining := cache.TimeUntilExpiration()

	// Should be close to 1 hour (with some margin for test execution time)
	if remaining < 59*time.Minute || remaining > 1*time.Hour {
		t.Errorf("Expected remaining time close to 1 hour, got %v", remaining)
	}
}

func TestCache_EmptyCache(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Empty cache should be expired
	if !cache.IsExpired() {
		t.Error("Empty cache should be expired")
	}

	// TimeUntilExpiration should return 0
	if cache.TimeUntilExpiration() != 0 {
		t.Error("Empty cache should have 0 time until expiration")
	}

	// Get should return false
	_, ok := cache.Get()
	if ok {
		t.Error("Get should return false for empty cache")
	}
}

func TestCache_SaveAndLoadFile(t *testing.T) {
	cache := NewCache(1 * time.Hour)
	tmpfile := t.TempDir() + "/cache.json"

	models := []Model{
		{ID: "test/model-1", Name: "Test Model 1"},
		{ID: "test/model-2", Name: "Test Model 2"},
	}

	// Set and save cache
	cache.Set(models)
	err := cache.SaveToFile(tmpfile)
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Create new cache and load from file
	cache2 := NewCache(1 * time.Hour)
	err = cache2.LoadFromFile(tmpfile)
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	// Verify loaded data
	loaded, ok := cache2.Get()
	if !ok {
		t.Fatal("Expected to get loaded cache data")
	}

	if len(loaded.Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(loaded.Models))
	}

	if loaded.Models[0].ID != "test/model-1" {
		t.Errorf("Expected first model ID to be 'test/model-1', got %s", loaded.Models[0].ID)
	}
}

func TestCache_SaveEmptyCache(t *testing.T) {
	cache := NewCache(1 * time.Hour)
	tmpfile := t.TempDir() + "/cache.json"

	// Try to save empty cache
	err := cache.SaveToFile(tmpfile)
	if err == nil {
		t.Error("Expected error when saving empty cache")
	}
}

func TestCache_LoadNonexistentFile(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Try to load from nonexistent file
	err := cache.LoadFromFile("/nonexistent/path/cache.json")
	if err == nil {
		t.Error("Expected error when loading from nonexistent file")
	}
}

func TestCache_LoadExpiredFile(t *testing.T) {
	cache := NewCache(100 * time.Millisecond)
	tmpfile := t.TempDir() + "/cache.json"

	models := []Model{
		{ID: "test/model-1", Name: "Test Model 1"},
	}

	// Set and save cache
	cache.Set(models)
	err := cache.SaveToFile(tmpfile)
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Try to load expired cache
	cache2 := NewCache(1 * time.Hour)
	err = cache2.LoadFromFile(tmpfile)
	if err == nil {
		t.Error("Expected error when loading expired cache")
	}
}

func TestValidateCachedModels_ValidData(t *testing.T) {
	now := time.Now()
	cached := &CachedModels{
		Models:    []Model{{ID: "test/model", Name: "Test"}},
		FetchedAt: now,
		ExpiresAt: now.Add(1 * time.Hour),
	}

	err := ValidateCachedModels(cached)
	if err != nil {
		t.Errorf("Expected no error for valid data, got %v", err)
	}
}

func TestValidateCachedModels_NilCache(t *testing.T) {
	err := ValidateCachedModels(nil)
	if err == nil {
		t.Error("Expected error for nil cache")
	}
	if !errors.Is(err, ErrNilCache) {
		t.Errorf("Expected ErrNilCache, got %v", err)
	}
}

func TestValidateCachedModels_NilModels(t *testing.T) {
	now := time.Now()
	cached := &CachedModels{
		Models:    nil,
		FetchedAt: now,
		ExpiresAt: now.Add(1 * time.Hour),
	}

	err := ValidateCachedModels(cached)
	if err == nil {
		t.Error("Expected error for nil models array")
	}
	if !errors.Is(err, ErrNilModels) {
		t.Errorf("Expected ErrNilModels, got %v", err)
	}
}

func TestValidateCachedModels_ZeroFetchedAt(t *testing.T) {
	now := time.Now()
	cached := &CachedModels{
		Models:    []Model{{ID: "test/model"}},
		FetchedAt: time.Time{}, // Zero value
		ExpiresAt: now.Add(1 * time.Hour),
	}

	err := ValidateCachedModels(cached)
	if err == nil {
		t.Error("Expected error for zero FetchedAt timestamp")
	}
	if !errors.Is(err, ErrZeroFetchedAt) {
		t.Errorf("Expected ErrZeroFetchedAt, got %v", err)
	}
}

func TestValidateCachedModels_ZeroExpiresAt(t *testing.T) {
	now := time.Now()
	cached := &CachedModels{
		Models:    []Model{{ID: "test/model"}},
		FetchedAt: now,
		ExpiresAt: time.Time{}, // Zero value
	}

	err := ValidateCachedModels(cached)
	if err == nil {
		t.Error("Expected error for zero ExpiresAt timestamp")
	}
	if !errors.Is(err, ErrZeroExpiresAt) {
		t.Errorf("Expected ErrZeroExpiresAt, got %v", err)
	}
}

func TestValidateCachedModels_ExpiresBeforeFetched(t *testing.T) {
	now := time.Now()
	cached := &CachedModels{
		Models:    []Model{{ID: "test/model"}},
		FetchedAt: now,
		ExpiresAt: now.Add(-1 * time.Hour), // Before FetchedAt
	}

	err := ValidateCachedModels(cached)
	if err == nil {
		t.Error("Expected error for ExpiresAt before FetchedAt")
	}
	if !errors.Is(err, ErrInvalidExpiration) {
		t.Errorf("Expected ErrInvalidExpiration, got %v", err)
	}
}

func TestCache_LoadInvalidJSON(t *testing.T) {
	cache := NewCache(1 * time.Hour)
	tmpfile := t.TempDir() + "/invalid.json"

	// Write invalid JSON
	err := os.WriteFile(tmpfile, []byte(`{"invalid": json}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err = cache.LoadFromFile(tmpfile)
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestCache_LoadCorruptedStructure(t *testing.T) {
	cache := NewCache(1 * time.Hour)
	tmpfile := t.TempDir() + "/corrupted.json"

	// Write JSON with missing required fields
	err := os.WriteFile(tmpfile, []byte(`{"Models": null, "FetchedAt": "2024-01-01T00:00:00Z", "ExpiresAt": "2024-01-02T00:00:00Z"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err = cache.LoadFromFile(tmpfile)
	if err == nil {
		t.Error("Expected error when loading corrupted cache structure")
	}
}

func TestNewCacheWithFileMode(t *testing.T) {
	cache := NewCacheWithFileMode(1*time.Hour, 0600)

	if cache.fileMode != 0600 {
		t.Errorf("Expected fileMode to be 0600, got %o", cache.fileMode)
	}

	if cache.ttl != 1*time.Hour {
		t.Errorf("Expected ttl to be 1 hour, got %v", cache.ttl)
	}
}

func TestNewCache_DefaultFileMode(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	if cache.fileMode != 0644 {
		t.Errorf("Expected default fileMode to be 0644, got %o", cache.fileMode)
	}
}

func TestCache_SaveWithCustomFileMode(t *testing.T) {
	cache := NewCacheWithFileMode(1*time.Hour, 0600)
	tmpfile := t.TempDir() + "/cache_0600.json"

	models := []Model{
		{ID: "test/model-1", Name: "Test Model 1"},
	}

	// Set and save cache
	cache.Set(models)
	err := cache.SaveToFile(tmpfile)
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(tmpfile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	fileMode := info.Mode().Perm()
	if fileMode != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", fileMode)
	}
}

func TestWithCacheFileMode(t *testing.T) {
	client := NewClient(
		WithAPIKey("test-key"),
		WithCacheFileMode(2*time.Hour, 0600),
	)

	if client.cache.fileMode != 0600 {
		t.Errorf("Expected cache fileMode to be 0600, got %o", client.cache.fileMode)
	}

	if client.cache.ttl != 2*time.Hour {
		t.Errorf("Expected cache ttl to be 2 hours, got %v", client.cache.ttl)
	}
}
