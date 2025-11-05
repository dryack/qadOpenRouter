package openrouter

import (
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
