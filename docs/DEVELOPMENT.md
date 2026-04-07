# Development Guide

Essential information for developers contributing to AgentGo.

---

## Setup

### Requirements

- **Go** 1.21+
- **Git**
- **Node.js** 20+ (for documentation site)
- **golangci-lint** (optional, for linting)

### Getting Started

```bash
go mod download

# Set API keys for testing
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...

# (Optional) Install dev tools
make install-tools
```

---

## Commands

### Testing

```bash
make test                                    # All tests (race detection + coverage)
go test -v ./pkg/agentgo/agent/...           # Specific package
make coverage                                # HTML coverage report
go test -bench=. -benchmem ./pkg/agentgo/agent/  # Benchmarks
```

### Code Quality

```bash
make fmt      # gofmt + goimports
make lint     # golangci-lint
make vet      # go vet
```

### Building

```bash
make build    # Build examples to bin/
make clean    # Remove build artifacts
```

---

## Code Style

- Follow idiomatic Go: tabs, `UpperCamelCase` exports, `lowerCamelCase` locals
- Constructors named `NewX`, configs suffixed `Config`, interfaces ending in `er`
- Document exported symbols with GoDoc comments
- Wrap errors: `fmt.Errorf("context: %w", err)`
- Run `make fmt` before pushing

### Error Handling

```go
if err != nil {
    return nil, fmt.Errorf("failed to create agent: %w", err)
}
```

### Context Usage

```go
func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // implementation
    }
}
```

---

## Testing Standards

- **Core packages**: >70% test coverage
- **New features**: must include tests
- **Bug fixes**: must include regression tests
- Keep tests table-driven and colocated (`*_test.go`)

---

## Project Structure

```
agent-go/
├── cmd/
│   └── examples/            # Example programs
├── pkg/
│   ├── agentgo/
│   │   ├── agent/           # Core Agent
│   │   ├── team/            # Multi-agent (4 modes)
│   │   ├── workflow/        # Workflow engine (5 primitives)
│   │   ├── models/          # 15+ LLM providers
│   │   ├── tools/           # Toolkits
│   │   ├── memory/          # Conversation memory
│   │   ├── session/         # Session persistence
│   │   ├── structured/      # Structured output (JSON Schema)
│   │   ├── hooks/           # Tool execution hooks
│   │   ├── guardrails/      # Input/output guardrails
│   │   ├── knowledge/       # RAG: loaders + chunkers
│   │   ├── vectordb/        # Vector databases
│   │   ├── embeddings/      # Embedding providers
│   │   ├── mcp/             # Model Context Protocol
│   │   ├── learning/        # User memory/learning
│   │   └── types/           # Core types and errors
│   └── agentos/             # HTTP API server (AgentOS)
├── docs/                    # Contributor docs
├── website/                 # VitePress documentation site
├── openspec/                # Change proposal specs
└── Makefile
```

---

## Adding Components

### Adding a Model Provider

1. Create `pkg/agentgo/models/<provider>/`
2. Implement the `models.Model` interface:
   ```go
   type Model interface {
       Invoke(ctx context.Context, req *InvokeRequest) (*types.ModelResponse, error)
       InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan types.ResponseChunk, error)
       GetProvider() string
       GetID() string
   }
   ```
3. Add tests in `<provider>_test.go`
4. Reference `models/openai/openai.go` as an example

### Adding a Tool

1. Create `pkg/agentgo/tools/<tool>/`
2. Embed `toolkit.BaseToolkit`, register functions
3. Add tests
4. Reference `tools/calculator/calculator.go` as an example

---

## Git Workflow

### Commit Messages

```
<type>(<scope>): <subject>

feat(agent): add streaming support
fix(models): fix openai timeout issue
test(agent): add unit tests for memory
docs(readme): update installation guide
```

Types: `feat`, `fix`, `test`, `docs`, `refactor`, `perf`, `chore`

### Pull Request Process

1. Create feature branch: `git checkout -b feature/my-feature`
2. Ensure CI passes, coverage maintained, code documented
3. Use Squash Merge, delete feature branch after merge

---

## Documentation Site (VitePress)

The documentation site lives in `website/`. To preview locally:

```bash
npm install
npm run docs:dev        # Dev server at http://localhost:5173
npm run docs:build      # Production build
npm run docs:preview    # Preview production build
```

The site auto-deploys to GitHub Pages on push to `main` via `.github/workflows/deploy-docs.yml`.

---

## Before Submitting

1. `make fmt` — format code
2. `make test` — all tests pass
3. `make lint` — no lint errors (if golangci-lint installed)
4. `make coverage` — verify coverage maintained

---

## Getting Help

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and discussions
- **Pull Requests**: Code review via PR comments
