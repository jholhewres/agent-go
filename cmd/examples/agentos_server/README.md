# AgentOS Server Example

This example starts the AgentOS HTTP server and prints the available REST
endpoints. It demonstrates session management and agent invocation over HTTP
using the `pkg/agentos` package.

## Prerequisites

- Go 1.24+
- No API key required

## How to Run

```bash
go run ./cmd/examples/agentos_server/
```

The server listens on `http://localhost:8080` and exposes:

- `GET /health`
- `POST /api/v1/sessions`
- `GET /api/v1/sessions/:id`
- `POST /api/v1/agents/:id/run`

## Related Docs

- [pkg/agentos](../../../pkg/agentos/)
- [Architecture guide](../../../website/advanced/architecture.md)
