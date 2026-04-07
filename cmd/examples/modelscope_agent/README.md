# ModelScope Agent Example

This example creates an agent powered by Alibaba Cloud's Qwen model via the
ModelScope / DashScope API and equips it with a calculator toolkit. It
demonstrates the `pkg/agentgo/models/modelscope` provider.

## Prerequisites

- Go 1.24+
- `DASHSCOPE_API_KEY` environment variable (obtain from <https://dashscope.aliyuncs.com>)

## How to Run

```bash
export DASHSCOPE_API_KEY=sk-...
go run ./cmd/examples/modelscope_agent/
```

## Related Docs

- [Models guide](../../../website/guide/models.md)
- [modelscope package](../../../pkg/agentgo/models/modelscope/)
