# eval

Evaluation framework for AgentGo agents. Provides composable evaluators, a test-suite runner, and CI-friendly report formats (JSON and JUnit XML).

## Evaluators

### AccuracyEvaluator

Compares each agent output against an expected string in one of three modes:

| Mode | Description |
|------|-------------|
| `MatchExact` | Case-insensitive, trimmed equality |
| `MatchContains` | Case-insensitive substring match |
| `MatchRegexp` | Full Go regexp match |

```go
ev := eval.NewAccuracyEvaluator(eval.MatchContains)
```

Metrics produced: `pass_rate`.

### PerformanceEvaluator

Calculates latency percentiles (p50 / p95 / p99) and token statistics from a set of runs. Optionally flags the suite as failed when p99 exceeds a threshold.

```go
ev := eval.NewPerformanceEvaluator(eval.PerformanceConfig{
    MaxLatencyMs: 2000, // optional; 0 = no threshold
})
```

Metrics produced: `latency_p50_ms`, `latency_p95_ms`, `latency_p99_ms`, `total_runs`, `total_tokens`, `avg_tokens_per_run`.

Token counts are read from `RunOutput.Metadata["total_tokens"]` when present.

### ReliabilityEvaluator

Measures error rates, retry attempts, and fallback usage. Metadata keys it reads from `RunOutput.Metadata`:

| Key | Type | Meaning |
|-----|------|---------|
| `retries` | int / float64 | Number of retry attempts for this run |
| `fallback_index` | int / float64 | Present and >= 0 when a fallback model was used |

```go
ev := eval.NewReliabilityEvaluator()
```

Metrics produced: `error_rate`, `retry_attempts_total`, `fallback_used_count`.

### JudgeEvaluator

Uses a second agent as an LLM judge. The judge receives the input, expected output, and actual output, and must return `{"pass": bool, "reason": string}`.

```go
judgeAgent, _ := agent.New(agent.Config{Name: "judge", Model: myJudgeModel})
ev := eval.NewJudgeEvaluator(
    judgeAgent,
    "You are a strict evaluator. Return JSON: {\"pass\": true|false, \"reason\": \"...\"}",
)
```

Metrics produced: `pass_rate`.

## Running a Suite

```go
suite := &eval.Suite{
    Name: "my-suite",
    Cases: []*eval.TestCase{
        {Name: "q1", Input: "Capital of France?", Expected: "Paris"},
        {Name: "q2", Input: "Capital of Germany?", Expected: "Berlin"},
    },
    Evaluators: []eval.Evaluator{
        eval.NewAccuracyEvaluator(eval.MatchContains),
        eval.NewPerformanceEvaluator(eval.PerformanceConfig{}),
        eval.NewReliabilityEvaluator(),
    },
}

reports, err := eval.RunSuite(ctx, myAgent, suite)
```

## Report Formats

### JSON

```go
var buf bytes.Buffer
eval.WriteJSON(&buf, reports)
// buf contains a deterministic (sorted-keys) JSON object
```

### JUnit XML (CI integration)

```go
var buf bytes.Buffer
eval.WriteJUnit(&buf, reports, "my-suite")
// write to file for upload to Jenkins / GitLab / GitHub Actions
```

Example CI command (GitHub Actions):

```yaml
- name: Run agent eval
  run: go run ./cmd/examples/eval_demo > /tmp/eval-report.xml

- name: Upload JUnit results
  uses: actions/upload-artifact@v4
  with:
    name: eval-junit
    path: /tmp/eval-report.xml
```

For GitLab CI:

```yaml
agent-eval:
  script:
    - go run ./cmd/examples/eval_demo > eval-report.xml
  artifacts:
    reports:
      junit: eval-report.xml
```

## Example

See `cmd/examples/eval_demo/main.go` for a complete working example.
