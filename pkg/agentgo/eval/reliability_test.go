package eval

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
)

func TestReliabilityEvaluator_NoErrors(t *testing.T) {
	ev := NewReliabilityEvaluator()
	runs := []*EvalRun{
		{Input: "q1", Output: &agent.RunOutput{Content: "a"}, Duration: 10 * time.Millisecond},
		{Input: "q2", Output: &agent.RunOutput{Content: "b"}, Duration: 10 * time.Millisecond},
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.Metrics["error_rate"] != 0 {
		t.Errorf("error_rate = %.2f, want 0", rep.Metrics["error_rate"])
	}
	if rep.PassRate != 1.0 {
		t.Errorf("passRate = %.2f, want 1.0", rep.PassRate)
	}
}

func TestReliabilityEvaluator_WithErrors(t *testing.T) {
	ev := NewReliabilityEvaluator()
	runs := []*EvalRun{
		{Input: "q1", Output: &agent.RunOutput{Content: "a"}, Duration: 10 * time.Millisecond},
		{Input: "q2", Err: errors.New("timeout"), Duration: 10 * time.Millisecond},
		{Input: "q3", Err: errors.New("bad gateway"), Duration: 10 * time.Millisecond},
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 2.0 / 3.0
	if rep.Metrics["error_rate"] < want-0.01 || rep.Metrics["error_rate"] > want+0.01 {
		t.Errorf("error_rate = %.4f, want ~%.4f", rep.Metrics["error_rate"], want)
	}
}

func TestReliabilityEvaluator_Retries(t *testing.T) {
	ev := NewReliabilityEvaluator()
	runs := []*EvalRun{
		{
			Input: "q1",
			Output: &agent.RunOutput{
				Content:  "a",
				Metadata: map[string]interface{}{"retries": 3},
			},
			Duration: 10 * time.Millisecond,
		},
		{
			Input: "q2",
			Output: &agent.RunOutput{
				Content:  "b",
				Metadata: map[string]interface{}{"retries": 1},
			},
			Duration: 10 * time.Millisecond,
		},
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.Metrics["retry_attempts_total"] != 4 {
		t.Errorf("retry_attempts_total = %.0f, want 4", rep.Metrics["retry_attempts_total"])
	}
}

func TestReliabilityEvaluator_Fallback(t *testing.T) {
	ev := NewReliabilityEvaluator()
	runs := []*EvalRun{
		{
			Input: "q1",
			Output: &agent.RunOutput{
				Content:  "a",
				Metadata: map[string]interface{}{"fallback_index": 1},
			},
			Duration: 10 * time.Millisecond,
		},
		{
			Input:    "q2",
			Output:   &agent.RunOutput{Content: "b"},
			Duration: 10 * time.Millisecond,
		},
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.Metrics["fallback_used_count"] != 1 {
		t.Errorf("fallback_used_count = %.0f, want 1", rep.Metrics["fallback_used_count"])
	}
}

func TestReliabilityEvaluator_Empty(t *testing.T) {
	ev := NewReliabilityEvaluator()
	rep, err := ev.Evaluate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 1 {
		t.Errorf("expected passRate=1 for empty runs")
	}
}
