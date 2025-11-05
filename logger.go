package openrouter

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Logger is the interface for request/response logging
type Logger interface {
	// LogRequest logs an outgoing HTTP request
	LogRequest(method, url string, headers http.Header, body []byte)

	// LogResponse logs an incoming HTTP response
	LogResponse(statusCode int, headers http.Header, body []byte, duration time.Duration)

	// LogError logs an error
	LogError(err error)
}

// NoopLogger is a logger that does nothing
type NoopLogger struct{}

// LogRequest does nothing
func (n *NoopLogger) LogRequest(method, url string, headers http.Header, body []byte) {}

// LogResponse does nothing
func (n *NoopLogger) LogResponse(statusCode int, headers http.Header, body []byte, duration time.Duration) {
}

// LogError does nothing
func (n *NoopLogger) LogError(err error) {}

// StandardLogger is a simple logger that logs to the standard logger.
//
// Body Truncation:
// By default, request/response bodies are truncated to 500 characters to prevent
// excessive log output. This is usually sufficient for debugging but may truncate
// large API payloads. To customize:
//   - Use NewStandardLoggerWithTruncate(true, true, 1000) for 1000 char limit
//   - Use NewStandardLoggerWithTruncate(true, true, 0) for unlimited logging
//   - Adjust TruncateLimit field directly after creation
//
// Example:
//
//	// Default 500 char truncation
//	logger := openrouter.NewStandardLogger(true, true)
//
//	// Custom 2000 char truncation for large payloads
//	logger := openrouter.NewStandardLoggerWithTruncate(true, true, 2000)
//
//	// Unlimited logging (not recommended for production)
//	logger := openrouter.NewStandardLoggerWithTruncate(true, true, 0)
type StandardLogger struct {
	logger *log.Logger
	// LogBodies determines whether request/response bodies are logged
	LogBodies bool
	// LogHeaders determines whether headers are logged
	LogHeaders bool
	// TruncateLimit is the maximum number of characters to log for request/response bodies.
	// Bodies longer than this will be truncated to prevent excessive log output.
	// Default: 500 characters. Set to 0 for unlimited logging (not recommended for production).
	TruncateLimit int
}

// NewStandardLogger creates a new standard logger writing to stderr with default truncation (500 chars)
func NewStandardLogger(logBodies, logHeaders bool) *StandardLogger {
	return &StandardLogger{
		logger:        log.New(os.Stderr, "[OpenRouter] ", log.LstdFlags),
		LogBodies:     logBodies,
		LogHeaders:    logHeaders,
		TruncateLimit: 500, // Default truncation limit
	}
}

// NewStandardLoggerWithTruncate creates a new standard logger with a custom truncation limit.
// Set truncateLimit to 0 for unlimited logging (no truncation).
//
// Since: v2.0
//
// See also: WithStandardLoggerTruncate for using this with the client, NewStandardLogger for default truncation
func NewStandardLoggerWithTruncate(logBodies, logHeaders bool, truncateLimit int) *StandardLogger {
	if truncateLimit < 0 {
		truncateLimit = 500 // Use default for negative values
	}
	return &StandardLogger{
		logger:        log.New(os.Stderr, "[OpenRouter] ", log.LstdFlags),
		LogBodies:     logBodies,
		LogHeaders:    logHeaders,
		TruncateLimit: truncateLimit,
	}
}

// LogRequest logs an outgoing HTTP request
func (s *StandardLogger) LogRequest(method, url string, headers http.Header, body []byte) {
	s.logger.Printf("→ %s %s", method, url)

	if s.LogHeaders && len(headers) > 0 {
		s.logger.Printf("  Headers: %v", headers)
	}

	if s.LogBodies && len(body) > 0 {
		// Truncate large bodies if limit is set
		bodyStr := string(body)
		if s.TruncateLimit > 0 && len(bodyStr) > s.TruncateLimit {
			bodyStr = bodyStr[:s.TruncateLimit] + "... (truncated)"
		}
		s.logger.Printf("  Body: %s", bodyStr)
	}
}

