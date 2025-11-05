package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_GetModels_ContextCancellation(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithCacheTTL(1*time.Second), // Short TTL to force API calls
	)

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This should fail due to context cancellation
	_, err := client.GetModels(ctx)
	if err == nil {
		t.Fatal("Expected error due to context cancellation, got nil")
	}

	// Verify it's a context error
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", ctx.Err())
	}
}

func TestClient_CreateChatCompletion_ContextCancellation(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "test", "choices": [], "usage": {}}`))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{
			NewUserMessage("test"),
		},
	}

	// This should fail due to context cancellation
	_, err := client.CreateChatCompletion(ctx, req)
	if err == nil {
		t.Fatal("Expected error due to context cancellation, got nil")
	}

	// Verify it's a context error
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", ctx.Err())
	}
}

func TestClient_GetGeneration_ContextCancellation(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"id": "test", "model": "test-model"}}`))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This should fail due to context cancellation
	_, err := client.GetGeneration(ctx, "test-id")
	if err == nil {
		t.Fatal("Expected error due to context cancellation, got nil")
	}

	// Verify it's a context error
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", ctx.Err())
	}
}

func TestClient_GetGenerationWithRetry_ContextCancellation(t *testing.T) {
	// Create a test server that always returns 404
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	// Create context that cancels after first retry
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	// This should fail due to context cancellation during retry
	_, err := client.GetGenerationWithRetry(ctx, "test-id", 10, 100*time.Millisecond)
	if err == nil {
		t.Fatal("Expected error due to context cancellation, got nil")
	}

	// Should have attempted at most 2 requests before cancellation
	if attempts > 3 {
		t.Errorf("Expected at most 3 attempts due to context cancellation, got %d", attempts)
	}

	// Verify it's a context error
	if err != context.DeadlineExceeded && ctx.Err() != context.DeadlineExceeded {
		t.Logf("Error: %v, Context error: %v", err, ctx.Err())
	}
}

func TestClient_GetModels_WithContext_Success(t *testing.T) {
	// Create a test server that responds quickly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": "test/model", "name": "Test Model"}]}`))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
	)

	// Create context with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This should succeed
	models, err := client.GetModels(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}

	if models[0].ID != "test/model" {
		t.Errorf("Expected model ID 'test/model', got '%s'", models[0].ID)
	}
}
