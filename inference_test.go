package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateChatCompletion_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header with test-key")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}

		// Send mock response
		response := ChatCompletionResponse{
			ID:      "test-id-123",
			Model:   "test-model",
			Created: 1234567890,
			Object:  "chat.completion",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: "Test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
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
	req := ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{
			NewUserMessage("Test prompt"),
		},
	}

	// Test chat completion
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.ID != "test-id-123" {
		t.Errorf("Expected ID test-id-123, got %s", resp.ID)
	}

	if resp.Model != "test-model" {
		t.Errorf("Expected model test-model, got %s", resp.Model)
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Test response" {
		t.Errorf("Expected content 'Test response', got %v", resp.Choices[0].Message.Content)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("Expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}

	if resp.Usage.CompletionTokens != 20 {
		t.Errorf("Expected 20 completion tokens, got %d", resp.Usage.CompletionTokens)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("Expected 30 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestCreateChatCompletion_MissingModel(t *testing.T) {
	client := NewClient(WithAPIKey("test-key"))
	ctx := context.Background()

	req := ChatCompletionRequest{
		Messages: []Message{
			NewUserMessage("Test prompt"),
		},
	}

	_, err := client.CreateChatCompletion(ctx, req)
	if err == nil {
		t.Fatal("Expected error for missing model")
	}

	if err.Error() != "model is required" {
		t.Errorf("Expected 'model is required' error, got %v", err)
	}
}

func TestCreateChatCompletion_MissingMessages(t *testing.T) {
	client := NewClient(WithAPIKey("test-key"))
	ctx := context.Background()

	req := ChatCompletionRequest{
		Model:    "test-model",
		Messages: []Message{},
	}

	_, err := client.CreateChatCompletion(ctx, req)
	if err == nil {
		t.Fatal("Expected error for missing messages")
	}

	if err.Error() != "at least one message is required" {
		t.Errorf("Expected 'at least one message is required' error, got %v", err)
	}
}

func TestCreateChatCompletion_MissingAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		// No API key
	)

	ctx := context.Background()
	req := ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{
			NewUserMessage("Test prompt"),
		},
	}

	_, err := client.CreateChatCompletion(ctx, req)
	if err == nil {
		t.Fatal("Expected error for missing API key")
	}

	if err.Error() != "API key is required for chat completions" {
		t.Errorf("Expected 'API key is required' error, got %v", err)
	}
}

func TestCreateChatCompletion_HTTPError(t *testing.T) {
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
	req := ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{
			NewUserMessage("Test prompt"),
		},
	}

	_, err := client.CreateChatCompletion(ctx, req)
	if err == nil {
		t.Fatal("Expected error for HTTP 500")
	}
}

func TestCreateChatCompletion_InvalidJSON(t *testing.T) {
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
	req := ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{
			NewUserMessage("Test prompt"),
		},
	}

	_, err := client.CreateChatCompletion(ctx, req)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestNewMessageHelpers(t *testing.T) {
	// Test NewUserMessage
	userMsg := NewUserMessage("Hello")
	if userMsg.Role != "user" {
		t.Errorf("Expected role 'user', got %s", userMsg.Role)
	}
	if userMsg.Content != "Hello" {
		t.Errorf("Expected content 'Hello', got %v", userMsg.Content)
	}

	// Test NewAssistantMessage
	assistantMsg := NewAssistantMessage("Hi there")
	if assistantMsg.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got %s", assistantMsg.Role)
	}
	if assistantMsg.Content != "Hi there" {
		t.Errorf("Expected content 'Hi there', got %v", assistantMsg.Content)
	}

	// Test NewSystemMessage
	systemMsg := NewSystemMessage("System prompt")
	if systemMsg.Role != "system" {
		t.Errorf("Expected role 'system', got %s", systemMsg.Role)
	}
	if systemMsg.Content != "System prompt" {
		t.Errorf("Expected content 'System prompt', got %v", systemMsg.Content)
	}

	// Test NewMessage
	customMsg := NewMessage("custom-role", "Custom content")
	if customMsg.Role != "custom-role" {
		t.Errorf("Expected role 'custom-role', got %s", customMsg.Role)
	}
	if customMsg.Content != "Custom content" {
		t.Errorf("Expected content 'Custom content', got %v", customMsg.Content)
	}
}

func TestCreateChatCompletion_WithParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request to verify parameters
		var req ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Temperature == nil || *req.Temperature != 0.7 {
			t.Errorf("Expected temperature 0.7")
		}

		if req.MaxTokens == nil || *req.MaxTokens != 100 {
			t.Errorf("Expected max_tokens 100")
		}

		// Send mock response
		response := ChatCompletionResponse{
			ID:      "test-id",
			Model:   "test-model",
			Choices: []Choice{{Index: 0, Message: Message{Role: "assistant", Content: "Response"}}},
			Usage:   Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithAPIKey("test-key"),
	)

	ctx := context.Background()
	temp := 0.7
	maxTokens := 100

	req := ChatCompletionRequest{
		Model:       "test-model",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		Messages: []Message{
			NewUserMessage("Test prompt"),
		},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.ID != "test-id" {
		t.Errorf("Expected ID test-id, got %s", resp.ID)
	}
}
