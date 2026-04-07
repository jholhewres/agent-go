# Search Agent Example

This example creates a GPT-4o-mini-powered agent equipped with the search
toolkit from `pkg/agentgo/tools/search`. It demonstrates how to give an agent
web-search capabilities using the AgentGo tool system.

## Prerequisites

- Go 1.24+
- `OPENAI_API_KEY` environment variable

## How to Run

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/examples/search_agent/
```

## Related Docs

- [Tools guide](../../../website/guide/tools.md)
- [search toolkit](../../../pkg/agentgo/tools/search/)
