# A2A Server Example

This example demonstrates the Agent-to-Agent (A2A) protocol server from the
`pkg/agentos/a2a` package. It starts an HTTP server that exposes a mock agent
over the A2A REST interface, allowing other agents or clients to discover and
invoke it remotely.

## Prerequisites

- Go 1.24+
- No API key required (uses a mock agent)

## How to Run

```bash
go run ./cmd/examples/a2a_server/
```

By default the server listens on `:8080`. Set the `PORT` environment variable
to override:

```bash
PORT=9090 go run ./cmd/examples/a2a_server/
```

## Related Docs

- [pkg/agentos/a2a](../../../pkg/agentos/a2a/)
- [AgentOS guide](../../../website/guide/agent.md)
