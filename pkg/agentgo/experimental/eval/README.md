# experimental/eval

> **Warning: Experimental — API may change without notice.**

Lightweight evaluation harness for benchmarking and comparing LLM models.

## Purpose

Runs `Scenario` test cases against any `models.Model`, captures latency and token usage, and aggregates results into `Summary` and `Comparison` reports serialisable to JSON.

## Main Types

- `Scenario` — `{Input, ExpectedContains}` evaluation case
- `RunMetrics` — per-run latency, token counts, success flag
- `Summary` — aggregate statistics across runs (success rate, avg latency, etc.)
- `Evaluator` — `EvaluateModel`, `CompareModels`

## Minimal Example

```go
import "github.com/jholhewres/agent-go/pkg/agentgo/experimental/eval"

e := &eval.Evaluator{}
runs, summary := e.EvaluateModel(ctx, myModel, []eval.Scenario{
    {Input: "What is 2+2?", ExpectedContains: "4"},
})
fmt.Println(string(summary.JSON()))
```

## Status

**experimental** — no consumers outside the package itself.
