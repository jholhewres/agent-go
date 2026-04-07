# Subagent Demo

This example demonstrates how to compose agents hierarchically: one agent is
wrapped as a tool using `pkg/agentgo/tools/agenttool` and called by a parent
agent. This enables multi-level agent delegation patterns.

## Prerequisites

- Go 1.24+
- `OPENAI_API_KEY` environment variable

## How to Run

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/examples/subagent_demo/
```

## Related Docs

- [Agent guide](../../../website/guide/agent.md)
- [agenttool package](../../../pkg/agentgo/tools/agenttool/)
