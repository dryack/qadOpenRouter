package openrouter

import "time"

// ModelsResponse represents the response from the /api/v1/models endpoint
type ModelsResponse struct {
	Data []Model `json:"data"`
}

// Model represents a single model from OpenRouter
type Model struct {
	ID              string          `json:"id"`
	CanonicalSlug   string          `json:"canonical_slug"`
	HuggingFaceID   string          `json:"hugging_face_id"`
	Name            string          `json:"name"`
	Created         int64           `json:"created"`
	Description     string          `json:"description"`
	ContextLength   int             `json:"context_length"`
	Architecture    Architecture    `json:"architecture"`
	Pricing         Pricing         `json:"pricing"`
	TopProvider     TopProvider     `json:"top_provider"`
	PerRequestLimits interface{}    `json:"per_request_limits"`
	SupportedParams []string        `json:"supported_parameters"`
	DefaultParams   DefaultParams   `json:"default_parameters"`
}

// Architecture represents the model's architecture details
type Architecture struct {
	Modality        string   `json:"modality"`
	InputModalities []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Tokenizer       string   `json:"tokenizer"`
	InstructType    *string  `json:"instruct_type"`
}

// Pricing represents the pricing information for a model
// All prices are in USD per token (prompt/completion) or per request/image
type Pricing struct {
	Prompt            string `json:"prompt"`              // USD per prompt token
	Completion        string `json:"completion"`          // USD per completion token
	Request           string `json:"request"`             // USD per request
	Image             string `json:"image"`               // USD per image
	WebSearch         string `json:"web_search"`          // USD per web search
	InternalReasoning string `json:"internal_reasoning"`  // USD per internal reasoning token
	InputCacheRead    string `json:"input_cache_read"`    // USD per cached input token read
}

// TopProvider represents information about the top provider for this model
type TopProvider struct {
	ContextLength      int  `json:"context_length"`
	MaxCompletionTokens int `json:"max_completion_tokens"`
	IsModerated        bool `json:"is_moderated"`
}

// DefaultParams represents default parameters for model inference
type DefaultParams struct {
	Temperature      *float64 `json:"temperature"`
	TopP             *float64 `json:"top_p"`
	FrequencyPenalty *float64 `json:"frequency_penalty"`
}

// CachedModels wraps the models response with cache metadata
type CachedModels struct {
	Models    []Model
	FetchedAt time.Time
	ExpiresAt time.Time
}
