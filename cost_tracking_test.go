package openrouter

import (
	"sync"
	"testing"
	"time"
)

func TestCostTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewCostTracker()

	// Add some test data
	for i := 0; i < 3; i++ {
		result := InferenceResult{
			Model:            "test-model",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			ActualCost:       0.001,
			Latency:          100 * time.Millisecond,
			Timestamp:        time.Now(),
		}
		tracker.Record(result)
	}

	// Test concurrent access to GetAllModelStats
	// This would deadlock with the old implementation
	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats := tracker.GetAllModelStats()
			if len(stats) != 1 {
				t.Errorf("Expected 1 model in stats, got %d", len(stats))
			}
		}()
	}

	// Wait with timeout to detect deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out - possible deadlock detected")
	}
}

func TestCostTracker_GetModelStats(t *testing.T) {
	tracker := NewCostTracker()

	// Record some results
	tracker.Record(InferenceResult{
		Model:            "model-a",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		ActualCost:       0.001,
		Latency:          100 * time.Millisecond,
		Timestamp:        time.Now(),
	})

	tracker.Record(InferenceResult{
		Model:            "model-a",
		PromptTokens:     200,
		CompletionTokens: 100,
		TotalTokens:      300,
		ActualCost:       0.002,
		Latency:          150 * time.Millisecond,
		Timestamp:        time.Now(),
	})

	// Get stats for model-a
	stats := tracker.GetModelStats("model-a")
	if stats == nil {
		t.Fatal("Expected stats for model-a, got nil")
	}

	if stats.RequestCount != 2 {
		t.Errorf("Expected 2 requests, got %d", stats.RequestCount)
	}

	if stats.SuccessCount != 2 {
		t.Errorf("Expected 2 successful requests, got %d", stats.SuccessCount)
	}

	if stats.TotalPromptTokens != 300 {
		t.Errorf("Expected 300 prompt tokens, got %d", stats.TotalPromptTokens)
	}

	if stats.TotalCompletionTokens != 150 {
		t.Errorf("Expected 150 completion tokens, got %d", stats.TotalCompletionTokens)
	}
}

func TestCostTracker_GetAllModelStats(t *testing.T) {
	tracker := NewCostTracker()

	// Record results for multiple models
	models := []string{"model-a", "model-b", "model-c"}
	for _, model := range models {
		tracker.Record(InferenceResult{
			Model:            model,
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			ActualCost:       0.001,
			Latency:          100 * time.Millisecond,
			Timestamp:        time.Now(),
		})
	}

	// Get all stats
	allStats := tracker.GetAllModelStats()
	if len(allStats) != 3 {
		t.Errorf("Expected stats for 3 models, got %d", len(allStats))
	}

	for _, model := range models {
		if _, ok := allStats[model]; !ok {
			t.Errorf("Missing stats for model: %s", model)
		}
	}
}

func TestCostTracker_RecordError(t *testing.T) {
	tracker := NewCostTracker()

	// Record a result with error
	tracker.Record(InferenceResult{
		Model:     "model-a",
		Error:     &testError{msg: "test error"},
		Timestamp: time.Now(),
	})

	stats := tracker.GetModelStats("model-a")
	if stats == nil {
		t.Fatal("Expected stats for model-a, got nil")
	}

	if stats.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", stats.ErrorCount)
	}

	if stats.SuccessCount != 0 {
		t.Errorf("Expected 0 successful requests, got %d", stats.SuccessCount)
	}
}

func TestCostTracker_Compare(t *testing.T) {
	tracker := NewCostTracker()

	// Record results for two models with different costs
	tracker.Record(InferenceResult{
		Model:            "cheap-model",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		ActualCost:       0.001,
		Latency:          200 * time.Millisecond,
		Timestamp:        time.Now(),
	})

	tracker.Record(InferenceResult{
		Model:            "expensive-model",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		ActualCost:       0.01,
		Latency:          100 * time.Millisecond,
		Timestamp:        time.Now(),
	})

	report := tracker.Compare()

	if report.BestCostModel != "cheap-model" {
		t.Errorf("Expected best cost model to be 'cheap-model', got '%s'", report.BestCostModel)
	}

	if report.BestLatencyModel != "expensive-model" {
		t.Errorf("Expected best latency model to be 'expensive-model', got '%s'", report.BestLatencyModel)
	}
}

func TestCostTracker_Clear(t *testing.T) {
	tracker := NewCostTracker()

	// Add data
	tracker.Record(InferenceResult{
		Model:      "model-a",
		ActualCost: 0.001,
		Timestamp:  time.Now(),
	})

	// Verify data exists
	if tracker.GetTotalCost() == 0 {
		t.Error("Expected non-zero total cost")
	}

	// Clear
	tracker.Clear()

	// Verify cleared
	if tracker.GetTotalCost() != 0 {
		t.Error("Expected zero total cost after clear")
	}

	allStats := tracker.GetAllModelStats()
	if len(allStats) != 0 {
		t.Errorf("Expected 0 models after clear, got %d", len(allStats))
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
