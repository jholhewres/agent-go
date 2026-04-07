# GLM (Zhipu AI) Model Integration

GLM (Zhipu AI) model integration for the AgentGo framework.

## Features

- **Full API Support**
  - Synchronous calls (`Invoke`)
  - Streaming responses (`InvokeStream`)
  - Tool/Function calling

- **JWT Authentication**
  - Secure HMAC-SHA256 signing
  - Automatic token generation
  - 7-day token expiration

- **Well-Tested**
  - 57.2% test coverage
  - Unit tests for all core functions
  - Mock server testing

## Supported Models

- **glm-4** - Main chat model
- **glm-4v** - Vision model (multimodal)
- **glm-3-turbo** - Faster, lower-cost model
- **charglm-3** - Character role-playing model

## Installation

```bash
go get github.com/jholhewres/agent-go
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/jholhewres/agent-go/pkg/agentgo/models/glm"
)

func main() {
    // Create GLM model
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"), // Format: {key_id}.{key_secret}
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Make a request
    resp, err := model.Invoke(context.Background(), &models.InvokeRequest{
        Messages: []*types.Message{
            types.NewUserMessage("Hello! Please introduce yourself."),
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Content)
}
```

### With Agent

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/jholhewres/agent-go/pkg/agentgo/agent"
    "github.com/jholhewres/agent-go/pkg/agentgo/models/glm"
    "github.com/jholhewres/agent-go/pkg/agentgo/tools/calculator"
    "github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
)

func main() {
    // Create GLM model
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create agent with GLM model and tools
    ag, err := agent.New(agent.Config{
        Name:         "GLM Assistant",
        Model:        model,
        Toolkits:     []toolkit.Toolkit{calculator.New()},
        Instructions: "You are a helpful AI assistant that can use the calculator tool to help users.",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Run agent
    output, err := ag.Run(context.Background(), "Calculate the result of 123 * 456")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

### Streaming Response

```go
// Create streaming request
chunks, err := model.InvokeStream(context.Background(), &models.InvokeRequest{
    Messages: []*types.Message{
        types.NewUserMessage("Write a poem about artificial intelligence"),
    },
})
if err != nil {
    log.Fatal(err)
}

// Process chunks
for chunk := range chunks {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    if chunk.Content != "" {
        fmt.Print(chunk.Content)
    }
    if chunk.Done {
        break
    }
}
fmt.Println()
```

## Configuration

### Environment Variables

```bash
# GLM API Key (required) - Format: {key_id}.{key_secret}
export ZHIPUAI_API_KEY=your-key-id.your-key-secret

# Custom Base URL (optional)
export ZHIPUAI_BASE_URL=https://open.bigmodel.cn/api/paas/v4
```

### Config Options

```go
type Config struct {
    APIKey      string  // Required: API key in format {key_id}.{key_secret}

    BaseURL     string  // Optional: Custom API endpoint
                        // Default: https://open.bigmodel.cn/api/paas/v4

    Temperature float64 // Optional: Temperature parameter (0.0-1.0)

    MaxTokens   int     // Optional: Maximum tokens to generate

    TopP        float64 // Optional: Top-p sampling parameter

    DoSample    bool    // Optional: Whether to use sampling
}
```

## API Key Format

GLM API keys consist of two parts separated by a dot:

```
{key_id}.{key_secret}
```

Example:
```
a1b2c3d4e5f6.g7h8i9j0k1l2m3n4
```

You can get your API key from: https://open.bigmodel.cn/

## Authentication

GLM uses JWT (JSON Web Token) authentication with HMAC-SHA256 signing:

1. API key is parsed into `key_id` and `key_secret`
2. JWT token is generated with claims: `api_key`, `timestamp`, `exp`
3. Token is signed using `key_secret` with HS256 algorithm
4. Token is sent in `Authorization: Bearer {token}` header

Tokens are valid for 7 days and are automatically regenerated for each request.

## Examples

See the complete example at: [cmd/examples/glm_agent/](../../../../cmd/examples/glm_agent/)

Run the example:
```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
go run cmd/examples/glm_agent/main.go
```

## Testing

Run tests:
```bash
go test -v ./pkg/agentgo/models/glm/...
```

Run tests with coverage:
```bash
go test -v -cover ./pkg/agentgo/models/glm/...
```

## Error Handling

The GLM client returns typed errors for better error handling:

```go
resp, err := model.Invoke(ctx, req)
if err != nil {
    switch e := err.(type) {
    case *types.InvalidConfigError:
        // Configuration error (e.g., missing API key)
        log.Printf("Config error: %v", e)

    case *types.APIError:
        // API call error (e.g., network error, API error)
        log.Printf("API error: %v", e)

    default:
        // Other errors
        log.Printf("Error: %v", err)
    }
}
```

## Limitations

- Streaming mode (`InvokeStream`) returns content chunks but does not currently support partial tool calls
- Some GLM-specific features like `web_search` tool are not yet exposed in the API

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see [LICENSE](../../../../LICENSE) for details

## Links

- **GLM Official Website**: https://www.bigmodel.cn/
- **API Documentation**: https://open.bigmodel.cn/dev/api
- **AgentGo Repository**: https://github.com/jholhewres/agent-go
