# AgentGo

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-80.8%25-brightgreen.svg)](docs/DEVELOPMENT.md#testing-standards)
[![zread](https://img.shields.io/badge/Ask_Zread-_.svg?style=flat&color=00b0aa&labelColor=000000&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTQuOTYxNTYgMS42MDAxSDIuMjQxNTZDMS44ODgxIDEuNjAwMSAxLjYwMTU2IDEuODg2NjQgMS42MDE1NiAyLjI0MDFWNC45NjAxQzEuNjAxNTYgNS4zMTM1NiAxLjg4ODEgNS42MDAxIDIuMjQxNTYgNS42MDAxSDQuOTYxNTZDNS4zMTUwMiA1LjYwMDEgNS42MDE1NiA1LjMxMzU2IDUuNjAxNTYgNC45NjAxVjIuMjQwMUM1LjYwMTU2IDEuODg2NjQgNS4zMTUwMiAxLjYwMDEgNC45NjE1NiAxLjYwMDFaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00Ljk2MTU2IDEwLjM5OTlIMi4yNDE1NkMxLjg4ODEgMTAuMzk5OSAxLjYwMTU2IDEwLjY4NjQgMS42MDE1NiAxMS4wMzk5VjEzLjc1OTlDMS42MDE1NiAxNC4xMTM0IDEuODg4MSAxNC4zOTk5IDIuMjQxNTYgMTQuMzk5OUg0Ljk2MTU2QzUuMzE1MDIgMTQuMzk5OSA1LjYwMTU2IDE0LjExMzQgNS42MDE1NiAxMy43NTk5VjExLjAzOTlDNS42MDE1NiAxMC42ODY0IDUuMzE1MDIgMTAuMzk5OSA0Ljk2MTU2IDEwLjM5OTlaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik0xMy43NTg0IDEuNjAwMUgxMS4wMzg0QzEwLjY4NSAxLjYwMDEgMTAuMzk4NCAxLjg4NjY0IDEwLjM5ODQgMi4yNDAxVjQuOTYwMUMxMC4zOTg0IDUuMzEzNTYgMTAuNjg1IDUuNjAwMSAxMS4wMzg0IDUuNjAwMUgxMy43NTg0QzE0LjExMTkgNS42MDAxIDE0LjM5ODQgNS4zMTM1NiAxNC4zOTg0IDQuOTYwMVYyLjI0MDFDMTQuMzk4NCAxLjg4NjY0IDE0LjExMTkgMS42MDAxIDEzLjc1ODQgMS42MDAxWiIgZmlsbD0iI2ZmZiIvPgo8cGF0aCBkPSJNNCAxMkwxMiA0TDQgMTJaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00IDEyTDEyIDQiIHN0cm9rZT0iI2ZmZiIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPgo8L3N2Zz4K&logoColor=ffffff)](https://zread.ai/jholhewres/agent-go)

> **Build AI agents in Go.** Connect LLMs to tools, chain multiple agents, and deploy as SDK or REST API.

AgentGo lets you create AI agents that call LLM APIs (OpenAI, Anthropic, Gemini, Ollama, etc.), use tools, collaborate in teams, and run structured workflows — all with Go's performance (~180ns agent creation, ~1.2KB memory per agent).

[**Documentation →**](https://zread.ai/jholhewres/agent-go)

---

## Quick Start

```bash
go get github.com/jholhewres/agent-go@latest
```

### SDK Mode

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/jholhewres/agent-go/pkg/agentgo/agent"
    "github.com/jholhewres/agent-go/pkg/agentgo/models/openai"
    "github.com/jholhewres/agent-go/pkg/agentgo/tools/calculator"
    "github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    ag, _ := agent.New(agent.Config{
        Name:     "Assistant",
        Model:    model,
        Toolkits: []toolkit.Toolkit{calculator.New()},
    })

    output, _ := ag.Run(context.Background(), "What is 25 * 4 + 15?")
    fmt.Println(output.Content)
}
```

### AgentOS Mode (REST API)

```bash
go run cmd/server/main.go
curl http://localhost:8080/health
```

See [pkg/agentos/README.md](pkg/agentos/README.md) for AgentOS documentation.

---

## What It Does

- **Agents** — Single agents with LLM + tools (calculator, HTTP, file, custom)
- **Teams** — Multi-agent collaboration (sequential, parallel, leader-follower, consensus)
- **Workflows** — Step-based orchestration (step, condition, loop, parallel, router)
- **Structured Output** — `RunTyped[T]()` returns typed Go structs from LLM responses
- **Model Fallback** — Chain multiple models with automatic retry and fallback
- **Sub-Agents** — Spawn child agents or use agents as tools for other agents
- **Guardrails & Hooks** — Pre/post tool execution hooks, approval gates, input/output validation
- **15+ Providers** — OpenAI, Anthropic, Gemini, DeepSeek, GLM, Ollama, Groq, and more
- **Session & Memory** — Conversation persistence, user learning, history injection

---

## Documentation

| Resource | Link |
| --- | --- |
| **Full Documentation** | [https://zread.ai/jholhewres/agent-go](https://zread.ai/jholhewres/agent-go) |
| **AgentOS API** | [pkg/agentos/README.md](pkg/agentos/README.md) |
| **Examples** | [cmd/examples/](cmd/examples/) |
| **Development** | [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) |
| **Changelog** | [CHANGELOG.md](CHANGELOG.md) |

---

## Examples

| Example | Description |
| --- | --- |
| [Simple Agent](cmd/examples/simple_agent/) | GPT-4o mini + calculator |
| [Structured Output](cmd/examples/structured_output/) | `RunTyped[T]()` with JSON schema |
| [Fallback Chain](cmd/examples/fallback_chain/) | Model fallback with retry |
| [Sub-Agent Demo](cmd/examples/subagent_demo/) | Agent-as-tool + SpawnAll |
| [Team Demo](cmd/examples/team_demo/) | Multi-agent coordination |
| [Workflow Demo](cmd/examples/workflow_demo/) | Step/Condition/Loop orchestration |
| [RAG Demo](cmd/examples/rag_demo/) | ChromaDB + embeddings |

Full examples catalog → [cmd/examples/](cmd/examples/)

---

## Development

```bash
# Run tests
make test

# Format code
make fmt

# Lint
make lint
```

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for complete guidelines.

---

## License

MIT © [Contributors](https://github.com/jholhewres/agent-go/graphs/contributors)

---

**Maintainer**: [Jhol Hewres](https://github.com/jholhewres)
