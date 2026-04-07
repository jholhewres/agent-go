# eval_demo

Demonstrates the `pkg/agentgo/eval` evaluation framework using a mock agent.

## What it does

1. Creates an agent backed by a simple mock model (always responds "Paris")
2. Defines a Suite with 5 test cases
3. Runs all 4 evaluators: AccuracyEvaluator, PerformanceEvaluator, ReliabilityEvaluator, JudgeEvaluator
4. Prints a JSON report and a JUnit XML report to stdout

## Run

```bash
go run ./cmd/examples/eval_demo/
```

## Build

```bash
go build ./cmd/examples/eval_demo/
```
