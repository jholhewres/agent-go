# Structured Output Example

This example shows how to use the `pkg/agentgo/structured` package to make an
agent return strongly-typed Go structs (e.g. a `CodeReview` struct) instead of
raw text. The model is guided by a JSON schema derived from Go struct tags.

## Prerequisites

- Go 1.24+
- `OPENAI_API_KEY` environment variable

## How to Run

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/examples/structured_output/
```

## Related Docs

- [Agent guide](../../../website/guide/agent.md)
- [structured package](../../../pkg/agentgo/structured/)
