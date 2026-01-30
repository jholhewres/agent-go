# AgentGo

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-80.8%25-brightgreen.svg)](docs/DEVELOPMENT.md#testing-standards)
[![Release](https://img.shields.io/badge/release-v2.0.0-blue.svg)](CHANGELOG.md)

**AgentGo** is a high-performance multi-agent framework written in Go. It keeps the KISS philosophy of the Agno project while embracing Go’s strengths: lightweight goroutines, a tiny memory footprint, single static binaries, and a batteries-included toolchain.

> **📜 Credits**: AgentGo is a fork and major enhancement of the original **agno-go** project, which itself was inspired by the Python [**Agno**](https://github.com/agno-agi/agno) framework. We extend the original with Learning System, Agent Skills, Prompt Engineering, pgvector support, and critical bug fixes. See [CREDITS.md](CREDITS.md) for full attribution.
>
> **⚡ New Architecture**: AgentGo offers **two deployment modes**:
> - **SDK** – Embed agents directly in your Go applications (zero HTTP overhead)
> - **AgentOS** – Production-ready REST API server for microservices and multi-language integrations

---

## Feature Highlights

### 🆕 New in v2.0.0

- **🧠 Learning System** – Agents that learn and improve over time with persistent user profiles, memories, and transferable knowledge across sessions. Supports PostgreSQL, SQLite, and MongoDB backends with GDPR compliance.
- **🎯 Agent Skills** – Modular capability system following the Agent Skills specification (agentskills.io). Load skills from local filesystem (`.agentgo/skills/`) or databases. Automatic tool generation: `get_skill_instructions`, `get_skill_reference`, `get_skill_script`.
- **💡 Prompt Engineering** – Advanced template system with Go `text/template` engine, variable validation (string, int, bool, array, object), few-shot examples injection, and YAML-based prompt definitions.
- **💾 pgvector Support** – PostgreSQL-based vector database with HNSW & IVFFlat indexes, cosine similarity search, metadata filtering, and batch operations. Complements existing ChromaDB and RedisDB support.

### ⚡ Performance & Architecture

- **🚀 Extreme Performance** – Agent instantiation in ~180 ns and ~1.2 KB memory per agent. **16× faster** than Python version with native goroutines and no GIL limitations.
- **🧩 Flexible Architecture** – Three orchestration patterns: **Agents** (autonomous), **Teams** (4 coordination modes: Sequential, Parallel, Leader-Follower, Consensus), **Workflows** (5 primitives: Step, Condition, Loop, Parallel, Router). Mix and compose freely.
- **🤖 Two Deployment Modes**:
  - **SDK** – Embed agents in Go apps with zero HTTP overhead
  - **AgentOS** – Production REST API server with OpenAPI 3.0, session storage (PostgreSQL/MongoDB/SQLite), SSE streaming, health checks, structured logging, CORS, timeouts, and caching

### 🤝 Model Providers & Reasoning

- **🔌 15+ Model Providers** – OpenAI (GPT-4o, o1, o3), Anthropic Claude (3.5 Sonnet, Opus), Google Gemini (2.0 Flash, Pro), DeepSeek, GLM (智谱AI), Ollama (local), Groq, Cohere, Together, Vertex AI, Azure OpenAI, Perplexity, Mistral, Fireworks, and more.
- **🧠 Unified Reasoning** – First-class reasoning support across all providers. Extract thinking/reasoning from OpenAI o1/o3, Claude extended thinking, Gemini 2.0 thinking mode, and Vertex AI Reasoning Engine with unified API.

### 🛠️ Tools & Knowledge

- **🔧 25+ Built-in Tools** – Calculator, HTTP client, file operations (read/write/list/delete), web search, Tavily search, Jira, Google Sheets, Gmail, ElevenLabs TTS, YouTube, and more. SDK for custom toolkits.
- **💾 Knowledge & RAG** – Three vector database options: **pgvector** (PostgreSQL), **ChromaDB**, **RedisDB**. Includes document chunking, embeddings, batching, caching, and metadata filtering.
- **📚 MCP Support** – Model Context Protocol integration for external tools and services.

### 🛡️ Safety & Observability

- **🛡️ Guardrails & Hooks** – Prompt injection guard, custom pre/post hooks (PreToolUse, PostToolUse, Stop), media validation, and input/output filters.
- **🪄 Session Management** – Shared sessions across agents/teams/workflows, async + sync summaries, run metadata with cache hits, and pluggable storage adapters.
- **📊 Rich Observability** – SSE event stream with reasoning snapshots, structured logging, Logfire integration, OpenTelemetry spans, token usage tracking, and run analytics.

---

## Getting Started

```bash
go get github.com/jholhewres/agent-go@latest
```

### Option 1: SDK (Embedded Agents)

Perfect for Go applications, CLIs, and libraries:

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
		Name:     "Math Assistant",
		Model:    model,
		Toolkits: []toolkit.Toolkit{calculator.New()},
	})

	output, _ := ag.Run(context.Background(), "What is 25 * 4 + 15?")
	fmt.Println(output.Content)
}
```

### Option 2: AgentOS (REST API Server)

Perfect for microservices and multi-language integrations:

```bash
# Run with Docker
docker compose up -d

# Or start manually
go run cmd/server/main.go

# Test the API
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/v1/agents/assistant/run \
  -H "Content-Type: application/json" \
  -d '{"input":"Hello, AgentOS!"}'
```

See [pkg/agentos/README.md](pkg/agentos/README.md) for full AgentOS documentation.

### AgentOS HTTP tips

- Override the default `GET /health` path via `Config.HealthPath` or attach your
  own handlers with `server.GetHealthRouter("/health-check").GET("", customHandler)`.
- `/openapi.yaml` always serves the current OpenAPI document and `/docs` hosts a
  self-contained Swagger UI bundle. Call `server.Resync()` after hot-swapping
  routers to remount the documentation routes.
- Sample probes:
  ```bash
  curl http://localhost:8080/health-check
  curl http://localhost:8080/openapi.yaml | head -n 5
  ```

---

## Documentation

| Resource | Link |
| --- | --- |
| **AgentOS API** | [pkg/agentos/README.md](pkg/agentos/README.md) |
| **Examples** | [cmd/examples/](cmd/examples/) |
| **Knowledge RAG** | [pkg/agentgo/knowledge/README.md](pkg/agentgo/knowledge/README.md) |
| **Changelog** | [CHANGELOG.md](CHANGELOG.md) |
| **Development** | [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) |
| **Internal Docs** | [docs/](docs/) |

## What's New in v1.2.9

- **EvoLink Media Agents** – First-class EvoLink provider under `pkg/agentgo/providers/evolink` and `pkg/agentgo/models/evolink/*` for text, image, and video generation, with example workflows in `website/examples/evolink-media-agents.md`.
- **Knowledge Upload Chunking** – `POST /api/v1/knowledge/content` now accepts `chunk_size` and `chunk_overlap` (JSON, `text/plain` query params, multipart form fields) and records these values plus `chunker_type` in stored chunk metadata.
- **AgentOS HTTP Tips in Docs** – The AgentOS API page now documents how to customize health endpoints, rely on `/openapi.yaml` and `/docs`, and when to call `server.Resync()` after router changes.

## Session Runtime & Storage Parity

- **Session reuse & history:** `POST /api/v1/sessions/{id}/reuse` shares conversations between agents, teams, and workflows, while `GET /api/v1/sessions/{id}/history?num_messages=N&stream_events=true` mirrors Python-style pagination and SSE toggles.
- **Summaries:** `GET`/`POST /api/v1/sessions/{id}/summary` trigger synchronous or async summaries via `session.SummaryManager`, persisting the latest snapshot on completion.
- **Run metadata:** responses include `runs[*].cache_hit`, `runs[*].status`, timestamps, and cancellation reasons to power audits and resumptions.
- **Pluggable stores:** choose Postgres, MongoDB, or SQLite adapters with identical JSON contracts; fall back to in-memory storage for tests.
- **Response caching:** enable the built-in cache to deduplicate identical model calls across runs.

```go
db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
store, _ := postgres.NewStorage(db, postgres.WithSchema("agentos"))

summaryModel, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: os.Getenv("OPENAI_API_KEY")})
summary := session.NewSummaryManager(
    session.WithSummaryModel(summaryModel),
    session.WithSummaryTimeout(45*time.Second),
)

server, _ := agentos.NewServer(&agentos.Config{
    Address:        ":8080",
    SessionStorage: store,
    SummaryManager: summary,
})

agent, _ := agent.New(agent.Config{
    Name:        "Cached Assistant",
    Model:       summaryModel,
    EnableCache: true,
})
```

`docs/README.md` explains the split between the public site (`website/`) and internal design notes (`docs/`).

---

### Knowledge upload chunking

`POST /api/v1/knowledge/content` now accepts `chunk_size` and `chunk_overlap`
in both JSON and multipart form uploads. Provide them as query parameters for
`text/plain` requests or as form fields (`chunk_size=2000&chunk_overlap=250`) when
streaming files. Both values propagate into the reader metadata, so downstream
pipelines can inspect how documents were segmented.

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -F file=@docs/guide.md \
  -F chunk_size=1800 \
  -F chunk_overlap=200 \
  -F metadata='{"source_url":"https://example.com/guide"}'
```

Each stored chunk automatically records `chunk_size`, `chunk_overlap`, and the
`chunker_type` used—mirroring the AgentOS Python responses.

---

## Observability & Reasoning

- **SSE Event Stream** – `POST /api/v1/agents/{id}/run/stream?types=run_start,reasoning,token,complete` emits structured events. `reasoning` events carry token counts, redacted transcripts, and provider metadata; `complete` events summarise the run.
- **Logfire Integration** – `cmd/examples/logfire_observability` shows how to export spans with OpenTelemetry (build with `-tags logfire`). Detailed walkthrough: [`docs/release/logfire_observability.md`](docs/release/logfire_observability.md).

---

### Anthropic Claude betas & context management

Set `anthropic.Config.Betas` to opt into long-context beta deployments and use
`anthropic.Config.ContextManagement` (or `req.Extra["context_management"]`) to
attach `applied_edits` and other context-management hints. The Go client merges
config-level and per-request metadata, and surfaced `context_management` payloads
end up in `RunOutput.Metadata`, so tool builders can inspect `applied_edits`
directly.

```go
model, _ := anthropic.New("claude-3-5-sonnet", anthropic.Config{
    APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
    Betas:   []string{"context-1m-2025-08-07"},
    ContextManagement: map[string]interface{}{"applied_edits": []string{"trim_history"}},
})
```

---

## Example Catalogue

| Example | Highlights | Run |
| --- | --- | --- |
| **Simple Agent** (`cmd/examples/simple_agent/`) | GPT‑4o mini, calculator toolkit, single agent | `go run cmd/examples/simple_agent/main.go` |
| **Claude Agent** (`cmd/examples/claude_agent/`) | Anthropic Claude 3.5, HTTP + calculator tools | `go run cmd/examples/claude_agent/main.go` |
| **Ollama Agent** (`cmd/examples/ollama_agent/`) | Local Llama 3 via Ollama, file operations | `go run cmd/examples/ollama_agent/main.go` |
| **Team Demo** (`cmd/examples/team_demo/`) | 4 coordination modes, researcher + writer workflow | `go run cmd/examples/team_demo/main.go` |
| **Workflow Demo** (`cmd/examples/workflow_demo/`) | Step / condition / loop / parallel orchestration | `go run cmd/examples/workflow_demo/main.go` |
| **RAG Demo** (`cmd/examples/rag_demo/`) | ChromaDB, embeddings, document Q&A | `go run cmd/examples/rag_demo/main.go` |
| **Reasoning Demo** (`examples/reasoning/`) | OpenAI o1 / Gemini 2.5 thinking extraction | `go run examples/reasoning/main.go` |
| **Logfire Observability** (`cmd/examples/logfire_observability/`) | OpenTelemetry spans + reasoning metadata | `go run -tags logfire cmd/examples/logfire_observability/main.go` |

More details live in the [Examples documentation](website/examples/index.md).

---

## Development & Contribution

1. Read [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) for tooling, linting, and testing workflow.
2. Docs: follow the structure in [`docs/README.md`](docs/README.md) and update the VitePress site (`website/`) when promoting features.
3. Run targeted tests, then `go test ./...` (with `GOCACHE=$(pwd)/.gocache` when sandboxed).
4. Submit PRs with lint/test evidence. Adhere to Conventional Commits.

For ongoing changes and release scope, see [`CHANGELOG.md`](CHANGELOG.md) and the VitePress site’s release notes (`website/release-notes.md`).

---

## License

MIT © [Contributors](https://github.com/jholhewres/agent-go/graphs/contributors)

## Credits & Acknowledgments

**AgentGo** is built on the excellent foundation provided by:

### 🔧 Original Go Implementation
**agno-go** by [@rexleimo](https://github.com/rexleimo)  
Repository: https://github.com/rexleimo/agno-go  
License: MIT

### 🐍 Python Inspiration
**Agno** - The original multi-agent framework  
Repository: https://github.com/agno-agi/agno  
Documentation: https://docs.agno.com  
License: Apache-2.0

### 🚀 AgentGo Enhancements

AgentGo extends the original agno-go with:
- **Learning System** - Agents that learn and improve over time with user profiles, memories, and knowledge transfer
- **Agent Skills** - Modular capabilities system following the Agent Skills specification
- **Advanced Reasoning** - Unified reasoning support across all model providers
- **pgvector Integration** - PostgreSQL-based vector database support
- **Prompt Engineering** - Template system with versioning and optimization

See [CREDITS.md](CREDITS.md) for complete attribution and detailed information.

---

**Maintainer**: Jhol Hewres ([@jholhewres](https://github.com/jholhewres))  
**Repository**: https://github.com/jholhewres/agent-go
