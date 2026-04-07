package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
)

// judgeVerdict is the JSON structure the LLM judge must return.
type judgeVerdict struct {
	Pass   bool   `json:"pass"`
	Reason string `json:"reason"`
}

// JudgeEvaluator uses an LLM agent to score each run.
type JudgeEvaluator struct {
	judge    *agent.Agent
	criteria string
}

// NewJudgeEvaluator constructs a JudgeEvaluator.
// criteria is a plain-English prompt instructing the judge to return JSON
// {"pass": bool, "reason": string}.
func NewJudgeEvaluator(judge *agent.Agent, criteria string) *JudgeEvaluator {
	return &JudgeEvaluator{judge: judge, criteria: criteria}
}

// Name implements Evaluator.
func (e *JudgeEvaluator) Name() string { return "judge" }

// Evaluate implements Evaluator.
func (e *JudgeEvaluator) Evaluate(ctx context.Context, runs []*EvalRun) (*Report, error) {
	if len(runs) == 0 {
		return &Report{
			Evaluator: e.Name(),
			PassRate:  0,
			Metrics:   map[string]float64{"pass_rate": 0},
			Timestamp: time.Now(),
		}, nil
	}

	passed := 0
	var failures []*Failure

	for _, r := range runs {
		actual := ""
		if r.Output != nil {
			actual = r.Output.Content
		}

		prompt := e.buildPrompt(r.Input, r.ExpectedOutput, actual)
		judgeOut, err := e.judge.Run(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("judge run failed for input %q: %w", r.Input, err)
		}

		verdict, err := parseVerdict(judgeOut.Content)
		if err != nil {
			return nil, fmt.Errorf("judge returned unparseable response for input %q: %w", r.Input, err)
		}

		if verdict.Pass {
			passed++
		} else {
			failures = append(failures, &Failure{
				Input:    r.Input,
				Expected: r.ExpectedOutput,
				Actual:   actual,
				Reason:   verdict.Reason,
			})
		}
	}

	passRate := float64(passed) / float64(len(runs))
	return &Report{
		Evaluator: e.Name(),
		PassRate:  passRate,
		Metrics:   map[string]float64{"pass_rate": passRate},
		Failures:  failures,
		Timestamp: time.Now(),
	}, nil
}

// buildPrompt assembles the judge prompt.
func (e *JudgeEvaluator) buildPrompt(input, expected, actual string) string {
	return fmt.Sprintf(`%s

Input: %s
Expected: %s
Actual: %s

Respond ONLY with valid JSON: {"pass": true|false, "reason": "..."}`,
		e.criteria, input, expected, actual)
}

// parseVerdict extracts the first JSON object from the judge's response.
func parseVerdict(content string) (*judgeVerdict, error) {
	// Find the JSON object within the response (the model may add surrounding text).
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no JSON object found in: %q", content)
	}
	raw := content[start : end+1]

	var v judgeVerdict
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}
	return &v, nil
}
