# Groq Agent Example

This example creates an agent backed by Groq's ultra-fast LLaMA 3.1 8B
inference and equips it with a calculator toolkit. It demonstrates using the
`pkg/agentgo/models/groq` provider for low-latency responses.

## Prerequisites

- Go 1.24+
- `GROQ_API_KEY` environment variable (obtain from <https://console.groq.com/keys>)

## How to Run

```bash
export GROQ_API_KEY=gsk-...
go run ./cmd/examples/groq_agent/
```

## Related Docs

- [Groq agent example](../../../website/examples/ollama-agent.md)
- [groq package](../../../pkg/agentgo/models/groq/)
