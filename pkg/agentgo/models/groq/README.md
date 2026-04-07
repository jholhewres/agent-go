# Groq Model Integration

Ultra-fast LLM inference integration for AgentGo, delivering industry-leading inference speeds via Groq.

## Features

- **Ultra-Fast Inference**: Leverages Groq's LPU (Language Processing Unit) for up to 10x faster inference
- **OpenAI Compatible**: Uses the OpenAI API format for easy integration
- **Tool Support**: Full support for Function Calling
- **Streaming Responses**: Supports both streaming and non-streaming inference modes
- **Multiple Models**: Supports LLaMA 3.1, Mixtral, Gemma, and more

## Supported Models

### LLaMA Models (Meta)
- `llama-3.1-8b-instant` - Fastest inference speed (recommended)
- `llama-3.1-70b-versatile` - Most powerful performance
- `llama-3.3-70b-versatile` - Latest version

### Mixtral Models (Mistral AI)
- `mixtral-8x7b-32768` - Mixture of Experts architecture

### Gemma Models (Google)
- `gemma2-9b-it` - Compact but powerful

### Special Purpose Models
- `whisper-large-v3` - Speech recognition
- `llama-guard-3-8b` - Content moderation

## Quick Start

### 1. Get an API Key

Visit [Groq Console](https://console.groq.com/keys) to obtain a free API key.

### 2. Set Environment Variable

```bash
export GROQ_API_KEY=gsk-...
```

### 3. Usage Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jholhewres/agent-go/pkg/agentgo/agent"
    "github.com/jholhewres/agent-go/pkg/agentgo/models/groq"
    "github.com/jholhewres/agent-go/pkg/agentgo/tools/calculator"
    "github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
)

func main() {
    // Create Groq model
    model, err := groq.New(groq.ModelLlama38B, groq.Config{
        APIKey:      "gsk-...",
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create Agent
    agent, err := agent.New(agent.Config{
        Name:         "Groq Agent",
        Model:        model,
        Toolkits:     []toolkit.Toolkit{calculator.New()},
        Instructions: "You are a helpful assistant powered by Groq.",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Run Agent
    output, err := agent.Run(context.Background(), "Calculate 123 + 456")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

## Configuration Options

```go
type Config struct {
    APIKey      string        // Groq API key (required)
    BaseURL     string        // API base URL (default: https://api.groq.com/openai/v1)
    Temperature float64       // Temperature parameter (0.0-2.0)
    MaxTokens   int           // Maximum number of tokens to generate
    Timeout     time.Duration // Request timeout (default: 60s)
}
```

## Performance

Groq's LPU architecture delivers:

- **Inference Speed**: Up to 10x faster than traditional cloud LLM providers
- **Latency**: Extremely low time-to-first-token latency
- **Throughput**: High concurrent request support

### Benchmark Example

```
Model: llama-3.1-8b-instant
Input tokens: 50
Output tokens: 100
Time: ~0.5s (vs ~5s for traditional providers)
```

## Running Examples

```bash
# Set API key
export GROQ_API_KEY=gsk-your-api-key

# Run the example program
go run cmd/examples/groq_agent/main.go
```

## Testing

```bash
# Run unit tests
go test ./pkg/agentgo/models/groq/

# Run tests with coverage report
go test -v -coverprofile=coverage.out ./pkg/agentgo/models/groq/
go tool cover -html=coverage.out
```

**Current test coverage**: 52.4%

## API Documentation

### Creating a Model

```go
model, err := groq.New(modelID string, config Config) (*Groq, error)
```

### Querying Model Information

```go
info, found := groq.GetModelInfo(groq.ModelLlama38B)
if found {
    fmt.Printf("Model: %s\n", info.Name)
    fmt.Printf("Context: %d tokens\n", info.ContextWindow)
    fmt.Printf("Supports Tools: %v\n", info.SupportsTools)
}
```

### Invoking the Model

```go
// Synchronous invocation
response, err := model.Invoke(ctx, &models.InvokeRequest{
    Messages: messages,
    Tools:    tools,
})

// Streaming invocation
chunks, err := model.InvokeStream(ctx, &models.InvokeRequest{
    Messages: messages,
})
for chunk := range chunks {
    fmt.Print(chunk.Content)
}
```

## Advantages

### vs OpenAI
- 10x faster inference speed
- Higher free tier limits
- Open-source model options

### vs Anthropic
- Lower latency
- Higher throughput
- Comparable quality (LLaMA 3.1 70B)

### vs Self-Hosted
- No hardware investment required
- Automatic scaling
- Better performance

## Limitations

- Requires internet connection
- Free tier has rate limits
- Fewer model choices compared to OpenAI

## Resources

- [Groq Website](https://groq.com/)
- [API Documentation](https://console.groq.com/docs)
- [Get API Key](https://console.groq.com/keys)
- [Model List](https://console.groq.com/docs/models)

## License

This integration follows the AgentGo project license. Groq API usage is subject to Groq's Terms of Service.