// LogResponse logs an incoming HTTP response
func (s *StandardLogger) LogResponse(statusCode int, headers http.Header, body []byte, duration time.Duration) {
	s.logger.Printf("← %d (%v)", statusCode, duration.Round(time.Millisecond))

	if s.LogHeaders && len(headers) > 0 {
		s.logger.Printf("  Headers: %v", headers)
	}

	if s.LogBodies && len(body) > 0 {
		// Truncate large bodies if limit is set
		bodyStr := string(body)
		if s.TruncateLimit > 0 && len(bodyStr) > s.TruncateLimit {
			bodyStr = bodyStr[:s.TruncateLimit] + "... (truncated)"
		}
		s.logger.Printf("  Body: %s", bodyStr)
	}
}

// LogError logs an error
func (s *StandardLogger) LogError(err error) {
	s.logger.Printf("✗ Error: %v", err)
}

// CustomLogger allows custom logging with a provided function
type CustomLogger struct {
	LogFn func(format string, args ...interface{})
}

// LogRequest logs using the custom function
func (c *CustomLogger) LogRequest(method, url string, headers http.Header, body []byte) {
	c.LogFn("Request: %s %s", method, url)
}

// LogResponse logs using the custom function
func (c *CustomLogger) LogResponse(statusCode int, headers http.Header, body []byte, duration time.Duration) {
	c.LogFn("Response: %d (%v)", statusCode, duration)
}

// LogError logs using the custom function
func (c *CustomLogger) LogError(err error) {
	c.LogFn("Error: %v", err)
}

// WithLogger is a client option to enable request/response logging
func WithLogger(logger Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithStandardLogger is a convenience option to enable standard logging with default truncation (500 chars)
func WithStandardLogger(logBodies, logHeaders bool) ClientOption {
	return WithLogger(NewStandardLogger(logBodies, logHeaders))
}

// WithStandardLoggerTruncate is a convenience option to enable standard logging with custom truncation limit.
// Set truncateLimit to 0 for unlimited logging (no truncation).
//
// Since: v2.0
//
// See also: NewStandardLoggerWithTruncate, WithStandardLogger for default truncation
func WithStandardLoggerTruncate(logBodies, logHeaders bool, truncateLimit int) ClientOption {
	return WithLogger(NewStandardLoggerWithTruncate(logBodies, logHeaders, truncateLimit))
}

// WithCustomLogger is a convenience option for custom logging functions
func WithCustomLogger(logFn func(format string, args ...interface{})) ClientOption {
	return WithLogger(&CustomLogger{LogFn: logFn})
}

// logRequest is an internal helper to log requests if a logger is configured
func (c *Client) logRequest(method, url string, headers http.Header, body []byte) {
	if c.logger != nil {
		// Redact Authorization header for security
		redactedHeaders := make(http.Header)
		for k, v := range headers {
			if k == "Authorization" {
				redactedHeaders[k] = []string{"[REDACTED]"}
			} else {
				redactedHeaders[k] = v
			}
		}
		c.logger.LogRequest(method, url, redactedHeaders, body)
	}
}

// logResponse is an internal helper to log responses if a logger is configured
func (c *Client) logResponse(statusCode int, headers http.Header, body []byte, duration time.Duration) {
	if c.logger != nil {
		c.logger.LogResponse(statusCode, headers, body, duration)
	}
}

// logError is an internal helper to log errors if a logger is configured
func (c *Client) logError(err error) {
	if c.logger != nil {
		c.logger.LogError(err)
	}
}

// prettyJSON formats JSON for better logging (optional, for debugging)
func prettyJSON(data []byte) string {
	if len(data) > 1000 {
		return fmt.Sprintf("%s... (%d bytes)", string(data[:1000]), len(data))
	}
	return string(data)
}
