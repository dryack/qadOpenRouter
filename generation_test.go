package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetGeneration_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Verify query parameter
		genID := r.URL.Query().Get("id")
		if genID != "test-gen-123" {
			t.Errorf("Expected generation ID test-gen-123, got %s", genID)
		}

		// Send mock response
		response := GenerationResponse{
			Data: GenerationStats{
				ID:                    "test-gen-123",
				Model:                 "test-model",
				ProviderName:          "test-provider",
				TokensPrompt:          100,
				TokensCompletion:      50,
				NativeCostsPrompt:     0.001,
				NativeCostsCompletion: 0.002,
				TotalCost:             0.003,
				CreatedAt:             "2024-01-01T00:00:00Z",
				FinishReason:          "stop",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	ctx := context.Background()

	// Test GetGeneration
	stats, err := client.GetGeneration(ctx, "test-gen-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.ID != "test-gen-123" {
		t.Errorf("Expected ID test-gen-123, got %s", stats.ID)
	}

	if stats.Model != "test-model" {
		t.Errorf("Expected model test-model, got %s", stats.Model)
	}

	if stats.ProviderName != "test-provider" {
		t.Errorf("Expected provider test-provider, got %s", stats.ProviderName)
	}

	if stats.TokensPrompt != 100 {
		t.Errorf("Expected 100 prompt tokens, got %d", stats.TokensPrompt)
	}

	if stats.TokensCompletion != 50 {
		t.Errorf("Expected 50 completion tokens, got %d", stats.TokensCompletion)
	}

	if stats.NativeCostsPrompt != 0.001 {
		t.Errorf("Expected prompt cost 0.001, got %f", stats.NativeCostsPrompt)
	}

	if stats.NativeCostsCompletion != 0.002 {
		t.Errorf("Expected completion cost 0.002, got %f", stats.NativeCostsCompletion)
	}

	if stats.TotalCost != 0.003 {
		t.Errorf("Expected total cost 0.003, got %f", stats.TotalCost)
	}

	if stats.FinishReason != "stop" {
		t.Errorf("Expected finish reason 'stop', got %s", stats.FinishReason)
	}
}

func TestGetGeneration_MissingID(t *testing.T) {
	client := NewClient(WithAPIKey("test-key"))
	ctx := context.Background()

	_, err := client.GetGeneration(ctx, "")
	if err == nil {
		t.Fatal("Expected error for missing generation ID")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != "generationID" {
		t.Errorf("Expected ValidationError for field 'generationID', got %v", err)
	}
}

func TestGetGeneration_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	ctx := context.Background()

	_, err := client.GetGeneration(ctx, "test-gen-123")
	if err == nil {
		t.Fatal("Expected error for HTTP 500")
	}
}

func TestGetGeneration_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Generation not found"))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	ctx := context.Background()

	_, err := client.GetGeneration(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("Expected error for 404 Not Found")
	}
}

func TestGetGeneration_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	ctx := context.Background()

	_, err := client.GetGeneration(ctx, "test-gen-123")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestGetGenerationCost_Success(t *testing.T) {
	expectedCost := 0.00567

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GenerationResponse{
			Data: GenerationStats{
				ID:                    "test-gen-123",
				Model:                 "test-model",
				ProviderName:          "test-provider",
				TokensPrompt:          100,
				TokensCompletion:      50,
				NativeCostsPrompt:     0.00267,
				NativeCostsCompletion: 0.00300,
				TotalCost:             expectedCost,
				CreatedAt:             "2024-01-01T00:00:00Z",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	ctx := context.Background()

	// Test GetGenerationCost
	cost, err := client.GetGenerationCost(ctx, "test-gen-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cost != expectedCost {
		t.Errorf("Expected cost %f, got %f", expectedCost, cost)
	}
}

func TestGetGenerationCost_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	ctx := context.Background()

	cost, err := client.GetGenerationCost(ctx, "test-gen-123")
	if err == nil {
		t.Fatal("Expected error for HTTP 500")
	}

	if cost != 0 {
		t.Errorf("Expected cost 0 on error, got %f", cost)
	}
}

func TestGetGeneration_WithOptionalFields(t *testing.T) {
	streamingLatency := 100
	totalLatency := 500
	appID := 42

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GenerationResponse{
			Data: GenerationStats{
				ID:                    "test-gen-123",
				Model:                 "test-model",
				StreamingLatency:      &streamingLatency,
				TotalLatency:          &totalLatency,
				ProviderName:          "test-provider",
				TokensPrompt:          100,
				TokensCompletion:      50,
				NativeCostsPrompt:     0.001,
				NativeCostsCompletion: 0.002,
				TotalCost:             0.003,
				CreatedAt:             "2024-01-01T00:00:00Z",
				AppID:                 &appID,
				Usage:                 0.5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	ctx := context.Background()

	// Test GetGeneration with optional fields
	stats, err := client.GetGeneration(ctx, "test-gen-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.StreamingLatency == nil || *stats.StreamingLatency != streamingLatency {
		t.Errorf("Expected streaming latency %d, got %v", streamingLatency, stats.StreamingLatency)
	}

	if stats.TotalLatency == nil || *stats.TotalLatency != totalLatency {
		t.Errorf("Expected total latency %d, got %v", totalLatency, stats.TotalLatency)
	}

	if stats.AppID == nil || *stats.AppID != appID {
		t.Errorf("Expected app ID %d, got %v", appID, stats.AppID)
	}

	if stats.Usage != 0.5 {
		t.Errorf("Expected usage 0.5, got %f", stats.Usage)
	}
}
