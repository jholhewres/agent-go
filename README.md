# AgentGo

[![Go Version](https://img.shields.io/badge/go-1.24%2B-blue.svg)](https://golang.org/dl/)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](docs/DEVELOPMENT.md#testing-standards)
[![Coverage](https://img.shields.io/badge/coverage-78%25-yellow.svg)](docs/DEVELOPMENT.md#testing-standards)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![GoDocs](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/jholhewres/agent-go)

> **Build high-performance multi-agent systems in Go.** Create agents with LLM access, orchestrate teams, and deploy as SDK or REST API.

## What is agent-go?

AgentGo is a high-performance Go framework for building autonomous and collaborative multi-agent AI systems. Create agents with access to 15+ LLM providers (OpenAI, Anthropic, Gemini, Ollama, and more), connect tools, orchestrate teams of agents, and run structured workflows. Inspired by Python's Agno, optimized for Go: ~180ns agent creation, ~1.2KB memory per agent, native concurrency with zero GIL.

**Ideal for**: Developers building AI systems in Go who need speed, reliability, and control.

## Features

- **Agents** — Single LLM with tools (calculator, HTTP, file, custom)
- **Teams** — Multi-agent collaboration (sequential, parallel, leader-follower, consensus)
- **Workflows** — Step-based orchestration (step, condition, loop, parallel, router)
- **15+ LLM Providers** — OpenAI, Anthropic, Gemini, DeepSeek, Groq, Ollama, GLM, and more
- **30+ Built-in Tools** — Calculator, HTTP, file operations, web search, custom tools
- **Structured Output** — `RunTyped[T]()` returns typed Go structs from LLM responses
- **Memory & RAG** — Conversation persistence, knowledge bases, embeddings (OpenAI, Chroma)
- **Sub-Agents** — Spawn child agents or use agents as tools for other agents
- **Hooks & Guardrails** — Pre/post tool execution hooks, approval gates, validation
- **Session Persistence** — Conversation history, user learning, token counting
- **Model Fallback** — Chain multiple models with automatic retry
- **MCP Support** — Model Context Protocol for external tool integration
- **AgentOS REST API** — Deploy as REST endpoint (Gin/Chi) or use as SDK
- **Performance** — ~180ns agent creation, ~1.2KB per agent, native goroutines

## Installation

```bash
go get github.com/jholhewres/agent-go@latest
```

Requires: **Go 1.24+** ([download](https://golang.org/dl/))

## Quick Start

### Your first agent (without API key)

```go
package main

import (
	"context"
	"fmt"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/calculator"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// MockModel for testing without API keys
type MockModel struct {
	models.BaseModel
}

func (m *MockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return &types.ModelResponse{
		Content: "The answer is 420. (25 * 4 + 15 = 100 + 15 = 115, multiplied by 4 = 460... wait, 25 * 4 = 100, + 15 = 115. Let me recalculate: 25 * 4 + 15 = 115.)",
	}, nil
}

func (m *MockModel) GetProvider() string { return "mock" }
func (m *MockModel) GetID() string       { return "mock-model" }

func main() {
	// Create agent with calculator tool
	ag, _ := agent.New(agent.Config{
		Name:     "MathBot",
		Model:    &MockModel{},
		Toolkits: []toolkit.Toolkit{calculator.New()},
	})

	// Run the agent
	output, _ := ag.Run(context.Background(), "What is 25 * 4 + 15?")
	fmt.Println(output.Content)
	// Output: The answer is 115.
}
```

### With OpenAI (SDK Mode)

```bash
export OPENAI_API_KEY=sk-...
go run cmd/examples/simple_agent/main.go
```

### REST API Mode (AgentOS)

```bash
go run cmd/server/main.go
curl http://localhost:8080/health
```

See [pkg/agentos/README.md](pkg/agentos/README.md) for AgentOS documentation.

## Examples

| Example | Description | Link |
|---------|-------------|------|
| Simple Agent | Single agent + calculator | [cmd/examples/simple_agent/](cmd/examples/simple_agent/) |
| Structured Output | Typed responses with JSON schema | [cmd/examples/structured_output/](cmd/examples/structured_output/) |
| Model Fallback | Chain models with automatic retry | [cmd/examples/fallback_chain/](cmd/examples/fallback_chain/) |
| Sub-Agent Demo | Agent-as-tool + SpawnAll | [cmd/examples/subagent_demo/](cmd/examples/subagent_demo/) |
| Team Demo | Multi-agent collaboration | [cmd/examples/team_demo/](cmd/examples/team_demo/) |
| Workflow Demo | Step/Condition/Loop orchestration | [cmd/examples/workflow_demo/](cmd/examples/workflow_demo/) |
| RAG Demo | ChromaDB + embeddings | [cmd/examples/rag_demo/](cmd/examples/rag_demo/) |
| Claude Agent | Anthropic Claude integration | [cmd/examples/claude_agent/](cmd/examples/claude_agent/) |
| Groq Agent | Ultra-fast inference (LLaMA 3.1) | [cmd/examples/groq_agent/](cmd/examples/groq_agent/) |
| GLM Agent | Zhipu AI (supports Chinese) | [cmd/examples/glm_agent/](cmd/examples/glm_agent/) |

Full catalog → [cmd/examples/](cmd/examples/)

## Supported Models

| Provider | Status | Docs |
|----------|--------|------|
| OpenAI (GPT-4, GPT-4o) | ✅ Stable | [openai/](pkg/agentgo/models/openai/) |
| Anthropic (Claude 3.x) | ✅ Stable | [anthropic/](pkg/agentgo/models/anthropic/) |
| Google Gemini | ✅ Stable | [gemini/](pkg/agentgo/models/gemini/) |
| Groq (LLaMA, Mixtral) | ✅ Stable | [groq/](pkg/agentgo/models/groq/) |
| DeepSeek | ✅ Stable | [deepseek/](pkg/agentgo/models/deepseek/) |
| Zhipu GLM (GLM-4, GLM-4V) | ✅ Stable | [glm/](pkg/agentgo/models/glm/) |
| Ollama (local models) | ✅ Stable | [ollama/](pkg/agentgo/models/ollama/) |
| Perplexity | ✅ Beta | [perplexity/](pkg/agentgo/models/perplexity/) |
| Fireworks | 🟡 Coming | [fireworks/](pkg/agentgo/models/fireworks/) |
| Mistral | 🟡 Coming | [mistral/](pkg/agentgo/models/mistral/) |

[Full list →](pkg/agentgo/models/)

## Architecture

```
User Input
    ↓
 Agent (LLM + Tools)
    ↓
Team (4 modes)  OR  Workflow (5 primitives)
    ↓
Output
```

**Agent** — Runs a single LLM with access to tools. Automatically calls tools in a loop until the task completes or max iterations reached.

**Team** — Coordinates multiple agents using 4 collaboration modes: Sequential (one after another), Parallel (simultaneous), Leader-Follower (leader assigns tasks), or Consensus (agents discuss until agreement).

**Workflow** — Step-based orchestration using 5 primitives: Step (run agent/function), Condition (branch on context), Loop (iterate with exit condition), Parallel (run steps concurrently), Router (dynamic routing).

→ [Full architecture guide](website/advanced/architecture.md)

## Performance

| Operation | Agent-go | Python Agno | Speedup |
|-----------|----------|------------|---------|
| Agent creation | ~180ns | ~2.9µs | 16× |
| Memory per agent | ~1.2KB | ~12KB | 10× |
| Team parallelism | Native goroutines | GIL-bound | ∞ |

→ [Performance benchmarks](website/advanced/performance.md)

## Development

```bash
# Run tests with race detection
make test

# Format code
make fmt

# Lint (requires golangci-lint)
make lint

# Generate coverage report
make coverage
```

→ [Full dev guide](docs/DEVELOPMENT.md)

## Getting Help

- **Issues** — Report bugs and request features: [GitHub Issues](https://github.com/jholhewres/agent-go/issues)
- **Discussions** — Ask questions and share ideas: [GitHub Discussions](https://github.com/jholhewres/agent-go/discussions)
- **Docs** — Full documentation: [website/](website/)

## Project Status

**Current**: v1.4.0 (stable) — See [CHANGELOG.md](CHANGELOG.md) for details.

**Contributing** — Contributions welcome! Open an issue or PR to get started.

## Documentation

| Resource | Link |
|----------|------|
| **Full Docs** | [website/](website/) |
| **API Reference** | [pkg.go.dev/github.com/jholhewres/agent-go](https://pkg.go.dev/github.com/jholhewres/agent-go) |
| **Production Deployment** | [website/advanced/production-deployment.md](website/advanced/production-deployment.md) |
| **AgentOS** | [pkg/agentos/](pkg/agentos/) |
| **Examples** | [cmd/examples/](cmd/examples/) |
| **Development** | [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) |
| **Changelog** | [CHANGELOG.md](CHANGELOG.md) |

## License

MIT © [Contributors](https://github.com/jholhewres/agent-go/graphs/contributors)

Inspired by [Agno](https://github.com/agno-agi/agno) (Python framework).
