package eval

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// PerformanceConfig holds optional thresholds for the PerformanceEvaluator.
type PerformanceConfig struct {
	// MaxLatencyMs, if > 0, causes runs exceeding this p99 to be flagged as failures.
	MaxLatencyMs float64
}

// PerformanceEvaluator calculates latency percentiles and token statistics.
type PerformanceEvaluator struct {
	cfg PerformanceConfig
}

// NewPerformanceEvaluator constructs a PerformanceEvaluator.
func NewPerformanceEvaluator(cfg PerformanceConfig) *PerformanceEvaluator {
	return &PerformanceEvaluator{cfg: cfg}
}

// Name implements Evaluator.
func (e *PerformanceEvaluator) Name() string { return "performance" }

// Evaluate implements Evaluator.
func (e *PerformanceEvaluator) Evaluate(_ context.Context, runs []*EvalRun) (*Report, error) {
	if len(runs) == 0 {
		return &Report{
			Evaluator: e.Name(),
			PassRate:  1,
			Metrics:   map[string]float64{},
			Timestamp: time.Now(),
		}, nil
	}

	latencies := make([]float64, 0, len(runs))
	totalTokens := 0
	for _, r := range runs {
		latencies = append(latencies, float64(r.Duration.Milliseconds()))
		if r.Output != nil {
			// RunOutput.Messages doesn't have token counts directly;
			// count via Metadata["total_tokens"] if available, else 0.
			if v, ok := r.Output.Metadata["total_tokens"]; ok {
				switch n := v.(type) {
				case int:
					totalTokens += n
				case float64:
					totalTokens += int(n)
				}
			}
		}
	}

	sort.Float64s(latencies)
	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)

	avgTokens := 0.0
	if len(runs) > 0 {
		avgTokens = float64(totalTokens) / float64(len(runs))
	}

	metrics := map[string]float64{
		"latency_p50_ms":     p50,
		"latency_p95_ms":     p95,
		"latency_p99_ms":     p99,
		"total_runs":         float64(len(runs)),
		"total_tokens":       float64(totalTokens),
		"avg_tokens_per_run": avgTokens,
	}

	passRate := 1.0
	var failures []*Failure
	if e.cfg.MaxLatencyMs > 0 && p99 > e.cfg.MaxLatencyMs {
		passRate = 0
		failures = append(failures, &Failure{
			Input:    "latency_check",
			Expected: formatMs(e.cfg.MaxLatencyMs),
			Actual:   formatMs(p99),
			Reason:   "p99 latency exceeds MaxLatencyMs threshold",
		})
	}

	return &Report{
		Evaluator: e.Name(),
		PassRate:  passRate,
		Metrics:   metrics,
		Failures:  failures,
		Timestamp: time.Now(),
	}, nil
}

// percentile returns the p-th percentile value from a sorted slice (nearest-rank).
func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func formatMs(ms float64) string {
	return fmt.Sprintf("%.2fms", ms)
}
