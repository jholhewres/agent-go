# Agent with Guardrails Example

This example shows how to attach input/output guardrails and lifecycle hooks to
an agent. It validates user input before the model is called and inspects the
response afterwards using the `pkg/agentgo/guardrails` and
`pkg/agentgo/hooks` packages.

## Prerequisites

- Go 1.24+
- `OPENAI_API_KEY` environment variable

## How to Run

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/examples/agent_with_guardrails/
```

## Related Docs

- [Tool Execution Hooks guide](../../../website/guide/tools.md)
- [guardrails package](../../../pkg/agentgo/guardrails/)
