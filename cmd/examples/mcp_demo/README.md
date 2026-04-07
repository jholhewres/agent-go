# MCP Demo

This example demonstrates connecting to a running MCP (Model Context Protocol)
server via the `pkg/agentgo/mcp/client` package and exposing the server's tools
to an agent through `pkg/agentgo/mcp/toolkit`.

## Prerequisites

- Go 1.24+
- A running MCP server (e.g. `uvx mcp install @modelcontextprotocol/server-calculator`)
- No API key required for the build

## How to Run

```bash
# Start an MCP server first, then:
go run ./cmd/examples/mcp_demo/
```

## Related Docs

- [MCP guide](../../../website/guide/mcp.md)
- [MCP implementation](../../../pkg/agentgo/mcp/IMPLEMENTATION.md)
