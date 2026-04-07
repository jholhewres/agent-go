package eval

import (
	"context"
	"fmt"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
)

// Evaluator is the interface all evaluators must implement.
type Evaluator interface {
	// Evaluate processes a slice of EvalRun results and returns a Report.
	Evaluate(ctx context.Context, runs []*EvalRun) (*Report, error)
	// Name returns the evaluator identifier used in report keys.
	Name() string
}

// EvalRun holds the inputs, outputs, and metadata for a single agent execution.
type EvalRun struct {
	Input          string
	ExpectedOutput string
	Output         *agent.RunOutput
	Duration       time.Duration
	Err            error
	Metadata       map[string]any
}

// Report is the result of one Evaluator applied to a set of EvalRuns.
type Report struct {
	Evaluator string             `json:"evaluator"`
	PassRate  float64            `json:"pass_rate"`
	Metrics   map[string]float64 `json:"metrics"`
	Failures  []*Failure         `json:"failures,omitempty"`
	Timestamp time.Time          `json:"timestamp"`
}

// Failure records one failed evaluation case.
type Failure struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Reason   string `json:"reason"`
}

// TestCase is one item in a Suite.
type TestCase struct {
	Name     string
	Input    string
	Expected string
	Tags     []string
}

// Suite bundles test cases with the evaluators that should score them.
type Suite struct {
	Name       string
	Cases      []*TestCase
	Evaluators []Evaluator
}

// RunSuite executes every TestCase against the agent, collects EvalRuns, then
// applies each Evaluator. It returns a map of evaluator-name → *Report.
func RunSuite(ctx context.Context, a *agent.Agent, suite *Suite) (map[string]*Report, error) {
	runs := make([]*EvalRun, 0, len(suite.Cases))
	for _, tc := range suite.Cases {
		start := time.Now()
		out, err := a.Run(ctx, tc.Input)
		dur := time.Since(start)

		r := &EvalRun{
			Input:          tc.Input,
			ExpectedOutput: tc.Expected,
			Output:         out,
			Duration:       dur,
			Err:            err,
			Metadata:       map[string]any{"case_name": tc.Name, "tags": tc.Tags},
		}
		runs = append(runs, r)
	}

	reports := make(map[string]*Report, len(suite.Evaluators))
	for _, ev := range suite.Evaluators {
		rep, err := ev.Evaluate(ctx, runs)
		if err != nil {
			return nil, fmt.Errorf("evaluator %q failed: %w", ev.Name(), err)
		}
		reports[ev.Name()] = rep
	}
	return reports, nil
}
