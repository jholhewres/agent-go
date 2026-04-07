# Storage Control Example

This example demonstrates how to control whether an agent stores tool messages
and conversation history in memory. It uses the `pkg/agentgo/agent` storage
options to selectively persist or discard messages.

## Prerequisites

- Go 1.24+
- `OPENAI_API_KEY` environment variable

## How to Run

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/examples/storage_control/
```

## Related Docs

- [Memory guide](../../../website/guide/memory.md)
- [Session state guide](../../../website/guide/session-state.md)
