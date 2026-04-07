# Tool Hooks Agent Example

This example demonstrates tool execution hooks (`pkg/agentgo/hooks`) that fire
before and after each tool call. A `LoggingToolHook` measures latency and logs
success/failure, while a `RateLimitHook` shows how to block tool calls that
exceed a threshold.

## Prerequisites

- Go 1.24+
- `OPENAI_API_KEY` environment variable

## How to Run

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/examples/tool_hooks_agent/
```

## Related Docs

- [Tool Execution Hooks guide](../../../website/guide/tools.md)
- [hooks package](../../../pkg/agentgo/hooks/)
