# DeepSeek Agent Example

This example creates an agent powered by the DeepSeek model (`deepseek-chat`)
and equips it with a calculator toolkit. It demonstrates how to use the
`pkg/agentgo/models/deepseek` provider with AgentGo.

## Prerequisites

- Go 1.24+
- `DEEPSEEK_API_KEY` environment variable (obtain from <https://platform.deepseek.com>)

## How to Run

```bash
export DEEPSEEK_API_KEY=sk-...
go run ./cmd/examples/deepseek_agent/
```

## Related Docs

- [Models guide](../../../website/guide/models.md)
- [deepseek package](../../../pkg/agentgo/models/deepseek/)
