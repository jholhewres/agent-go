package eval

import (
	"context"
	"testing"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
)

func makeRun(input, expected, actual string) *EvalRun {
	return &EvalRun{
		Input:          input,
		ExpectedOutput: expected,
		Output:         &agent.RunOutput{Content: actual},
		Duration:       10 * time.Millisecond,
	}
}

func TestAccuracyEvaluator_Exact(t *testing.T) {
	ev := NewAccuracyEvaluator(MatchExact)
	runs := []*EvalRun{
		makeRun("q1", "hello", "hello"),
		makeRun("q2", "hello", "world"),
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 0.5 {
		t.Errorf("pass_rate = %.2f, want 0.50", rep.PassRate)
	}
	if len(rep.Failures) != 1 {
		t.Errorf("failures = %d, want 1", len(rep.Failures))
	}
}

func TestAccuracyEvaluator_Contains(t *testing.T) {
	ev := NewAccuracyEvaluator(MatchContains)
	runs := []*EvalRun{
		makeRun("q1", "brown", "the quick brown fox"),
		makeRun("q2", "missing", "the quick brown fox"),
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 0.5 {
		t.Errorf("pass_rate = %.2f, want 0.50", rep.PassRate)
	}
}

func TestAccuracyEvaluator_Regexp(t *testing.T) {
	ev := NewAccuracyEvaluator(MatchRegexp)
	runs := []*EvalRun{
		makeRun("q1", `^\d{4}$`, "2024"),
		makeRun("q2", `^\d{4}$`, "abc"),
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 0.5 {
		t.Errorf("pass_rate = %.2f, want 0.50", rep.PassRate)
	}
}

func TestAccuracyEvaluator_Empty(t *testing.T) {
	ev := NewAccuracyEvaluator(MatchExact)
	rep, err := ev.Evaluate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 0 {
		t.Errorf("expected 0 pass_rate for empty runs")
	}
}

func TestAccuracyEvaluator_AllPass(t *testing.T) {
	ev := NewAccuracyEvaluator(MatchContains)
	runs := []*EvalRun{
		makeRun("q1", "fox", "the quick brown fox"),
		makeRun("q2", "quick", "the quick brown fox"),
	}
	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 1.0 {
		t.Errorf("pass_rate = %.2f, want 1.0", rep.PassRate)
	}
	if len(rep.Failures) != 0 {
		t.Errorf("expected no failures")
	}
}
