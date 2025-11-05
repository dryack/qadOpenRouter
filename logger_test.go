package openrouter

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNoopLogger(t *testing.T) {
	logger := &NoopLogger{}

	// These should not panic
	logger.LogRequest("GET", "https://example.com", nil, nil)
	logger.LogResponse(200, nil, nil, time.Second)
	logger.LogError(errors.New("test"))
}

func TestStandardLogger_LogRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStandardLogger(true, true)
	logger.logger.SetOutput(&buf)

	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{"test-agent"},
	}
	body := []byte(`{"test": "data"}`)

	logger.LogRequest("POST", "https://api.example.com/test", headers, body)

	output := buf.String()

	if !strings.Contains(output, "POST") {
		t.Error("Expected output to contain method POST")
	}

	if !strings.Contains(output, "https://api.example.com/test") {
		t.Error("Expected output to contain URL")
	}

	if !strings.Contains(output, "Headers:") {
		t.Error("Expected output to contain headers")
	}

	if !strings.Contains(output, "Body:") {
		t.Error("Expected output to contain body")
	}

	if !strings.Contains(output, `{"test": "data"}`) {
		t.Error("Expected output to contain request body")
	}
}

func TestStandardLogger_LogRequest_NoHeaders(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStandardLogger(true, false) // LogHeaders = false
	logger.logger.SetOutput(&buf)

	headers := http.Header{
		"Content-Type": []string{"application/json"},
	}

	logger.LogRequest("GET", "https://api.example.com/test", headers, nil)

	output := buf.String()

	if strings.Contains(output, "Headers:") {
		t.Error("Expected output to NOT contain headers when LogHeaders is false")
	}
}

func TestStandardLogger_LogRequest_NoBody(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStandardLogger(false, true) // LogBodies = false
	logger.logger.SetOutput(&buf)

	body := []byte(`{"test": "data"}`)

	logger.LogRequest("POST", "https://api.example.com/test", nil, body)

	output := buf.String()

	if strings.Contains(output, "Body:") {
		t.Error("Expected output to NOT contain body when LogBodies is false")
	}
}

func TestStandardLogger_LogRequest_TruncateLargeBody(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStandardLogger(true, false)
	logger.logger.SetOutput(&buf)

	// Create a body larger than 500 bytes
	largeBody := bytes.Repeat([]byte("x"), 600)

	logger.LogRequest("POST", "https://api.example.com/test", nil, largeBody)

	output := buf.String()

	if !strings.Contains(output, "... (truncated)") {
		t.Error("Expected output to indicate truncation for large body")
	}

	// Should contain first 500 characters
	if !strings.Contains(output, string(largeBody[:100])) {
		t.Error("Expected output to contain beginning of body")
	}
}

func TestStandardLogger_LogResponse(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStandardLogger(true, true)
	logger.logger.SetOutput(&buf)

	headers := http.Header{
		"Content-Type": []string{"application/json"},
	}
	body := []byte(`{"result": "success"}`)
	duration := 123 * time.Millisecond

	logger.LogResponse(200, headers, body, duration)

	output := buf.String()

	if !strings.Contains(output, "200") {
		t.Error("Expected output to contain status code 200")
	}

	if !strings.Contains(output, "123ms") {
		t.Error("Expected output to contain duration")
	}

	if !strings.Contains(output, "Headers:") {
		t.Error("Expected output to contain headers")
	}

	if !strings.Contains(output, "Body:") {
		t.Error("Expected output to contain body")
	}
}

func TestStandardLogger_LogError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStandardLogger(false, false)
	logger.logger.SetOutput(&buf)

	testErr := errors.New("test error message")
	logger.LogError(testErr)

	output := buf.String()

	if !strings.Contains(output, "Error:") {
		t.Error("Expected output to contain 'Error:'")
	}

	if !strings.Contains(output, "test error message") {
		t.Error("Expected output to contain error message")
	}
}

func TestCustomLogger(t *testing.T) {
	var messages []string
	logger := &CustomLogger{
		LogFn: func(format string, args ...interface{}) {
			messages = append(messages, format)
		},
	}

	logger.LogRequest("GET", "https://example.com", nil, nil)
	logger.LogResponse(200, nil, nil, time.Second)
	logger.LogError(errors.New("test"))

	if len(messages) != 3 {
		t.Errorf("Expected 3 log messages, got %d", len(messages))
	}

	if !strings.Contains(messages[0], "Request:") {
		t.Error("Expected first message to contain 'Request:'")
	}

	if !strings.Contains(messages[1], "Response:") {
		t.Error("Expected second message to contain 'Response:'")
	}

	if !strings.Contains(messages[2], "Error:") {
		t.Error("Expected third message to contain 'Error:'")
	}
}

func TestClient_logRequest_RedactsAuthHeader(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStandardLogger(false, true) // Only log headers
	logger.logger.SetOutput(&buf)

	client := NewClient(
		WithAPIKey("secret-api-key"),
		WithLogger(logger),
	)

	headers := http.Header{
		"Authorization": []string{"Bearer secret-api-key"},
		"Content-Type":  []string{"application/json"},
	}

	client.logRequest("POST", "https://api.example.com/test", headers, nil)

	output := buf.String()

	if strings.Contains(output, "secret-api-key") {
		t.Error("Expected Authorization header to be redacted")
	}

	if !strings.Contains(output, "[REDACTED]") {
		t.Error("Expected output to contain [REDACTED] for Authorization header")
	}

	if !strings.Contains(output, "application/json") {
		t.Error("Expected other headers to not be redacted")
	}
}

func TestClient_logRequest_NoLogger(t *testing.T) {
	client := NewClient(WithAPIKey("test-key"))

	// Should not panic when logger is nil
	client.logRequest("GET", "https://example.com", nil, nil)
	client.logResponse(200, nil, nil, time.Second)
	client.logError(errors.New("test"))
}

func TestWithStandardLogger(t *testing.T) {
	client := NewClient(WithStandardLogger(true, true))

	if client.logger == nil {
		t.Error("Expected logger to be set")
	}

	stdLogger, ok := client.logger.(*StandardLogger)
	if !ok {
		t.Fatal("Expected StandardLogger")
	}

	if !stdLogger.LogBodies {
		t.Error("Expected LogBodies to be true")
	}

	if !stdLogger.LogHeaders {
		t.Error("Expected LogHeaders to be true")
	}
}

func TestWithCustomLogger(t *testing.T) {
	called := false
	logFn := func(format string, args ...interface{}) {
		called = true
	}

	client := NewClient(WithCustomLogger(logFn))

	if client.logger == nil {
		t.Error("Expected logger to be set")
	}

	customLogger, ok := client.logger.(*CustomLogger)
	if !ok {
		t.Fatal("Expected CustomLogger")
	}

	// Test that the custom function is called
	customLogger.LogRequest("GET", "https://example.com", nil, nil)

	if !called {
		t.Error("Expected custom log function to be called")
	}
}

func TestPrettyJSON(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "Short JSON",
			data: []byte(`{"test": "data"}`),
			want: `{"test": "data"}`,
		},
		{
			name: "Long JSON",
			data: bytes.Repeat([]byte("x"), 1500),
			want: string(bytes.Repeat([]byte("x"), 1000)) + "... (1500 bytes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prettyJSON(tt.data)
			if got != tt.want {
				t.Errorf("prettyJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
