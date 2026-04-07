package eval

import (
	"context"
	"testing"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
)

func makeRunWithDuration(dur time.Duration) *EvalRun {
	return &EvalRun{
		Input:    "input",
		Output:   &agent.RunOutput{Content: "output"},
		Duration: dur,
	}
}

func TestPerformanceEvaluator_Basic(t *testing.T) {
	ev := NewPerformanceEvaluator(PerformanceConfig{})
	runs := []*EvalRun{
		makeRunWithDuration(10 * time.Millisecond),
		makeRunWithDuration(20 * time.Millisecond),
		makeRunWithDuration(30 * time.Millisecond),
		makeRunWithDuration(40 * time.Millisecond),
		makeRunWithDuration(50 * time.Millisecond),
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.Metrics["total_runs"] != 5 {
		t.Errorf("total_runs = %.0f, want 5", rep.Metrics["total_runs"])
	}
	if rep.Metrics["latency_p50_ms"] <= 0 {
		t.Errorf("p50 should be > 0")
	}
	if rep.Metrics["latency_p95_ms"] <= 0 {
		t.Errorf("p95 should be > 0")
	}
	if rep.Metrics["latency_p99_ms"] <= 0 {
		t.Errorf("p99 should be > 0")
	}
}

func TestPerformanceEvaluator_MaxLatencyThreshold(t *testing.T) {
	ev := NewPerformanceEvaluator(PerformanceConfig{MaxLatencyMs: 5})
	runs := []*EvalRun{
		makeRunWithDuration(100 * time.Millisecond),
		makeRunWithDuration(200 * time.Millisecond),
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 0 {
		t.Errorf("expected passRate=0 when p99 exceeds threshold, got %.2f", rep.PassRate)
	}
	if len(rep.Failures) == 0 {
		t.Errorf("expected at least one failure")
	}
}

func TestPerformanceEvaluator_Empty(t *testing.T) {
	ev := NewPerformanceEvaluator(PerformanceConfig{})
	rep, err := ev.Evaluate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 1 {
		t.Errorf("expected passRate=1 for empty runs")
	}
}

func TestPerformanceEvaluator_TokensFromMetadata(t *testing.T) {
	ev := NewPerformanceEvaluator(PerformanceConfig{})
	runs := []*EvalRun{
		{
			Input:    "q",
			Output:   &agent.RunOutput{Content: "a", Metadata: map[string]interface{}{"total_tokens": 100}},
			Duration: 10 * time.Millisecond,
		},
		{
			Input:    "q2",
			Output:   &agent.RunOutput{Content: "b", Metadata: map[string]interface{}{"total_tokens": 200}},
			Duration: 20 * time.Millisecond,
		},
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.Metrics["total_tokens"] != 300 {
		t.Errorf("total_tokens = %.0f, want 300", rep.Metrics["total_tokens"])
	}
	if rep.Metrics["avg_tokens_per_run"] != 150 {
		t.Errorf("avg_tokens_per_run = %.0f, want 150", rep.Metrics["avg_tokens_per_run"])
	}
}
