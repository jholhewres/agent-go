# A2A (Agent-to-Agent) Interface

The A2A interface provides standardized agent-to-agent communication for AgentGo, based on JSON-RPC 2.0.

## Features

- **JSON-RPC 2.0** - Standardized protocol
- **REST API** - HTTP endpoints
- **Streaming Support** - Server-Sent Events
- **Multimedia Support** - Text, images, files
- **Simple Integration** - Expose agents in a few lines of code

## Quick Start

### 1. Create an Agent

```go
type MyAgent struct {
    ID   string
    Name string
}

func (a *MyAgent) Run(ctx context.Context, input string) (interface{}, error) {
    return &a2a.RunOutput{
        Content: "Hello from agent!",
    }, nil
}

func (a *MyAgent) GetID() string { return a.ID }
func (a *MyAgent) GetName() string { return a.Name }
```

### 2. Create A2A Interface

```go
a2aInterface, err := a2a.New(a2a.Config{
    Agents: []a2a.Entity{myAgent},
    Prefix: "/a2a",
})
```

### 3. Register Routes

```go
router := gin.Default()
a2aInterface.RegisterRoutes(router)
router.Run(":7777")
```

### 4. Call the Agent

```bash
curl -X POST http://localhost:7777/a2a/message/send \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "id": "req-1",
    "params": {
      "message": {
        "messageId": "msg-1",
        "role": "user",
        "agentId": "my-agent",
        "contextId": "session-123",
        "parts": [{"kind": "text", "text": "Hello!"}]
      }
    }
  }'
```

## API Endpoints

### POST /a2a/message/send

Send message to agent (non-streaming).

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "message/send",
  "id": "request-id",
  "params": {
    "message": {
      "messageId": "msg-id",
      "role": "user",
      "agentId": "agent-id",
      "contextId": "context-id",
      "parts": [
        {"kind": "text", "text": "Hello"}
      ]
    }
  }
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "id": "request-id",
  "result": {
    "task": {
      "id": "task-123",
      "context_id": "context-id",
      "status": "completed",
      "history": [...]
    }
  }
}
```

### POST /a2a/message/stream

Send message to agent (streaming).

Uses Server-Sent Events for real-time responses.

## Message Part Types

### Text

```json
{
  "kind": "text",
  "text": "Message content"
}
```

### File (URI)

```json
{
  "kind": "file",
  "file": {
    "uri": "https://example.com/image.png",
    "mimeType": "image/png",
    "name": "image.png"
  }
}
```

### File (Bytes)

```json
{
  "kind": "file",
  "file": {
    "bytes": "base64-encoded-content",
    "mimeType": "image/png",
    "name": "image.png"
  }
}
```

### Data

```json
{
  "kind": "data",
  "data": {
    "content": "{\"key\": \"value\"}",
    "mimeType": "application/json"
  }
}
```

## Error Handling

A2A uses standard JSON-RPC 2.0 error codes:

| Code   | Meaning            |
|--------|--------------------|
| -32700 | Parse error        |
| -32600 | Invalid request    |
| -32601 | Method not found   |
| -32602 | Invalid params     |
| -32603 | Internal error     |
| -32000 | Server error       |

**Error Response Example**:
```json
{
  "jsonrpc": "2.0",
  "id": "req-1",
  "error": {
    "code": -32600,
    "message": "Invalid request: agentId is required"
  }
}
```

## Complete Example

See `cmd/examples/a2a_server/main.go` for a complete working example.

## Architecture

```
┌─────────────┐
│HTTP Client  │
└──────┬──────┘
       │ JSON-RPC 2.0 Request
       ▼
┌──────────────────────┐
│  A2A Interface       │
│  ┌────────────────┐  │
│  │ Validator      │  │ Validate request
│  └────────┬───────┘  │
│           ▼          │
│  ┌────────────────┐  │
│  │ Mapper         │  │ A2A → RunInput
│  └────────┬───────┘  │
│           ▼          │
│  ┌────────────────┐  │
│  │ Entity         │  │ Agent/Team/Workflow
│  │ (Run)          │  │
│  └────────┬───────┘  │
│           ▼          │
│  ┌────────────────┐  │
│  │ Mapper         │  │ RunOutput → A2A
│  └────────┬───────┘  │
└───────────┼──────────┘
            ▼
     JSON-RPC 2.0 Response
```

## Compatibility

Compatible with Python Agno's A2A implementation; can interoperate with Python agents.

## License

Apache License 2.0
