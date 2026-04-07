package eval

import (
	"context"
	"testing"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// mockJudgeModel always returns a fixed JSON verdict.
type mockJudgeModel struct {
	models.BaseModel
	verdict string
}

func (m *mockJudgeModel) Invoke(_ context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return &types.ModelResponse{Content: m.verdict, Model: m.ID}, nil
}

func (m *mockJudgeModel) InvokeStream(_ context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk, 2)
	go func() {
		defer close(ch)
		ch <- types.ResponseChunk{Content: m.verdict}
		ch <- types.ResponseChunk{Done: true}
	}()
	return ch, nil
}

func newJudgeAgent(verdict string) *agent.Agent {
	m := &mockJudgeModel{
		BaseModel: models.BaseModel{ID: "mock-judge", Provider: "mock"},
		verdict:   verdict,
	}
	a, err := agent.New(agent.Config{
		Name:  "judge",
		Model: m,
	})
	if err != nil {
		panic(err)
	}
	return a
}

func TestJudgeEvaluator_AllPass(t *testing.T) {
	judgeAgent := newJudgeAgent(`{"pass": true, "reason": "looks good"}`)
	ev := NewJudgeEvaluator(judgeAgent, "Evaluate if the response is correct.")

	runs := []*EvalRun{
		{Input: "What is 2+2?", ExpectedOutput: "4", Output: &agent.RunOutput{Content: "4"}, Duration: 10 * time.Millisecond},
		{Input: "Capital of France?", ExpectedOutput: "Paris", Output: &agent.RunOutput{Content: "Paris"}, Duration: 10 * time.Millisecond},
	}

	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 1.0 {
		t.Errorf("passRate = %.2f, want 1.0", rep.PassRate)
	}
	if len(rep.Failures) != 0 {
		t.Errorf("expected no failures")
	}
}

func TestJudgeEvaluator_AllFail(t *testing.T) {
	judgeAgent := newJudgeAgent(`{"pass": false, "reason": "wrong answer"}`)
	ev := NewJudgeEvaluator(judgeAgent, "Evaluate if the response is correct.")

	runs := []*EvalRun{
		{Input: "What is 2+2?", ExpectedOutput: "4", Output: &agent.RunOutput{Content: "5"}, Duration: 10 * time.Millisecond},
	}

	rep, err := ev.Evaluate(context.Background(), runs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 0.0 {
		t.Errorf("passRate = %.2f, want 0.0", rep.PassRate)
	}
	if len(rep.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(rep.Failures))
	}
	if rep.Failures[0].Reason != "wrong answer" {
		t.Errorf("failure reason = %q, want 'wrong answer'", rep.Failures[0].Reason)
	}
}

func TestJudgeEvaluator_Empty(t *testing.T) {
	judgeAgent := newJudgeAgent(`{"pass": true, "reason": "ok"}`)
	ev := NewJudgeEvaluator(judgeAgent, "Evaluate.")
	rep, err := ev.Evaluate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.PassRate != 0 {
		t.Errorf("expected passRate=0 for empty runs")
	}
}

func TestParseVerdict(t *testing.T) {
	tests := []struct {
		input    string
		wantPass bool
		wantErr  bool
	}{
		{`{"pass": true, "reason": "ok"}`, true, false},
		{`{"pass": false, "reason": "nope"}`, false, false},
		{`Here is the verdict: {"pass": true, "reason": "good"}`, true, false},
		{`no json here`, false, true},
	}

	for _, tt := range tests {
		v, err := parseVerdict(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("input %q: expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tt.input, err)
			continue
		}
		if v.Pass != tt.wantPass {
			t.Errorf("input %q: pass = %v, want %v", tt.input, v.Pass, tt.wantPass)
		}
	}
}
