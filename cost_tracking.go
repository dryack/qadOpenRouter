package openrouter

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// InferenceResult captures the result of an inference for cost tracking
type InferenceResult struct {
	Model            string
	GenerationID     string
	Prompt           string
	Response         string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	EstimatedCost    float64
	ActualCost       float64
	Latency          time.Duration
	Timestamp        time.Time
	Error            error
}

// ModelStats aggregates statistics for a specific model
type ModelStats struct {
	Model             string
	RequestCount      int
	SuccessCount      int
	ErrorCount        int
	TotalPromptTokens int
	TotalCompletionTokens int
	TotalTokens       int
	TotalEstimatedCost float64
	TotalActualCost   float64
	AvgLatency        time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	Results           []InferenceResult
}

// CostTracker tracks inference costs for A/B testing
type CostTracker struct {
	mu      sync.RWMutex
	results map[string][]InferenceResult // map[model][]results
}

// NewCostTracker creates a new cost tracker
func NewCostTracker() *CostTracker {
	return &CostTracker{
		results: make(map[string][]InferenceResult),
	}
}

// Record adds an inference result to the tracker
func (ct *CostTracker) Record(result InferenceResult) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.results[result.Model] = append(ct.results[result.Model], result)
}

// GetModelStats returns aggregated statistics for a specific model
func (ct *CostTracker) GetModelStats(model string) *ModelStats {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	results, ok := ct.results[model]
	if !ok {
		return nil
	}

	stats := &ModelStats{
		Model:   model,
		Results: results,
	}

	var totalLatency time.Duration
	stats.MinLatency = time.Hour // Start with a high value

	for _, r := range results {
		stats.RequestCount++

		if r.Error != nil {
			stats.ErrorCount++
			continue
		}

		stats.SuccessCount++
		stats.TotalPromptTokens += r.PromptTokens
		stats.TotalCompletionTokens += r.CompletionTokens
		stats.TotalTokens += r.TotalTokens
		stats.TotalEstimatedCost += r.EstimatedCost
		stats.TotalActualCost += r.ActualCost

		totalLatency += r.Latency
		if r.Latency < stats.MinLatency {
			stats.MinLatency = r.Latency
		}
		if r.Latency > stats.MaxLatency {
			stats.MaxLatency = r.Latency
		}
	}

	if stats.SuccessCount > 0 {
		stats.AvgLatency = totalLatency / time.Duration(stats.SuccessCount)
	}

	return stats
}

// GetAllModelStats returns statistics for all tracked models
func (ct *CostTracker) GetAllModelStats() map[string]*ModelStats {
	// First, collect all model names under read lock
	ct.mu.RLock()
	models := make([]string, 0, len(ct.results))
	for model := range ct.results {
		models = append(models, model)
	}
	ct.mu.RUnlock()

	// Then call GetModelStats for each model (which acquires its own lock)
	allStats := make(map[string]*ModelStats)
	for _, model := range models {
		stats := ct.GetModelStats(model)
		if stats != nil {
			allStats[model] = stats
		}
	}

	return allStats
}

// ComparisonReport generates a comparison report between models
type ComparisonReport struct {
	Models           []string
	Stats            map[string]*ModelStats
	BestCostModel    string
	BestLatencyModel string
	GeneratedAt      time.Time
}

// Compare generates a comparison report for all tracked models
func (ct *CostTracker) Compare() *ComparisonReport {
	allStats := ct.GetAllModelStats()

	report := &ComparisonReport{
		Stats:       allStats,
		GeneratedAt: time.Now(),
	}

	var bestCost, bestLatency float64
	var models []string

	for model, stats := range allStats {
		models = append(models, model)

		if stats.SuccessCount == 0 {
			continue
		}

		avgCost := stats.TotalActualCost / float64(stats.SuccessCount)

		// Track best cost (lowest)
		if report.BestCostModel == "" || avgCost < bestCost {
			bestCost = avgCost
			report.BestCostModel = model
		}

		// Track best latency (lowest)
		if report.BestLatencyModel == "" || float64(stats.AvgLatency) < bestLatency {
			bestLatency = float64(stats.AvgLatency)
			report.BestLatencyModel = model
		}
	}

	sort.Strings(models)
	report.Models = models

	return report
}

// Clear removes all tracked results
func (ct *CostTracker) Clear() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.results = make(map[string][]InferenceResult)
}

// GetTotalCost returns the total cost across all models
func (ct *CostTracker) GetTotalCost() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var total float64
	for _, results := range ct.results {
		for _, r := range results {
			total += r.ActualCost
		}
	}

	return total
}

// FormatReport generates a human-readable report
func (report *ComparisonReport) FormatReport() string {
	output := fmt.Sprintf("=== A/B Testing Comparison Report ===\n")
	output += fmt.Sprintf("Generated: %s\n\n", report.GeneratedAt.Format(time.RFC3339))

	if len(report.Models) == 0 {
		return output + "No data available.\n"
	}

	output += fmt.Sprintf("Models Tested: %d\n", len(report.Models))
	output += fmt.Sprintf("Best Cost: %s\n", report.BestCostModel)
	output += fmt.Sprintf("Best Latency: %s\n\n", report.BestLatencyModel)

	// Sort models by cost for display
	type costModel struct {
		model   string
		avgCost float64
	}
	var costModels []costModel

	for model, stats := range report.Stats {
		if stats.SuccessCount > 0 {
			avgCost := stats.TotalActualCost / float64(stats.SuccessCount)
			costModels = append(costModels, costModel{model, avgCost})
		}
	}

	sort.Slice(costModels, func(i, j int) bool {
		return costModels[i].avgCost < costModels[j].avgCost
	})

	output += "=== Model Details (sorted by cost) ===\n\n"
	for _, cm := range costModels {
		stats := report.Stats[cm.model]
		output += fmt.Sprintf("Model: %s\n", cm.model)
		output += fmt.Sprintf("  Requests: %d (Success: %d, Errors: %d)\n",
			stats.RequestCount, stats.SuccessCount, stats.ErrorCount)
		output += fmt.Sprintf("  Tokens: %d prompt + %d completion = %d total\n",
			stats.TotalPromptTokens, stats.TotalCompletionTokens, stats.TotalTokens)
		output += fmt.Sprintf("  Total Cost: $%.6f (Est: $%.6f)\n",
			stats.TotalActualCost, stats.TotalEstimatedCost)

		if stats.SuccessCount > 0 {
			avgCost := stats.TotalActualCost / float64(stats.SuccessCount)
			output += fmt.Sprintf("  Avg Cost/Request: $%.6f\n", avgCost)
			output += fmt.Sprintf("  Avg Latency: %v (Min: %v, Max: %v)\n",
				stats.AvgLatency.Round(time.Millisecond),
				stats.MinLatency.Round(time.Millisecond),
				stats.MaxLatency.Round(time.Millisecond))
		}
		output += "\n"
	}

	return output
}

// AvgCostPerRequest calculates average cost per successful request for a model
func (stats *ModelStats) AvgCostPerRequest() float64 {
	if stats.SuccessCount == 0 {
		return 0
	}
	return stats.TotalActualCost / float64(stats.SuccessCount)
}

// SuccessRate returns the success rate as a percentage
func (stats *ModelStats) SuccessRate() float64 {
	if stats.RequestCount == 0 {
		return 0
	}
	return (float64(stats.SuccessCount) / float64(stats.RequestCount)) * 100
}
