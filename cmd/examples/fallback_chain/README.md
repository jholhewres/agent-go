# Fallback Chain Example

This example demonstrates the `pkg/agentgo/models/fallback` package, which
chains multiple model providers so that if the primary model fails (rate limit,
timeout, etc.) the next model in the chain is tried automatically.

## Prerequisites

- Go 1.24+
- `OPENAI_API_KEY` environment variable

## How to Run

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/examples/fallback_chain/
```

## Related Docs

- [Models guide](../../../website/guide/models.md)
- [fallback package](../../../pkg/agentgo/models/fallback/)
