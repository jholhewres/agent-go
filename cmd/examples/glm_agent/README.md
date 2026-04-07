# GLM Agent Example (智谱AI)

This example creates an agent powered by Zhipu AI's GLM-4 model and equips it
with a calculator toolkit. It demonstrates Chinese-language conversation using
the `pkg/agentgo/models/glm` provider.

## Prerequisites

- Go 1.24+
- `ZHIPUAI_API_KEY` environment variable — format `{key_id}.{key_secret}`
  (obtain from <https://open.bigmodel.cn>)

## How to Run

```bash
export ZHIPUAI_API_KEY=your-id.your-secret
go run ./cmd/examples/glm_agent/
```

## Related Docs

- [GLM agent example](../../../website/examples/glm-agent.md)
- [glm package](../../../pkg/agentgo/models/glm/)
