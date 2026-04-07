# Gemini Agent Example

This example creates an agent backed by Google's Gemini Pro model and equips
it with a calculator toolkit. It demonstrates how to use the
`pkg/agentgo/models/gemini` provider with AgentGo.

## Prerequisites

- Go 1.24+
- `GEMINI_API_KEY` environment variable (obtain from <https://aistudio.google.com>)

## How to Run

```bash
export GEMINI_API_KEY=AIza...
go run ./cmd/examples/gemini_agent/
```

## Related Docs

- [Models guide](../../../website/guide/models.md)
- [gemini package](../../../pkg/agentgo/models/gemini/)
