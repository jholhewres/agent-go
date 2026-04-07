package eval

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MatchMode controls how AccuracyEvaluator compares actual vs expected.
type MatchMode int

const (
	// MatchExact requires the actual output to equal the expected string (case-insensitive trim).
	MatchExact MatchMode = iota
	// MatchContains requires the actual output to contain the expected string (case-insensitive).
	MatchContains
	// MatchRegexp requires the actual output to match the expected string as a Go regexp.
	MatchRegexp
)

// AccuracyEvaluator measures how often agent outputs satisfy the expected value.
type AccuracyEvaluator struct {
	mode MatchMode
}

// NewAccuracyEvaluator constructs an AccuracyEvaluator with the given match mode.
func NewAccuracyEvaluator(mode MatchMode) *AccuracyEvaluator {
	return &AccuracyEvaluator{mode: mode}
}

// Name implements Evaluator.
func (e *AccuracyEvaluator) Name() string { return "accuracy" }

// Evaluate implements Evaluator.
func (e *AccuracyEvaluator) Evaluate(_ context.Context, runs []*EvalRun) (*Report, error) {
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

		ok, err := e.matches(actual, r.ExpectedOutput)
		if err != nil {
			return nil, fmt.Errorf("regexp compile error for expected %q: %w", r.ExpectedOutput, err)
		}
		if ok {
			passed++
		} else {
			failures = append(failures, &Failure{
				Input:    r.Input,
				Expected: r.ExpectedOutput,
				Actual:   actual,
				Reason:   e.failReason(actual, r.ExpectedOutput),
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

func (e *AccuracyEvaluator) matches(actual, expected string) (bool, error) {
	switch e.mode {
	case MatchExact:
		return strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(expected)), nil
	case MatchContains:
		return strings.Contains(strings.ToLower(actual), strings.ToLower(expected)), nil
	case MatchRegexp:
		re, err := regexp.Compile(expected)
		if err != nil {
			return false, err
		}
		return re.MatchString(actual), nil
	default:
		return false, nil
	}
}

func (e *AccuracyEvaluator) failReason(actual, expected string) string {
	switch e.mode {
	case MatchExact:
		return fmt.Sprintf("exact match failed: got %q, want %q", actual, expected)
	case MatchContains:
		return fmt.Sprintf("output does not contain %q", expected)
	case MatchRegexp:
		return fmt.Sprintf("output does not match regexp %q", expected)
	default:
		return "no match"
	}
}
