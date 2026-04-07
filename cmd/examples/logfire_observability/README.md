# Logfire Observability Example

This example instruments an OpenAI-powered agent with OpenTelemetry traces and
ships them to a Logfire (or any OTLP-compatible) backend. It demonstrates how
to integrate distributed tracing into AgentGo agents.

> **Note:** This example requires the `logfire` build tag and an OTLP endpoint.

## Prerequisites

- Go 1.24+
- `OPENAI_API_KEY` environment variable
- `LOGFIRE_WRITE_TOKEN` environment variable
- `LOGFIRE_OTLP_ENDPOINT` environment variable (default: `https://logfire-api.pydantic.dev`)

## How to Run

```bash
export OPENAI_API_KEY=sk-...
export LOGFIRE_WRITE_TOKEN=...
go run -tags logfire ./cmd/examples/logfire_observability/
```

## Related Docs

- [Observability guide](../../../website/advanced/observability.md)
- [Logfire example](../../../website/examples/logfire-observability.md)
