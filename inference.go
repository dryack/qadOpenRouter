package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ChatCompletionRequest represents a request to the /chat/completions endpoint
type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []Message              `json:"messages"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	TopK             *int                   `json:"top_k,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	RepetitionPenalty *float64              `json:"repetition_penalty,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
	Tools            []Tool                 `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"`
	Seed             *int                   `json:"seed,omitempty"`

	// OpenRouter-specific parameters
	Transforms       []string               `json:"transforms,omitempty"`
	Models           []string               `json:"models,omitempty"`
	Route            string                 `json:"route,omitempty"`
	Provider         *ProviderPreferences   `json:"provider,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string      `json:"role"`    // system, user, assistant, tool
	Content interface{} `json:"content"` // string or array of content parts
	Name    string      `json:"name,omitempty"`
	ToolCallID string   `json:"tool_call_id,omitempty"`
}

// ResponseFormat specifies the output format
type ResponseFormat struct {
	Type string `json:"type"` // text or json_object
}

// Tool represents a function tool
type Tool struct {
	Type     string       `json:"type"` // function
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function tool
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ProviderPreferences for OpenRouter-specific routing
type ProviderPreferences struct {
	AllowFallbacks bool     `json:"allow_fallbacks,omitempty"`
	RequireParameters bool  `json:"require_parameters,omitempty"`
	DataCollection string   `json:"data_collection,omitempty"` // allow or deny
	Order          []string `json:"order,omitempty"`
}

// ChatCompletionResponse represents the response from chat completions
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Created int64    `json:"created"`
	Object  string   `json:"object"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`

	// OpenRouter-specific fields
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	FinishReason string       `json:"finish_reason"` // stop, length, content_filter, tool_calls
	LogProbs     interface{}  `json:"logprobs,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CreateChatCompletion sends a chat completion request with automatic retry on transient errors
func (c *Client) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if req.Model == "" {
		return nil, &ValidationError{
			Field:   "model",
			Message: "model is required",
		}
	}

	if len(req.Messages) == 0 {
		return nil, &ValidationError{
			Field:   "messages",
			Message: "at least one message is required",
		}
	}

	// Execute with retry
	var completion *ChatCompletionResponse
	err := retryWithBackoff(ctx, c.retryConfig, func() error {
		var execErr error
		completion, execErr = c.executeChatCompletion(ctx, req)
		return execErr
	})
	if err != nil {
		return nil, err
	}

	return completion, nil
}

// executeChatCompletion performs the actual chat completion request (internal, for retry)
func (c *Client) executeChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {

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

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add context to request
	httpReq = httpReq.WithContext(ctx)

	// Set headers
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	} else {
		return nil, &ValidationError{
			Field:   "apiKey",
			Message: "API key is required for chat completions",
		}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Log request
	c.logRequest(httpReq.Method, httpReq.URL.String(), httpReq.Header, jsonData)

	// Execute request and track duration
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
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
			Message: "chat completion request failed",
			Err:     err,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logError(err)
		return nil, &NetworkError{
			Message: "failed to read response",
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

	var completion ChatCompletionResponse
	if err := json.Unmarshal(body, &completion); err != nil {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to unmarshal response: %v", err),
			Err:        ErrInvalidRequest,
		}
	}

	return &completion, nil
}

// NewMessage creates a new message with the specified role and content
func NewMessage(role, content string) Message {
	return Message{
		Role:    role,
		Content: content,
	}
}

// NewUserMessage creates a user message
func NewUserMessage(content string) Message {
	return NewMessage("user", content)
}

// NewAssistantMessage creates an assistant message
func NewAssistantMessage(content string) Message {
	return NewMessage("assistant", content)
}

// NewSystemMessage creates a system message
func NewSystemMessage(content string) Message {
	return NewMessage("system", content)
}
