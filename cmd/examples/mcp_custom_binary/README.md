# MCP Custom Binary Example

This example shows how to connect the AgentGo MCP client to a custom binary
executable (e.g. a Python script or any process implementing the Model Context
Protocol). It covers whitelisted commands, relative/absolute paths, and custom
whitelist configuration.

## Prerequisites

- Go 1.24+
- No API key required for the build/demo itself
- An MCP-compatible process to connect to (optional for full run)

## How to Run

```bash
go run ./cmd/examples/mcp_custom_binary/
```

## Related Docs

- [MCP guide](../../../website/guide/mcp.md)
- [mcp/client package](../../../pkg/agentgo/mcp/client/)
