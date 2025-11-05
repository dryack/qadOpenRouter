package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestClient_RateLimit_GetModels(t *testing.T) {
	// Track request times
	var requestTimes []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	// Create client with rate limiter: 2 requests per second
	limiter := rate.NewLimiter(rate.Every(500*time.Millisecond), 1)
	client := NewClient(
		WithBaseURL(server.URL),
		WithRateLimiter(limiter),
		WithCacheTTL(1*time.Millisecond), // Short TTL to force API calls
	)

	// Make 3 requests
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		client.ClearCache() // Clear cache to force API call
		_, err := client.GetModels(ctx)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
	}

	// Verify rate limiting worked
	if len(requestTimes) != 3 {
		t.Fatalf("Expected 3 requests, got %d", len(requestTimes))
	}

	// Check that requests were spaced out by at least 500ms
	for i := 1; i < len(requestTimes); i++ {
		duration := requestTimes[i].Sub(requestTimes[i-1])
		if duration < 400*time.Millisecond { // Allow small margin
			t.Errorf("Requests %d and %d were too close: %v (expected >= 500ms)", i, i+1, duration)
		}
	}
}

func TestClient_RateLimit_CreateChatCompletion(t *testing.T) {
	// Track request count
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "test", "choices": [{"message": {"content": "test"}}], "usage": {"total_tokens": 10}}`))
	}))
	defer server.Close()

	// Create client with rate limiter: 5 requests per second
	limiter := rate.NewLimiter(rate.Every(200*time.Millisecond), 1)
	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
		WithRateLimiter(limiter),
	)

	// Make multiple rapid requests
	ctx := context.Background()
	start := time.Now()

	for i := 0; i < 5; i++ {
		req := ChatCompletionRequest{
			Model: "test-model",
			Messages: []Message{
				NewUserMessage("test"),
			},
		}
		_, err := client.CreateChatCompletion(ctx, req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
	}

	duration := time.Since(start)

	// Verify all requests completed
	if requestCount != 5 {
		t.Errorf("Expected 5 requests, got %d", requestCount)
	}

	// Verify rate limiting added delay (at least 800ms for 5 requests at 200ms intervals)
	minDuration := 800 * time.Millisecond
	if duration < minDuration {
		t.Errorf("Requests completed too quickly: %v (expected >= %v)", duration, minDuration)
	}
}

func TestClient_RateLimit_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	// Create client with very slow rate limiter
	limiter := rate.NewLimiter(rate.Every(10*time.Second), 1)
	client := NewClient(
		WithBaseURL(server.URL),
		WithRateLimiter(limiter),
		WithCacheTTL(1*time.Millisecond),
	)

	// Make first request (will succeed)
	ctx1 := context.Background()
	client.ClearCache() // Clear cache to force API call
	_, err := client.GetModels(ctx1)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	// Make second request with short timeout (should fail due to rate limit wait)
	ctx2, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client.ClearCache() // Clear cache to force API call
	_, err = client.GetModels(ctx2)
	if err == nil {
		t.Fatal("Expected error due to context cancellation during rate limit wait, got nil")
	}

	// Verify it's a context or rate limit error
	if ctx2.Err() != context.DeadlineExceeded {
		t.Logf("Error: %v, Context error: %v", err, ctx2.Err())
	}
}

func TestClient_NoRateLimit(t *testing.T) {
	// Track request times
	var requestTimes []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	// Create client WITHOUT rate limiter
	client := NewClient(
		WithBaseURL(server.URL),
		WithCacheTTL(1*time.Millisecond),
	)

	// Make 3 rapid requests
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		client.ClearCache() // Clear cache to force API call
		_, err := client.GetModels(ctx)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
	}

	// Verify no rate limiting (requests should be very close together)
	if len(requestTimes) != 3 {
		t.Fatalf("Expected 3 requests, got %d", len(requestTimes))
	}

	// Check that requests completed quickly (no artificial delays)
	totalDuration := requestTimes[2].Sub(requestTimes[0])
	if totalDuration > 100*time.Millisecond {
		t.Errorf("Requests took too long without rate limiting: %v (expected < 100ms)", totalDuration)
	}
}

func TestClient_RateLimit_BurstAllowance(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	// Create client with burst allowance of 3
	limiter := rate.NewLimiter(rate.Every(time.Second), 3)
	client := NewClient(
		WithBaseURL(server.URL),
		WithRateLimiter(limiter),
		WithCacheTTL(1*time.Millisecond),
	)

	// Make 3 rapid requests (should all succeed immediately due to burst)
	ctx := context.Background()
	start := time.Now()

	for i := 0; i < 3; i++ {
		client.ClearCache() // Clear cache to force API call
		_, err := client.GetModels(ctx)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
	}

	duration := time.Since(start)

	// Verify all requests completed quickly (burst should allow immediate completion)
	if duration > 200*time.Millisecond {
		t.Errorf("Burst requests took too long: %v (expected < 200ms)", duration)
	}

	// Verify all requests completed
	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
}
