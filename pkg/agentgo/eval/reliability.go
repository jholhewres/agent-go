package eval

import (
	"context"
	"time"
)

// ReliabilityEvaluator measures error rates, retry attempts, and fallback usage.
// It reads metadata from EvalRun.Output.Metadata using the keys:
//   - "retries"        (int or float64) — number of retry attempts for this run
//   - "fallback_index" (int or float64) — present (and >= 0) when a fallback was used
type ReliabilityEvaluator struct{}

// NewReliabilityEvaluator constructs a ReliabilityEvaluator.
func NewReliabilityEvaluator() *ReliabilityEvaluator {
	return &ReliabilityEvaluator{}
}

// Name implements Evaluator.
func (e *ReliabilityEvaluator) Name() string { return "reliability" }

// Evaluate implements Evaluator.
func (e *ReliabilityEvaluator) Evaluate(_ context.Context, runs []*EvalRun) (*Report, error) {
	if len(runs) == 0 {
		return &Report{
			Evaluator: e.Name(),
			PassRate:  1,
			Metrics:   map[string]float64{"error_rate": 0, "retry_attempts_total": 0, "fallback_used_count": 0},
			Timestamp: time.Now(),
		}, nil
	}

	errCount := 0
	retryTotal := 0
	fallbackCount := 0

	for _, r := range runs {
		if r.Err != nil {
			errCount++
		}
		if r.Output != nil {
			if v, ok := r.Output.Metadata["retries"]; ok {
				retryTotal += toInt(v)
			}
			if v, ok := r.Output.Metadata["fallback_index"]; ok {
				if toInt(v) >= 0 {
					fallbackCount++
				}
			}
		}
	}

	total := float64(len(runs))
	errorRate := float64(errCount) / total

	return &Report{
		Evaluator: e.Name(),
		PassRate:  1 - errorRate,
		Metrics: map[string]float64{
			"error_rate":           errorRate,
			"retry_attempts_total": float64(retryTotal),
			"fallback_used_count":  float64(fallbackCount),
		},
		Timestamp: time.Now(),
	}, nil
}

// toInt converts an interface{} numeric value to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}
