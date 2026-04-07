# Workflow History Example

This example demonstrates the Workflow History feature from
`pkg/agentgo/workflow`, showing how to persist step outputs and replay or
inspect them across workflow runs.

## Prerequisites

- Go 1.24+
- `OPENAI_API_KEY` environment variable

## How to Run

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/examples/workflow_history/
```

## Related Docs

- [Workflow History guide](../../../website/guide/workflow-history.md)
- [workflow package](../../../pkg/agentgo/workflow/)
