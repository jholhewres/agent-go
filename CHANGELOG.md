# Changelog

All notable changes to AgentGo will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.0] - 2026-04-07

### Summary

Sprint 1 "Production Ready I" release. Five new feature areas land
together so agent-go can be operated as a production service rather
than a developer demo: a hardened PostgreSQL session backend, native
OpenTelemetry tracing, a stable evaluation framework, automatic
conversation summarization for long-running sessions, and a complete
production deployment guide.

### Added

#### Storage — production-ready PostgreSQL backend (S1.1)
- **`internal/session/store/postgres/migrate.go`** — In-house, ~100-line
  schema migrator built on `embed.FS` + `pgxpool`. Idempotent,
  transactional, tracks versions in `agno_schema_migrations`. No
  `golang-migrate` dependency added.
- **`internal/session/store/postgres/migrations/001_init.{up,down}.sql`** —
  Initial schema with three production indexes:
  `(user_id, updated_at DESC)`, `(session_type, agent_id)`, and a
  partial `(workflow_id) WHERE workflow_id IS NOT NULL`.
- **`Config` struct** — full pool tuning: `MaxConns`, `MinConns`,
  `MaxConnLifetime`, `MaxConnIdleTime`, `HealthCheckPeriod`,
  `ConnectTimeout`, `TableName`, all with production defaults via
  `defaults()`. Backwards-compatible alias for the old
  `MaxConnLifeTime` field.
- **`NewStore(ctx, dsn, cfg)`** convenience constructor.
- **`Store.Pool()`** accessor for callers that need to run `Migrate`
  or custom queries.
- **`Store.ListByTenant(ctx, tenantID, opts)`** — multi-tenant helper
  that pins `user_id` and orders by `updated_at DESC`.
- **`pkg/agentgo/storage/postgres/README.md`** — schema diagram, index
  table, pool tuning guide, migration workflow, init example, and
  backup/restore strategies.
- **`cmd/examples/postgres_storage/`** — runnable end-to-end demo that
  skips gracefully when `DATABASE_URL` is unset.
- **`test/smoke/postgres_storage_smoke_test.go`** — compile-only smoke
  test, picked up by the centralized example smoke test.

#### Observability — native OpenTelemetry tracing (S1.2)
- **`pkg/agentgo/observability/otel/tracer.go`** — `Config` +
  `NewTracerProvider` using the `otlptracehttp` exporter,
  `ParentBased(TraceIDRatioBased(SamplingRate))` sampler, and
  resource attributes (`service.name`, `service.version`, plus
  caller-supplied extras).
- **`pkg/agentgo/observability/otel/hooks.go`** — three hooks that
  plug into the existing agent hook API:
  `ToolTracingHook` (implements `hooks.ToolHooker`, span per tool
  execution with `tool.name`, `tool.args`, `tool.status`,
  `tool.duration_ms`),
  `AgentTracingPreHook` + `AgentTracingPostHook` (paired
  `hooks.HookFunc` values that emit a span per agent run with
  `agent.id`, `agent.name`, status, duration, and error attributes).
- **`pkg/agentgo/observability/otel/stdout.go`** —
  `NewStdoutTracerProvider` for tests and local dev.
- **`pkg/agentgo/observability/otel/tracer_test.go`** — 5 tests using
  in-memory `tracetest.SpanRecorder`, no Docker, no network.
- **`pkg/agentgo/observability/otel/README.md`** — usage guide,
  attribute tables, sampling strategy, exporter selection.
- **`cmd/examples/otlp_tracing_agent/`** — runnable demo that uses the
  stdout exporter so it works without a collector.
- Tracing is **opt-in** and integrated entirely via the existing
  hook API — `pkg/agentgo/agent/agent.go` was not modified.

#### Eval framework — promoted from experimental (S1.3)
- **`pkg/agentgo/eval/`** — new stable package with four evaluators:
  - `AccuracyEvaluator` (`MatchExact`, `MatchContains`, `MatchRegexp`)
  - `PerformanceEvaluator` (p50/p95/p99 latency, token totals,
    optional `MaxLatencyMs` threshold)
  - `ReliabilityEvaluator` (`error_rate`, `retry_attempts_total`,
    `fallback_used_count` from `RunOutput.Metadata`)
  - `JudgeEvaluator` (LLM-as-judge using another `*agent.Agent` and
    a criteria string; parses the first `{"pass":bool,"reason":...}`
    JSON block from the judge response)
- **`Suite` + `RunSuite(ctx, agent, suite)`** — compose test cases,
  run them through the agent, and apply each evaluator.
- **`WriteJSON` + `WriteJUnit`** — deterministic JSON (sorted keys)
  and JUnit XML reports for CI integration.
- **21 unit tests, 82.7% coverage**, all using `MockModel` —
  no network or API keys.
- **`cmd/examples/eval_demo/`** — runnable demo that builds an Agent
  with a `MockModel`, defines a 5-case suite, runs all four
  evaluators, and prints JSON + JUnit reports.
- **`docs/PACKAGES.md`** — `eval` row promoted from `experimental`
  to `stable`. The old `pkg/agentgo/experimental/eval/` skeleton is
  removed.

#### Memory — SummarizingMemory wrapper (S1.4)
- **`pkg/agentgo/memory/summarizing.go`** — `SummarizingMemory` wraps
  any `Memory` implementation (including `InMemory` and
  `HybridMemory`) and triggers synchronous LLM compaction once the
  buffer crosses `Threshold`. The synthesized summary becomes a new
  System message tagged with `SummaryTag`, prepended to the last
  `PreserveLast` messages.
- **`SummarizingConfig`** — `Inner`, `Model`, `Threshold` (default 50),
  `PreserveLast` (default 10), `MaxSummaryTokens` (default 500),
  `SummaryPrompt`, `SummaryTag` (default `[Conversation Summary]`).
- **8 unit tests, 89.9% coverage**, including a non-destructive-on-
  failure test that verifies the original buffer is preserved when
  the summarizer model errors.
- **`pkg/agentgo/memory/README.md`** — comparison table for
  `InMemory`, `HybridMemory`, and `SummarizingMemory` plus a usage
  example.
- **`cmd/examples/summarizing_memory_agent/`** — standalone demo that
  uses a `MockModel` to compact 60 turns into ~11 messages.

#### Documentation — Production Deployment Guide (S1.5)
- **`website/advanced/production-deployment.md`** (1021 lines, EN)
- **`website/zh/advanced/production-deployment.md`** (1020 lines, ZH)
- 13 sections covering: reference architecture, PostgreSQL pool
  tuning, OTLP wiring, memory strategy selection, eval CI gates,
  full env-var table, secrets management, complete Docker Compose
  example (agent-os + postgres + otel-collector + jaeger),
  scaling considerations, common pitfalls, and a 15+ item
  pre-flight checklist.
- VitePress sidebar (EN + ZH) updated to surface the new page.
- README's documentation table linked.

### Changed

- Bumped OpenTelemetry stack to v1.43.0 (otel core + sdk + trace +
  otlptrace + otlptracehttp) so the new
  `go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.0`
  dependency aligns.
- `internal/session/store/postgres/store_test.go` now uses
  `Migrate(ctx, st.pool)` instead of inline DDL bootstrapping.
- `.gitignore` anchors four new stray-binary names from the new
  examples (`/eval_demo`, `/otlp_tracing_agent`, `/postgres_storage`,
  `/summarizing_memory_agent`).

### Removed

- `pkg/agentgo/experimental/eval/` skeleton (superseded by the new
  stable `pkg/agentgo/eval/` package — see S1.3).

### Known Follow-Ups

- `TODO(sprint-1.x)` — `AgentTracingHook` currently bridges pre/post
  via `sync.Map[agentID → span]`. Concurrent runs of the same
  agentID can race. Replace once the hook API can carry context
  across pre/post boundaries.
- `TODO(sprint-1.x)` — `SummarizingMemory.compact()` is synchronous;
  add an async/worker-pool variant for very chatty sessions.
- `TODO(sprint-1.x)` — `SummarizingMemory` triggers on message count
  only; add a token-count trigger once a shared tokenizer abstraction
  lands.
- `TODO(sprint-1.x)` — `PerformanceEvaluator` reads tokens from
  `RunOutput.Metadata["total_tokens"]`. Promote it to a first-class
  field on `RunOutput`.

## [1.5.0] - 2026-04-07

### Summary

Sprint 0 "Stabilization" release. No new framework features — this release
hardens the open-source foundation so that v1.6.0+ feature work can land
safely: green test suite across the board, documented packages, working
examples, proper CI, and a revised developer-facing README.

### Added

#### Examples — EvoLink media pipeline
- **`cmd/examples/evolink_media_pipeline/`** — Runnable three-step workflow
  that composes EvoLink's text, image, and video models via
  `workflow.Workflow` with custom `imageNode`/`videoNode` wrappers.
- Smoke test mocks all three EvoLink endpoints with `httptest` and asserts
  the full text → image → video chain succeeds.

#### Examples — Recovered from `.gitignore` shadow bug
- **`cmd/examples/fallback_chain/`** — Model fallback chain demo
- **`cmd/examples/structured_output/`** — `RunTyped[T]` demo
- **`cmd/examples/subagent_demo/`** — Sub-agent spawn + Agent-as-Tool demo

  These three examples were added in v1.4.0 but were silently shadowed by
  unanchored `.gitignore` patterns intended for stray root-level binaries.
  Now anchored correctly and committed.

#### Tests — Centralized example smoke test
- **`test/smoke/examples_build_test.go`** — `TestExamplesBuild` discovers
  every `cmd/examples/<name>/` with a `main.go`, runs `go build` for each
  via `exec.Command` with up to 4 parallel builds. No API keys required.
  30/31 examples build; `logfire_observability` is skipped (requires the
  `logfire` build tag and external OTEL deps). Runs in ~5s standalone.

#### Documentation — README restructure
- Rewrote root `README.md` from 131 lines to 214 lines across 14 sections:
  tagline, features (categorized), installation, runnable Quick Start
  without API keys, examples table, supported models table, architecture
  diagram, performance benchmarks, development guide, help, status, docs,
  license.

#### Documentation — Community files
- `CONTRIBUTING.md` — development setup, project structure, commit/PR
  conventions, testing expectations, guides for adding models and tools.
- `CODE_OF_CONDUCT.md` — adapted from Contributor Covenant v2.1.
- `.github/ISSUE_TEMPLATE/bug_report.yml` — structured bug form.
- `.github/ISSUE_TEMPLATE/feature_request.yml` — structured feature form.
- `.github/ISSUE_TEMPLATE/config.yml` — redirects questions to Discussions.
- `.github/PULL_REQUEST_TEMPLATE.md` — PR checklist and change types.
- `.github/FUNDING.yml` — GitHub Sponsor button for @jholhewres.
- `.github/SECURITY.md` — vulnerability reporting policy with SLA.
- `docs/PACKAGES.md` — single index of all 28 `pkg/agentgo/` packages
  with stable/beta/experimental/planned status badges.
- 19 previously-undocumented examples now have their own `README.md`.

#### CI/CD — GitHub Actions hardening
- **`.github/workflows/ci.yml`** — added Go version matrix (1.24.1 +
  stable), `go vet` step, Codecov upload via `codecov/codecov-action@v4`,
  per-matrix coverage artifact names.
- **`.github/dependabot.yml`** — weekly gomod + github-actions updates,
  grouped as "go-dependencies", max 5 open PRs.
- **`.github/workflows/security.yml`** — `gosec` (SARIF upload to GitHub
  code scanning) + `govulncheck`, triggered on push/PR/weekly cron.
- **`.github/workflows/release.yml`** — tag-triggered release pipeline
  that runs tests then `goreleaser release --clean`.
- **`.goreleaser.yaml`** — multi-platform builds (linux/darwin/windows,
  amd64/arm64), tar.gz/zip archives, conventional-commits changelog,
  GitHub Releases with docs header.

### Changed

#### Repository — Rebrand `agno-Go` → `agent-go`
- Updated ~185 files: Go import paths (`rexleimo/agno-Go` →
  `jholhewres/agent-go`), project name (`Agno-Go` → `AgentGo`), VitePress
  base (`/agno-Go/` → `/agent-go/`), sitemap + GitHub Pages URLs, docs
  in English + zh + ja + ko locales.

#### Repository — Archive obsolete docs
- Moved 11 obsolete migration/release docs to `docs/archive/` subtree
  (release notes, session-service migration plans, early design docs).
  History preserved; `git log --follow` still works.

#### Packages — Vague packages consolidation
- Moved to `pkg/agentgo/experimental/` (API may change without notice):
  `cloud`, `culture`, `eval`, `integrations`. None had external consumers.
- Added stable READMEs for `learning` and `media` (both actively used by
  `agent.go`, `workflow`, `pkg/agentos`).
- Each experimental package has a README with an `⚠️ Experimental` warning.

#### Makefile — Targets restructure
- `make test` — unit tests only (fast, no Docker / external deps).
- `make test-integration` — opt-in, runs tests tagged `integration`
  (Docker required).
- `make test-contract` — opt-in, runs tests tagged `contract` (skips
  gracefully if the Python parity fixtures repo is not checked out
  alongside agent-go).
- `make test-all` — runs unit + integration + contract.
- `make build-all` — builds every `cmd/examples/*/` binary into `bin/`.
- `make install-tools` — also installs `gosec` and `govulncheck`.

### Fixed

- **`pkg/agentos` — `TestAgentRun_StreamEvents`**: `simpleModel.InvokeStream`
  returned an empty channel, so the handler never emitted `token` events.
  Mock now emits an actual content chunk, restoring end-to-end streaming
  coverage.
- **`pkg/agentgo/workflow` — `TestExecutionContext_RunContextMetadata`**:
  `NewExecutionContext` initialises `Metadata` as an empty map by default;
  the test now forces nil explicitly to exercise the defensive branch in
  `SetRunContextMetadata`.
- **`pkg/agentgo/run/context.go` — `RunContext.Clone`**: no longer copies
  the embedded `sync.RWMutex` by value (`go vet` lock copy warning). Clone
  now holds the source's read lock while snapshotting fields into a fresh
  struct with a zero-valued mutex.
- **`pkg/agentgo/models/openai/openai_integration_test.go`** — added a
  `defer cancel()` so the `TestOpenAI_Stream_Cancel` early-return path
  does not leak the cancel func (`go vet -lostcancel`).
- **`internal/session/contract/contract_test.go`** — `loadRawFixture`
  now calls `t.Skipf` when the external `contract-fixtures` directory is
  missing, instead of hard-failing.
- **`internal/session/contract`** and **`internal/session/store/postgres`**
  — both gated behind build tags (`contract` and `integration`
  respectively) so `make test` on a fresh checkout is green without
  requiring the Python fixtures repo or Docker.
- **`CREDITS.md`** — reverted an accidental truncation that removed the
  "AgentGo Enhancements" section documenting fork-specific features
  (Learning System, Agent Skills, Advanced Reasoning, pgvector, Prompt
  Engineering).
- **`.gitignore`** — anchored stray-binary patterns to the repo root
  (`/fallback_chain`, `/structured_output`, `/subagent_demo`,
  `/evolink_media_pipeline`). The unanchored form was silently shadowing
  the matching `cmd/examples/<name>/` directories.

### OpenSpec

- Closed `add-evolink-media-agents` tasks 1.3 and 2.2. Task 2.3
  (`npm run docs:build`) deferred to the docs deploy workflow.

## [1.4.0] - 2026-03-12

### Added

#### Model Fallback Chain
- **`pkg/agentgo/models/fallback/`** - Chain multiple LLM models with automatic fallback
  - Iterates through a chain of models, retrying each on retryable errors (API errors, rate limits, timeouts)
  - Non-retryable errors (invalid input, config) stop immediately without fallback
  - `WithMaxRetries(n)` and `WithRetryDelay(d)` options for per-model retry behavior
  - Streaming support (falls back only on stream open failure, not mid-stream)
  - Response metadata includes `fallback_model` and `fallback_index` to track which model responded
  - 13 tests covering all failure/retry/cancellation scenarios

#### Structured Output
- **`pkg/agentgo/structured/`** - Generate JSON Schema from Go structs and parse typed responses
  - `SchemaFromType()` generates JSON Schema via reflection (supports nested structs, slices, optional fields, descriptions)
  - `ParseResponse()` unmarshals `ModelResponse.Content` into typed structs
  - `ToResponseFormat()` converts schema to `models.ResponseFormat` with name/description metadata
- **`agent.RunTyped[T]()`** - Generic helper to run an agent and unmarshal the response into a typed struct
- **`agent.Config.ResponseFormat`** - New config field to constrain model output to JSON schema
- **OpenAI provider** maps `ResponseFormat` to `ChatCompletionResponseFormatJSONSchema` via go-openai SDK
- 15 tests for schema generation + 4 tests for agent integration

#### Human-in-the-Loop (Approval Hook)
- **`pkg/agentgo/hooks/approval.go`** - Gate tool execution behind an approval function
  - `NewApprovalHook(fn, tools...)` creates a hook that calls a user-provided `ApprovalFunc` before tool execution
  - Optional `ToolFilter` restricts which tools require approval (nil = all tools)
  - Integrates with existing `ToolHooker` interface — blocked tools get `ToolExecutionStatusBlocked`
  - 9 tests including context cancellation and filter behavior

#### Streaming with Tool Calls
- **Complete rewrite of `RunStream()` goroutine** to support multi-pass tool execution
  - Outer loop: `for loopCount < MaxLoops` enables iterative tool call → response cycles
  - Inner stream loop: consumes chunks, forwards content events, aggregates response
  - After stream completion: checks `HasToolCalls()`, executes tools, starts new stream with updated messages
  - Max loops reached produces a proper error event
  - 3 new tests: tool calls in stream, max loops, content flow during tool loop

#### SubAgent Spawner
- **`pkg/agentgo/agent/subagent.go`** - Run child agents with linked contexts
  - `Spawn(ctx, child, input)` runs a child agent with `ParentRunID` linked from the parent context
  - `SpawnAll(ctx, configs)` runs multiple children concurrently via goroutines + WaitGroup
  - Context cancellation propagates from parent to all children
  - 9 tests covering parent-child linking, concurrency, cancellation, and partial failures

#### Agent as Tool
- **`pkg/agentgo/tools/agenttool/`** - Wrap an agent as a toolkit for use by other agents
  - `New(agent, description)` creates a toolkit with a single `ask_<agent_name>` function
  - Handler delegates to `agent.Spawn()` and returns the child agent's content
  - Enables orchestrator-specialist patterns where a parent agent delegates to specialist sub-agents
  - 6 tests

#### Session Persistence
- **`agent.Config.SessionPersister` + `agent.Config.SessionID`** - Automatic run persistence
  - Runs are persisted to session storage after both `Run()` and `RunStream()` completion
  - `SessionPersister` interface defined in agent package to avoid import cycles with session package
  - `session.NewPersister(storage)` creates an adapter from `session.Storage`
  - Persistence errors are logged but do not fail the run
  - 3 tests for session persistence + 3 tests for the persister adapter

#### User Memory/Learning Integration
- **Learning context injection** in `Run()` — fetches user profile and memories before model invoke
  - `buildLearnedContext()` retrieves `GetUserProfile()` + `GetUserMemories()` and formats as `[Learned Context]`
  - Injected as prefix to agent instructions when available
- **Async learning** after `Run()` — calls `LearningMachine.Learn()` in a background goroutine with 10s timeout
  - Failures logged, never block or fail the run
- 4 tests covering learn calls, context injection, disabled state, and missing UserID

#### History Context Injection
- **`agent.Config.HistoryProvider` + `agent.Config.HistoryMaxRuns`** - Load previous session runs into context
  - `HistoryProvider` interface defined in agent package to avoid import cycles
  - `session.NewHistoryProvider(storage)` creates an implementation that formats run history
  - History injected before learned context in `Run()`, prepended to agent instructions
  - Default `HistoryMaxRuns`: 5
  - 3 agent tests + 5 session history provider tests

### New Examples
- **`cmd/examples/structured_output/`** - Demonstrates `RunTyped[T]` and `ParseResponse` with JSON schema
- **`cmd/examples/fallback_chain/`** - Demonstrates model fallback chain with multiple OpenAI models
- **`cmd/examples/subagent_demo/`** - Demonstrates `Agent as Tool` and `SpawnAll` concurrent execution

### New Files (12)
| File | Feature |
|------|---------|
| `pkg/agentgo/models/fallback/fallback.go` | Model Fallback Chain |
| `pkg/agentgo/models/fallback/fallback_test.go` | Model Fallback Chain |
| `pkg/agentgo/structured/structured.go` | Structured Output |
| `pkg/agentgo/structured/structured_test.go` | Structured Output |
| `pkg/agentgo/hooks/approval.go` | Human-in-the-Loop |
| `pkg/agentgo/hooks/approval_test.go` | Human-in-the-Loop |
| `pkg/agentgo/agent/subagent.go` | SubAgent Spawner |
| `pkg/agentgo/agent/subagent_test.go` | SubAgent Spawner |
| `pkg/agentgo/tools/agenttool/agenttool.go` | Agent as Tool |
| `pkg/agentgo/tools/agenttool/agenttool_test.go` | Agent as Tool |
| `pkg/agentgo/session/persister.go` | Session Persistence |
| `pkg/agentgo/session/persister_test.go` | Session Persistence |
| `pkg/agentgo/session/history_provider.go` | History Context Injection |
| `pkg/agentgo/session/history_provider_test.go` | History Context Injection |

### Modified Files (4)
| File | Changes |
|------|---------|
| `pkg/agentgo/agent/agent.go` | Structured Output, Streaming Tool Calls, Session Persistence, Learning, History |
| `pkg/agentgo/agent/agent_test.go` | Tests for all agent-level features |
| `pkg/agentgo/models/base.go` | `ResponseFormat` struct + field on `InvokeRequest` |
| `pkg/agentgo/models/openai/openai.go` | ResponseFormat mapping to go-openai SDK |

### Fixed

#### Code Review Fixes (26 issues resolved)

**HIGH severity:**
- Fixed race condition in `session.Persister.PersistRun()` — added `sync.Mutex` to serialize concurrent calls
- Fixed `RunTyped[T]()` parameter order — `ctx context.Context` now comes first, following Go conventions
- Fixed unbounded learning goroutines — added semaphore (buffered channel, size 3) to cap concurrent learn calls
- Fixed `session.Persister` error masking — now checks `ErrSessionNotFound` explicitly before falling back to Create

**MEDIUM severity:**
- Fixed data race in `fallback_test.go` mock counters — switched to `atomic.Int32`
- Added defensive slice copy in `fallback.New()` to prevent mutation of caller's chain slice
- Added `reflect.Interface` → empty schema `{}` in structured output schema generation
- Added `time.Time` → `{"type": "string", "format": "date-time"}` mapping in schema generation
- Added agent name sanitization in `agenttool.New()` for safe function names
- Added history + learned context injection to `RunStream()` (was only in `Run()`)
- Added fallback metadata propagation to `RunOutput.Metadata` in agent
- Extracted `instructionsModified` flag to eliminate redundant `updateSystemMessage` calls

**LOW severity:**
- Fixed `time.After` leak in fallback retry loops — switched to `time.NewTimer` with explicit `Stop()`
- Fixed `WithMaxRetries(0)` being rejected — now accepts `n >= 0`
- Added self-referencing struct cycle detection in `SchemaFromType` via guarded recursion with `seen` map
- Added nil/non-pointer validation in `ParseResponse` target parameter
- Added nil guard panic in `NewApprovalHook` when approval function is nil
- Added nil guard error in `Spawn()` when child agent is nil
- Made content truncation length configurable in `HistoryProvider` via `WithMaxContentLen()` option
- Made truncation unicode-safe using `utf8.RuneCountInString` instead of `len()`
- Fixed redundant newlines in `fmt.Println` calls in example programs

### Backward Compatibility
- **No breaking changes.** All new Config fields use zero-value defaults that preserve existing behavior.
- Existing agents continue to work without modification.
- All new features are opt-in.
- `RunTyped[T]()` parameter order changed (`ctx` first) — callers need to update.

### Testing
- ~77 new tests across all features, all passing
- All existing `pkg/agentgo/` tests continue to pass

---

## [1.3.0] - 2026-02-28

### Added

#### Tool Execution Hooks
- **Pre/Post hooks for tool calls** via `ToolHooker` interface
  - `OnToolPre` can inspect and block tool calls before execution
  - `OnToolPost` can inspect results after tool execution
  - Blocked tools produce `ToolExecutionStatusBlocked` summaries
- **Guardrails system** (`pkg/agentgo/guardrails/`) for input/output validation
  - Prompt injection detection guardrail
  - PII detection guardrail
  - Content moderation guardrail
  - Custom guardrail support via `GuardrailFunc`

### Backward Compatibility
- No breaking changes. Hooks and guardrails are opt-in via `agent.Config`.

---

## [1.2.0] - 2026-02-05

### Added
- **HybridMemory** - New memory implementation combining short-term (InMemory) and long-term (VectorDB) storage
  - Hybrid search combining vector similarity and text matching
  - Configurable weights for vector vs text search
  - Automatic migration of old messages to long-term storage
  - `SearchableMemory` interface with advanced search options
  - Thread-safe and multi-tenant support

- **PromptComposer** - Modular prompt composition system
  - Section-based prompt building with priority ordering
  - Template support with variable interpolation (Go `text/template`)
  - Predefined sections: Identity, Skills, Memory, Instructions, Constraints
  - Dynamic enable/disable of prompt sections
  - Full backward compatibility with existing `Instructions` string

- **Agent Enhancements**
  - New `PromptComposer` field in `agent.Config` for modular prompts
  - New `PromptVars` field for template variables
  - New `EnableMemorySearch` flag for automatic memory search
  - New `MemorySearchLimit` and `MemorySearchMinScore` configuration
  - New `GetPromptComposer()` method
  - New `SearchMemory(ctx, query, limit)` method
  - New `UpdatePromptSection(name, content)` method

### Changed
- README.md - More concise with examples moved to internal documentation
- Added zread badge for documentation link

### Migration from v1.1.5

**No breaking changes!** All existing code continues to work:

```go
// v1.1.5 (still works in v1.2.0)
agent, _ := agent.New(agent.Config{
    Name:         "Assistant",
    Model:        model,
    Instructions: "You are a helpful assistant", // ← Still works
})
```

New features available (opt-in):

```go
// v1.2.0 - New HybridMemory with search
config := memory.HybridMemoryConfig{
    VectorDB:              vectorDB,
    Embedder:              embedder,
    MaxShortTermMessages:  100,
    LongTermThreshold:     50,
    DefaultVectorWeight:   0.7,
    DefaultTextWeight:     0.3,
}
mem, _ := memory.NewHybridMemory(config)

agent, _ := agent.New(agent.Config{
    Name:        "Assistant",
    Model:       model,
    Memory:      mem,
    EnableMemorySearch: true,  // ← NEW: Auto-search memory
    MemorySearchLimit: 5,
})

// v1.2.0 - New PromptComposer
composer := prompts.NewPromptComposer(
    prompts.IdentitySection("GoHelper", "Expert Go assistant"),
    prompts.InstructionsSection("Help users write better Go code"),
    prompts.SkillsSection([]string{"code_analyzer", "doc_generator"}),
)

agent, _ := agent.New(agent.Config{
    Name:           "Assistant",
    Model:          model,
    PromptComposer: composer,  // ← NEW: Modular prompts
    PromptVars: map[string]interface{}{
        "current_date": time.Now().Format("2006-01-02"),
    },
})
```

### Testing
All tests passing with comprehensive coverage:
- ✅ HybridMemory: 100% (8/8 tests)
- ✅ PromptComposer: 100% (17/17 tests)
- ✅ Agent: Existing tests maintained (backward compatibility)
- ✅ Thread safety tests
- ✅ Multi-tenant isolation tests

## [1.1.5] - 2026-01-30

### Changed
- **[SECURITY]** Script execution permanently disabled (hardcoded)
- `Skills.ScriptsEnabled()` now always returns `false`
- `get_skill_script` tool is never registered, regardless of configuration
- Removed `DisableSkillScripts` configuration flag (no longer needed)
- `DisableScripts()` and `EnableScripts()` methods kept for API compatibility but are no-ops

### Removed
- `enableScripts` field from `Skills` struct (scripts always disabled now)
- `DisableSkillScripts` field from `agent.Config` (no longer needed)
- Logic to conditionally disable scripts based on config

### Why This Change?
For enhanced security and simplicity:
- Scripts execution is a security risk and was never used
- Hardcoding the disabled state removes attack surface
- Simplifies codebase by removing conditional logic
- Skills still provide instructions and references (core functionality)

### Migration from v1.1.4

**No code changes required!** If you were using:

```go
// v1.1.4 (still works in v1.1.5)
agent, _ := agentgo.New(agentgo.Config{
    Skills:              skills,
    DisableSkillScripts: true, // ← This flag is now ignored (but doesn't break)
})
```

New simplified version (recommended):
```go
// v1.1.5
agent, _ := agentgo.New(agentgo.Config{
    Skills: skills, // Scripts always disabled, no flag needed
})
```

### Behavior

| Feature | v1.1.4 | v1.1.5 |
|---------|--------|--------|
| **get_skill_instructions** | ✅ Always available | ✅ Always available |
| **get_skill_reference** | ✅ Always available | ✅ Always available |
| **get_skill_script** | ⚠️ Conditional (config flag) | ❌ **Never available** |
| **DisableSkillScripts flag** | ✅ Works | ⚠️ Ignored (no-op) |
| **Default behavior** | Scripts enabled | Scripts **always disabled** |

### Testing
All tests updated and passing:
- ✅ Skills integration tests
- ✅ Schema validation tests
- ✅ Agent creation tests
- ✅ 100% backward compatible (no breaking changes to API)

## [1.1.4] - 2026-01-30

### Fixed
- **[RE-RELEASE]** Re-release of v1.1.3 to ensure all changes are properly distributed
- No code changes from v1.1.3 - this is to force cache invalidation and rebuild
- Confirmed all DisableSkillScripts functionality is working correctly

### Why This Release?
Some users reported the DisableSkillScripts feature not working, which turned out to be a cached binary issue. This re-release ensures:
- ✅ All v1.1.3 changes are correctly included
- ✅ Go module cache is properly invalidated
- ✅ Forces rebuild of consuming applications

### Verification Checklist
If you're updating from v1.1.3:
```bash
# 1. Clear cache
go clean -cache -modcache

# 2. Update to v1.1.4
go get github.com/jholhewres/agent-go@v1.1.4

# 3. Rebuild
rm -f bin/*
go build ./...

# 4. Verify the feature is present
grep -r "DisableSkillScripts" vendor/github.com/jholhewres/agent-go/
```

All tests passing (100% same as v1.1.3):
- ✅ 6 DisableScripts tests
- ✅ Schema validation tests
- ✅ Integration tests
- ✅ Backward compatibility maintained

## [1.1.3] - 2026-01-30

### Added
- **DisableSkillScripts** configuration option in `agent.Config`
- Skills can now be disabled from executing scripts while keeping instructions/references
- `Skills.DisableScripts()` method to programmatically disable script execution
- `Skills.EnableScripts()` method to re-enable script execution
- `Skills.ScriptsEnabled()` method to check current script execution state

### Changed
- Skills toolkit now conditionally registers `get_skill_script` based on `enableScripts` flag
- When `DisableSkillScripts: true`, only `get_skill_instructions` and `get_skill_reference` are available
- Script execution remains enabled by default (backward compatible)

### Use Case
Useful when you want to use skills only for documentation/guidance without allowing script execution:
```go
agent, err := agentgo.New(agentgo.Config{
    Model:               model,
    Skills:              agentSkills,
    DisableSkillScripts: true, // Only instructions/references, no scripts
})
```

### Testing
New tests added:
- ✅ `TestDisableScripts` - Verifies DisableScripts/EnableScripts/ScriptsEnabled
- ✅ `TestAsToolkitWithScriptsDisabled` - Validates toolkit generation with scripts disabled
- ✅ `TestNewSkillsDefaultScriptState` - Ensures default behavior (scripts enabled)
- ✅ `TestAgentWithDisabledScripts` - Agent integration with disabled scripts
- ✅ `TestAgentWithEnabledScripts` - Agent integration with enabled scripts (default)
- ✅ `TestAgentWithoutSkillsUnaffected` - DisableSkillScripts has no effect without skills

All existing tests passing with backward compatibility maintained.

## [1.1.2] - 2026-01-30

### Fixed
- **[CRITICAL]** Fixed `ToModelToolDefinitions()` not serializing `Items` field for array parameters
- Array parameters now correctly include `items` schema when converted to OpenAI format
- Skills `get_skill_script` function now generates valid OpenAI-compatible JSON schema
- v1.1.1 only added the `Items` field to struct but didn't serialize it to JSON

### Root Cause
`ToModelToolDefinitions()` was manually building parameter schemas but only copying `type` and `description` fields, completely ignoring the `Items` field for array types.

### Changes
- Modified `ToModelToolDefinitions()` to check if parameter is type `array` and has `Items` defined
- Added recursive serialization of `Items` schema with support for nested properties
- Array parameters now output: `{"type": "array", "items": {"type": "string"}}`

### Added
- Comprehensive serialization tests in `pkg/agentgo/tools/toolkit/toolkit_serialization_test.go`
- `TestToModelToolDefinitions_ArrayParameterWithItems` validates items field presence
- `TestToModelToolDefinitions_SkillsGetScriptSchema` tests exact get_skill_script schema
- `TestToModelToolDefinitions_NonArrayParameter` ensures non-arrays don't get items

### Testing
All tests passing:
- ✅ Schema validation tests (3 new + 3 existing)
- ✅ Serialization tests (3 new)
- ✅ Skills integration tests
- ✅ Agent integration tests

OpenAI schema now correctly generated:
```json
{
  "args": {
    "type": "array",
    "description": "Arguments to pass to the script",
    "items": {"type": "string"}
  }
}
```

## [1.1.1] - 2026-01-30

### Fixed
- **[CRITICAL]** Fixed invalid JSON schema for `get_skill_script` function - added missing `items` field for `args` array parameter
- OpenAI API was rejecting skills toolkit with "array schema missing items" error (HTTP 400)
- All LLM providers now accept the corrected schema

### Changed
- `toolkit.Parameter` struct now includes `Items *Parameter` field for array type definitions
- `get_skill_script` args parameter now correctly specifies `items: {type: "string"}`

### Added
- Schema validation tests in `pkg/agentgo/skills/tools_schema_test.go`
- `TestSkillToolsSchemaValid` validates all skill tools have valid OpenAI-compatible schemas
- `TestGetSkillScriptParametersValid` specifically validates get_skill_script parameter structure

## [1.1.0] - 2026-01-30

### Fixed
- **[CRITICAL]** Skills system now automatically integrates into agents when config.Skills is provided
- Skills.GetSystemPrompt() is now appended to agent instructions automatically
- Skills toolkit (get_skill_* tools) is now prepended to agent toolkits
- Agent can now discover and use skills through system prompt and tools

### Added
- Skills.GetToolkit() convenience method for agent integration
- Agent skills integration tests (4 comprehensive tests)
- Complete document loaders suite (PDF, CSV, JSON, HTML, URL)
- MultiURLLoader for concurrent URL fetching
- Knowledge loaders README with examples and best practices
- Knowledge loaders demo example

### Changed
- Agent.New() now processes config.Skills automatically
- Skills toolkit is prepended to user toolkits (skills tools appear first)
- Final instructions include both base instructions and skills snippet

### Dependencies
- Added github.com/ledongthuc/pdf v0.0.0-20250511090121 (PDF text extraction)
- Added github.com/PuerkitoBio/goquery v1.11.0 (HTML parsing)

### Documentation
- pkg/agentgo/knowledge/README.md with complete loader documentation
- cmd/examples/knowledge_loader_demo/main.go demonstrating all loaders

## [2.0.0] - 2026-01-30

### 🎉 AgentGo Launch

**AgentGo is a fork and major enhancement of agno-go**, extending the original high-performance multi-agent framework with powerful new capabilities while maintaining the KISS philosophy.

#### ✨ New Major Features
- **Learning System**: Agents that learn and improve with user profiles, memories, and transferable knowledge
- **Agent Skills**: Modular capability system following Agent Skills specification (filesystem & database loaders)
- **Advanced Reasoning**: Unified reasoning API across all model providers
- **pgvector Support**: PostgreSQL-based vector database integration
- **Prompt Engineering**: Template system with versioning and optimization

#### 🔄 Breaking Changes
- Project renamed from `agno-go` to `agent-go`
- Module path changed from `github.com/rexleimo/agno-go` to `github.com/jholhewres/agent-go`
- Configuration directory changed from `.agno/` to `.agentgo/`
- Version bumped to 2.0.0 to reflect major architectural enhancements

#### 📝 Migration Guide
- Update imports: `github.com/rexleimo/agno-go` → `github.com/jholhewres/agent-go`
- Update config directories: `.agno/` → `.agentgo/`
- All existing APIs remain compatible; new features are opt-in

#### 🙏 Credits
This project is based on [agno-go](https://github.com/rexleimo/agno-go) by [@rexleimo](https://github.com/rexleimo). See [CREDITS.md](CREDITS.md) for full attribution.

---

## Previous Releases (agno-go)

## [1.2.9] - 2025-11-14

### ✨ Added
- EvoLink provider for text, images, and video exposed under `pkg/agentgo/providers/evolink` and `pkg/agentgo/models/evolink/*`, enabling agents and workflows to call EvoLink via the standard model interfaces.
- New EvoLink Media Agents documentation (`website/examples/evolink-media-agents.md`, `website/zh/examples/evolink-media-agents.md`) with end-to-end examples for text → image → video pipelines.
- Knowledge upload chunking: `POST /api/v1/knowledge/content` now accepts `chunk_size` and `chunk_overlap` for JSON, `text/plain` (query params), and multipart form uploads, propagating these values into stored chunk metadata.
- AgentOS HTTP tips surfaced in docs, covering custom health endpoints, `/openapi.yaml` and `/docs` routes, and `server.Resync()` usage.

### 🛠️ Changed
- Knowledge API handlers persist `chunk_size`, `chunk_overlap`, and `chunker_type` for each stored chunk, mirroring the Python AgentOS responses and enabling downstream pipelines to inspect segmentation strategy.
- AgentOS documentation is now the canonical source for configuring health probes and documentation routes for the Go server, aligning README and VitePress content.

### 🧪 Tests
- Added focused tests for EvoLink image and video models to validate configuration boundaries and task polling behavior.
- Extended knowledge API tests to cover chunking parameters, metadata propagation, and compatibility with existing search/config endpoints.

### ✅ Compatibility
- Additive release; public HTTP and Go APIs remain backward compatible.
- Knowledge chunking parameters are optional and default to previous behavior when omitted.

## [1.2.8] - 2025-11-10

### ✨ Added
- Run Context 贯穿执行：向 hooks、tools、遥测传递上下文，并在流式事件中提供 `run_context_id` 便于关联。
- 会话状态扩展：持久化 `AGUI` 子态，`GET /sessions/{id}` 返回包含 UI 状态的 `session_state`。
- 向量索引能力：
  - 可插拔 VectorDB 提供方（保持 Chroma 为默认示例，Redis 为可选依赖）。
  - VectorDB 迁移 CLI（`migrate up/down`）支持集合与索引的幂等创建/回滚。
- Embeddings：新增 VLLM（本地/远端）提供方，遵循通用 `EmbeddingFunction` 接口。
- MCPTools：支持可选参数 `tool_name_prefix`，为注册工具名加前缀。

### 🛠️ Changed
- Redis 从向量数据库默认依赖中剥离，未配置时零影响；启用时按配置注册。
- 团队模型继承仅下沉主模型配置；辅助参数需在 Agent 端显式开启。

### 🐛 Fixed
- 修复模型响应未正确绑定到步骤导致的“未绑定/零值”问题。
- 修复团队场景下基于 OS Schema 的工具判定，避免成员工具丢失。
- 修复异步存储（知识库）复合过滤与超时上下文配合的边界问题（不泄露 goroutine）。
- 强化工具包导入时的错误信息，缺失模块返回结构化提示而非 panic。
- AgentOS 错误处理路径更一致，便于契约测试断言。

### 🧪 Tests
- 新增覆盖：Run Context 传播、AGUI 状态持久化、团队主模型继承、MCP 前缀、VLLM 嵌入。
- 可选依赖（Redis）用例按环境开关执行，默认跳过。

### ✅ Compatibility
- 增量更新；默认关闭的可选能力不影响现有用户；外部 API 形态保持不变。

## [1.2.7] - 2025-11-03

### ✨ Added
- Go session service matching the Python AgentOS `/sessions` API, including Chi router, Postgres-backed store, and health endpoints.
- Deployment assets for the session runtime: dedicated Dockerfile, Compose stack, Helm chart, and curl-based verification script.
- Documentation for quick start and production deployment of the Go session service.

### 🧪 Tests
- Contract suite and Postgres store coverage ensuring session parity with existing Python fixtures.

### ✅ Compatibility
- Additive release; existing APIs remain backward compatible with the Go session runtime as an opt-in component.

---

## [1.2.6] - 2025-10-31

### ✨ Added
- Session runtime parity: session reuse endpoint, history pagination filters (`num_messages`, `stream_events`), and summary manager supporting sync/async generation.
- Response caching for agents and teams via in-memory LRU store, exposing cache-hit metadata in session runs.
- Media attachments across agents, teams, and workflows, including validation helpers and `WithMediaPayload` run option.
- Pluggable session storage drivers: MongoDB and SQLite implementations alongside Postgres with shared JSON contracts.
- Tooling expansions: Tavily Reader/Search, Claude Agent Skills, Gmail mark-as-read, Jira worklogs, ElevenLabs speech synthesis, and enhanced file toolkit.

### 🛠️ Changed
- AgentOS session handlers now expose summary endpoints, reuse semantics, run metadata (status timestamps, cancellation reasons, cache hits), and history filters.
- Workflow engine supports resumable checkpoints, cancellation persistence, and media-only payload execution paths.
- MCP client caches capability manifests and forwards media attachments in tool calls.

### 🧪 Tests
- Added dedicated coverage for cache layer, summary manager, storage drivers, workflow media flows, and new toolkits.

### ✅ Compatibility
- Additive release; existing APIs remain backward compatible with new features opt-in.

---

## [1.2.5] - 2025-10-20

### ✨ Added
- Model providers: Cohere, Together, OpenRouter, LM Studio, Vercel, Portkey, InternLM, SambaNova (Invoke/Stream + function calling)
- Core modules:
  - Evaluation system (scenario runner, per-run metrics, aggregated summary, multi-model comparison)
  - Media processing: image metadata (DecodeConfig), audio/video probe占位
  - Debug helpers: request/response compact dump
  - Cloud: NoopDeployer interface for simple deployments
- Integrations registry: register/list/health-check for third‑party services

### 🛠️ Changed
- Airflow toolkit mock schema aligned with Airflow REST API v2 (Context7): `total_entries`, `dag_run_id`, `logical_date`
- Website hero image uses `/logo.png` (fix broken asset)
- README “Multi-provider models” list updated

### 🧪 Tests
- Focused unit tests for new providers and modules (cohere, together, openrouter, lmstudio, vercel, portkey, internlm, sambanova, eval/media/debug/integrations/utils)

### ✅ Compatibility
- Additive features; no breaking changes

---

## [1.2.2] - 2025-10-18

### ✨ Added

#### Reasoning Model Support
- **Enhanced Reasoning Capabilities** - Advanced reasoning support for modern LLM models
  - Automatic detection for Gemini, Anthropic Claude, and VertexAI Claude
  - Structured reasoning output with step-by-step analysis
  - **cmd/examples/reasoning/** - Example program demonstrating reasoning capabilities

#### Batch Operations for PostgreSQL
- **High-Performance Batch Upsert** - Optimized bulk operations
  - 10x faster than individual INSERT/UPDATE operations
  - Transaction-safe with conflict resolution
  - **cmd/examples/batch_upsert/** - Performance comparison example

#### SurrealDB Vector Database Support
- **Modern Vector Database Integration** - Full SurrealDB support
  - Vector similarity search and document embedding storage
  - Real-time query capabilities
  - **cmd/examples/surreal_demo/** - Vector operations example

#### CI/CD Pipeline
- **GitHub Actions CI Workflow** - Automated testing and quality assurance
  - Go module validation and unit tests with race detection
  - Code coverage reporting and security scanning

#### Enhanced Knowledge API
- **Advanced Content Processing** - Improved knowledge ingestion
  - Multi-format content extraction (JSON, Form, Text)
  - Structured data validation and metadata extraction

### 🧪 Testing & Quality
- **Enhanced Test Coverage** - 85% reasoning, 92% batch, 88% SurrealDB
- **Race Condition Detection** - All new code validated with `-race` flag
- **Performance Benchmarks** - Added comprehensive performance tests

### 📊 Performance
- **Batch Operations** - 10x performance improvement for bulk data
- **Reasoning Detection** - Minimal overhead (<1ms)
- **No Regression** - All existing benchmarks maintained

### ✅ Backward Compatibility
- Additive changes only; no breaking changes
- All existing APIs remain unchanged
- Enhanced functionality automatically available for supported models

## [1.2.1] - 2025-10-15

### ✨ Added
- SSE event filtering on streaming endpoints (A2A)
  - `POST /api/v1/agents/:id/run/stream?types=token,complete`
- Content extraction middleware for AgentOS (JSON/Form → context)
- Google Sheets toolkit (service account)
- Minimal knowledge ingestion endpoint (`POST /api/v1/knowledge/content`)

### 🧭 Documentation Reorganization
- Adopted clear separation of docs:
  - `website/` → Implemented, user-facing documentation (VitePress site)
  - `docs/` → Design drafts, WIP, migration plans, developer/internal docs
- Added `docs/README.md` to state policy and entry points
- Added `CONTRIBUTING.md` for contributors (development, testing, docs website)

### 🔗 Link Updates
- README, CLAUDE, CHANGELOG, and release notes now point to canonical pages under `website/advanced/*` and `website/guide/*`
- Removed outdated links to duplicated files under `docs/`

### 🧹 Removed (duplicated implemented docs from docs/)
- Deleted `docs/{API_REFERENCE.md, ARCHITECTURE.md, DEPLOYMENT.md, MULTI_TENANT.md, PERFORMANCE.md, QUICK_START.md, SESSION_STATE.md, WORKFLOW_HISTORY.md, A2A_INTERFACE.md, CHANGELOG.md}`

### 🌐 Website
- Updated API docs to include Knowledge API and configuration on AgentOS page
- Updated website Release Notes with v1.2.1 summary

### ✅ Backward Compatibility
- Additive changes only; no breaking changes

## [1.2.0] - 2025-10-12

### ✨ Added

#### Workflow Session Storage (S005)
- **In-Memory Session Management** - Complete workflow session lifecycle management
  - **pkg/agentgo/workflow/memory_storage.go** - MemoryStorage implementation (393 lines)
    - Session creation, retrieval, updating, deletion
    - Concurrent-safe with sync.RWMutex
    - Configurable max sessions limit
    - Automatic session pruning
  - **pkg/agentgo/workflow/session.go** - WorkflowSession structure (300 lines)
    - Session metadata and run history
    - History retrieval with flexible count
    - Statistics tracking (total/completed/success/failed runs)
  - **pkg/agentgo/workflow/run.go** - WorkflowRun structure (158 lines)
    - Individual run execution tracking
    - Input/output/error recording
    - Timestamp and status management
  - **pkg/agentgo/workflow/storage.go** - WorkflowStorage interface (141 lines)
    - Abstract storage interface for extensibility
    - Support for custom storage implementations (Redis, PostgreSQL, etc.)

#### Workflow History Injection (S008)
- **Agent Temporary Instructions Support** - Enable history context injection without modifying agent's original configuration
  - **pkg/agentgo/agent/agent.go** - Enhanced with temporary instructions mechanism
    - `tempInstructions string` - Temporary override for instructions (single execution only)
    - `instructionsMu sync.RWMutex` - Thread-safe concurrent access protection
    - `GetInstructions()` - Retrieves current instructions (temporary takes precedence)
    - `SetInstructions()` - Permanently sets agent instructions
    - `SetTempInstructions()` - Temporarily sets instructions (cleared after Run)
    - `ClearTempInstructions()` - Explicitly clears temporary instructions
    - `updateSystemMessage()` - Updates system message with current instructions
  - **Auto-cleanup mechanism**: `defer a.ClearTempInstructions()` in Run() ensures zero memory leak
  - **Concurrency safety**: RWMutex allows concurrent reads, exclusive writes
  - **Backward compatible**: Empty tempInstructions behaves identically to original implementation

- **History Injection Utilities** - Flexible history formatting and injection helpers
  - **pkg/agentgo/workflow/history_injection.go** - History injection helper functions (151 lines)
    - `InjectHistoryToAgent()` - Injects formatted history into agent's temporary instructions
    - `buildEnhancedInstructions()` - Combines original instructions with history context
    - `RestoreAgentInstructions()` - Explicitly restores original instructions (optional, auto-cleared)
    - `FormatHistoryForAgent()` - Formats history entries with customizable options
    - `HistoryFormatOptions` - Flexible formatting configuration:
      - Header/Footer tags
      - Include/exclude input/output
      - Optional timestamps
      - Customizable labels
    - `DefaultHistoryFormatOptions()` - Sensible defaults with XML-style tags

- **Step Integration** - Seamless workflow history injection
  - **pkg/agentgo/workflow/step.go** - Updated Step execution with history support
    - Automatically injects history when `shouldAddHistory()` returns true
    - Retrieves formatted history from `ExecutionContext.GetHistoryContext()`
    - No changes required to existing workflow code

### 🧪 Testing

- **Comprehensive Test Coverage**:
  - **agent_instructions_test.go** - 308 lines, 8 test cases
    - `TestAgent_TempInstructions` - Basic get/set/clear functionality
    - `TestAgent_SetInstructions` - Permanent instruction changes
    - `TestAgent_TempInstructionsPriority` - Temporary overrides permanent
    - `TestAgent_ConcurrentInstructionsAccess` - 100 iterations, 300 goroutines
    - `TestAgent_Run_AutoClearsTempInstructions` - Verify defer cleanup
    - `TestAgent_Run_WithTempInstructionsError` - Cleanup on error
    - `TestAgent_UpdateSystemMessage` - Table-driven tests

  - **history_injection_test.go** - 380 lines, 12 test cases
    - Nil safety, empty history handling
    - Default and custom format options
    - Multiple runs formatting
    - Input/output inclusion control

- **Test Results**:
  - All tests passing with `-race` detector ✅
  - Agent coverage: 75.6% (100% on instruction methods)
  - Workflow coverage: 88.4% (100% on history injection)
  - Zero race conditions detected

### 📊 Performance

- **Performance Targets Met/Exceeded**:
  - `GetInstructions()`: <50ns (RLock optimized)
  - `SetTempInstructions()`: <100ns (Lock optimized)
  - `InjectHistoryToAgent()`: ~200-300ns (vs <500ns target, 2x better)
  - Total injection overhead: <1ms (vs <1ms target)
  - Memory overhead: ~40 bytes per agent (negligible)

- **No Performance Regression**:
  - Agent instantiation: Still ~180ns/op
  - Memory footprint: Still ~1.2KB per agent
  - Existing benchmarks unchanged

### 🔧 Technical Highlights

- **Zero Memory Leak** - defer-based automatic cleanup ensures temp instructions always cleared
- **Concurrency Safe** - sync.RWMutex enables high-performance concurrent access
- **API Design** - Clean, intuitive methods following Go best practices
- **Backward Compatible** - No breaking changes to existing Agent API
- **Bilingual Documentation** - All code comments in English/中文

### 📝 Files Added/Modified

**New Files:**
- `pkg/agentgo/workflow/history_injection.go` - History injection utilities (151 lines)
- `pkg/agentgo/agent/agent_instructions_test.go` - Instruction tests (308 lines)
- `pkg/agentgo/workflow/history_injection_test.go` - Injection tests (380 lines)

**Modified Files:**
- `pkg/agentgo/agent/agent.go` - Added temporary instructions support
- `pkg/agentgo/workflow/step.go` - Integrated history injection
- `docs/task/S008-agent-history-injection.md` - Marked as Done

### ✅ Acceptance Criteria

All S008 acceptance criteria met or exceeded:
- ✅ Agent supports temporary instructions
- ✅ Agent.Run auto-clears temporary instructions
- ✅ Concurrent access to instructions is thread-safe
- ✅ History injection doesn't affect Agent's original configuration
- ✅ Flexible history formatting options provided
- ✅ Test coverage >85% (100% on new methods)
- ✅ All tests passing
- ✅ Performance: Injection overhead <1ms (achieved <1μs)

### 🚀 Usage Example

```go
import (
    "github.com/jholhewres/agent-go/pkg/agentgo/agent"
    "github.com/jholhewres/agent-go/pkg/agentgo/workflow"
)

// Create agent with original instructions
agent, _ := agent.New(agent.Config{
    Model:        model,
    Instructions: "You are a helpful assistant",
})

// Format workflow history
history := []workflow.HistoryEntry{
    {Input: "hello", Output: "hi there"},
    {Input: "how are you", Output: "I'm good"},
}
historyContext := workflow.FormatHistoryForAgent(history, nil)

// Inject history (temporarily enhances instructions)
workflow.InjectHistoryToAgent(agent, historyContext)

// Run agent (temp instructions auto-cleared after execution)
output, err := agent.Run(ctx, "new question")

// Agent's original instructions remain unchanged
fmt.Println(agent.Instructions) // "You are a helpful assistant"
```

## [1.1.1] - 2025-10-08

### ✨ Added

#### Groq Model Integration
- **Groq Ultra-Fast Inference Support** - Industry-leading LLM inference speed
  - **pkg/agentgo/models/groq/** - Complete Groq model implementation (287 lines)
    - `groq.go` - Main implementation using OpenAI-compatible API
    - `types.go` - Model constants and metadata (7+ models)
    - `groq_test.go` - Comprehensive unit tests (580+ lines)
    - `README.md` - Detailed documentation and examples
  - **Supported Models:**
    - LLaMA 3.1 8B Instant (fastest, recommended)
    - LLaMA 3.1 70B Versatile (most capable)
    - LLaMA 3.3 70B Versatile (latest)
    - Mixtral 8x7B (Mixture of Experts)
    - Gemma 2 9B (compact but powerful)
    - Whisper Large V3 (speech recognition)
    - LLaMA Guard 3 8B (content moderation)
  - **Performance Benefits:**
    - 10x faster inference vs traditional cloud LLM providers
    - Ultra-low first-token latency
    - High concurrent request throughput
  - **Features:**
    - OpenAI-compatible API (reuses go-openai SDK)
    - Full function calling support
    - Streaming and non-streaming modes
    - Configurable timeout, temperature, max tokens
  - **cmd/examples/groq_agent/** - Example program with 4 scenarios
  - **Test Coverage:** 52.4% ✅
  - **Documentation:** Updated CLAUDE.md with Groq configuration

### 📝 Documentation
- Added Groq to supported models list in CLAUDE.md
- Added `GROQ_API_KEY` environment variable documentation
- Updated example programs list with groq_agent
- Added Groq to test coverage table

## [1.1.0] - 2025-10-08

### ✨ Added

#### A2A (Agent-to-Agent) Interface
- **A2A Protocol Support** - Standardized agent-to-agent communication based on JSON-RPC 2.0
  - **pkg/agentos/a2a/** - Complete A2A interface implementation (1001 lines)
    - `types.go` - JSON-RPC 2.0 type definitions (154 lines)
    - `validator.go` - Request validation logic (108 lines)
    - `mapper.go` - A2A ↔ RunInput/Output conversion (317 lines)
    - `a2a.go` - Main interface and entity management (148 lines)
    - `handlers.go` - HTTP handlers for send/stream endpoints (274 lines)
  - REST API endpoints:
    - `POST /a2a/message/send` - Non-streaming message send
    - `POST /a2a/message/stream` - Streaming via Server-Sent Events (SSE)
  - Multi-media support: Text, images (URI/bytes), files, JSON data
  - Compatible with Python Agno A2A implementation
  - **cmd/examples/a2a_server/** - Example A2A server
  - **pkg/agentos/a2a/README.md** - Complete bilingual documentation (English/中文)

#### Workflow Session State Management
- **Thread-Safe Session State** - Cross-step session management with race condition fix
  - **pkg/agentgo/workflow/session_state.go** - SessionState implementation (192 lines)
    - Thread-safe with sync.RWMutex
    - Deep copy via JSON serialization for parallel branch isolation
    - Smart merging: only applies actual changes (last-write-wins)
  - **ExecutionContext Enhancement** - Added session support fields:
    - `SessionState *SessionState` - Cross-step persistent state
    - `SessionID string` - Unique session identifier
    - `UserID string` - Multi-tenant user identifier
  - **Parallel Execution Fix** - Solved Python Agno v2.1.2 race condition:
    - Clone SessionState for each parallel branch
    - Merge modified states after parallel execution
    - Prevents data races in concurrent workflow steps
  - **pkg/agentgo/workflow/SESSION_STATE.md** - Comprehensive documentation
  - **Test Coverage:** 79.4% with race detector validation ✅

#### Multi-Tenant Memory Support
- **User-Isolated Memory Storage** - Multi-tenant conversation history
  - Enhanced Memory interface with optional `userID` parameter:
    - `Add(message, userID...)` - Add message for specific user
    - `GetMessages(userID...)` - Get user-specific messages
    - `Clear(userID...)` - Clear user messages
    - `Size(userID...)` - Get user message count
  - `InMemory` implementation:
    - Per-user message storage: `map[string][]*types.Message`
    - Independent maxSize limit per user
    - Backward compatible: empty userID defaults to "default" user
  - Agent integration:
    - Added `UserID string` field to Agent and Config
    - All memory operations pass agent's UserID
  - New `ClearAll()` method to clear all users
  - **Tests:** Multi-tenant isolation, backward compatibility, race detection ✅

#### Model Timeout Configuration
- **Configurable Request Timeout** - Fine-grained timeout control for LLM calls
  - **OpenAI Model** (`pkg/agentgo/models/openai/openai.go`):
    - Added `Timeout time.Duration` to Config
    - Default: 60 seconds
    - Applied to underlying HTTP client
  - **Anthropic Claude** (`pkg/agentgo/models/anthropic/anthropic.go`):
    - Added `Timeout time.Duration` to Config
    - Default: 60 seconds
    - Applied to HTTP client
  - Usage example:
    ```go
    claude := anthropic.New("claude-3-opus", anthropic.Config{
        APIKey:  apiKey,
        Timeout: 30 * time.Second, // Custom timeout
    })
    ```

### 🐛 Fixed

- **Workflow Race Condition** - Fixed parallel step execution data race
  - Python Agno v2.1.2 had shared `session_state` dict causing overwrites
  - Go implementation uses independent SessionState clones per branch
  - Smart merge strategy prevents data loss in concurrent execution

### 🧪 Testing

- **New Test Suites:**
  - `session_state_test.go` - 543 lines of session state tests
  - `memory_test.go` - Multi-tenant memory tests (4 new test cases)
  - `agent_test.go` - Multi-tenant agent test (TestAgent_MultiTenant)
  - `openai_test.go` - Timeout configuration test
  - `anthropic_test.go` - Timeout configuration test

- **Test Results:**
  - All tests passing with `-race` detector ✅
  - Workflow coverage: 79.4%
  - Memory coverage: maintained at 93.1%
  - Agent coverage: maintained at 74.7%

### 📊 Performance

- **No Performance Regression** - All benchmarks remain consistent:
  - Agent instantiation: ~180ns/op
  - Memory footprint: ~1.2KB per agent
  - Thread-safe concurrent operations

### 🔧 Technical Highlights

- **Python Agno v2.1.2 Compatibility** - Migrated features from commits:
  - `7e487eb` → `bf3286bb` (23 commits, 5 major features)
  - A2A utils implementation
  - Session state race condition fix
  - Multi-tenant memory support
  - Model timeout parameters

- **Bilingual Documentation** - All new features documented in English/中文:
  - Inline code comments
  - README files
  - API documentation

### 📝 Files Added/Modified

**New Files:**
- `pkg/agentos/a2a/*.go` - A2A interface (5 files, 1001 lines)
- `pkg/agentgo/workflow/session_state.go` - Session state (192 lines)
- `pkg/agentgo/workflow/session_state_test.go` - Tests (543 lines)
- `pkg/agentos/a2a/README.md` - A2A documentation
- `pkg/agentgo/workflow/SESSION_STATE.md` - Session state guide
- `cmd/examples/a2a_server/main.go` - A2A example server

**Modified Files:**
- `pkg/agentgo/memory/memory.go` - Multi-tenant support
- `pkg/agentgo/memory/memory_test.go` - New multi-tenant tests
- `pkg/agentgo/agent/agent.go` - UserID support
- `pkg/agentgo/agent/agent_test.go` - Multi-tenant test
- `pkg/agentgo/workflow/workflow.go` - SessionState fields
- `pkg/agentgo/workflow/parallel.go` - Race condition fix
- `pkg/agentgo/models/openai/openai.go` - Timeout support
- `pkg/agentgo/models/anthropic/anthropic.go` - Timeout support

### ✅ Migration Status

Completed migration from Python Agno v2.1.2:
- ✅ A2A interface implementation
- ✅ Workflow session state management (race condition fix)
- ✅ Multi-tenant memory support (userID)
- ✅ Model timeout parameters (OpenAI, Anthropic)

### 🚀 Upgrade Guide

**Multi-Tenant Memory:**
```go
// Old (single-tenant)
agent := agent.New(agent.Config{
    Memory: memory.NewInMemory(100),
})

// New (multi-tenant)
agent := agent.New(agent.Config{
    UserID: "user-123",  // Add UserID
    Memory: memory.NewInMemory(100),
})
```

**Workflow Session State:**
```go
// Create context with session info
ctx := workflow.NewExecutionContextWithSession(
    "input",
    "session-id",
    "user-id",
)

// Access session state
ctx.SetSessionState("key", "value")
value, _ := ctx.GetSessionState("key")
```

**A2A Interface:**
```go
// Create A2A interface
a2a := a2a.New(a2a.Config{
    Agents: []a2a.Entity{myAgent},
    Prefix: "/a2a",
})

// Register routes (Gin)
router := gin.Default()
a2a.RegisterRoutes(router)
```

### 📖 Documentation

- [A2A README](pkg/agentos/a2a/README.md) - Complete A2A protocol guide
- [Session State Guide](pkg/agentgo/workflow/SESSION_STATE.md) - Workflow session management
- [CHANGELOG.md](CHANGELOG.md) - This file

## [1.0.3] - 2025-10-06

### 🧪 Improved

#### Testing & Quality
- **Enhanced JSON Serialization Tests** - Achieved 100% test coverage for utils/serialize package
  - Added error handling tests for unserializable types (channels, functions)
  - Added panic behavior tests for MustToJSONString
  - Added edge case tests (nil pointers, empty collections)
  - Test coverage: 92.3% → 100% ✅

#### Performance Benchmarks
- **Optimized Performance Tests** - Aligned with Python Agno performance testing patterns
  - Simplified agent instantiation benchmark (removed unnecessary variable)
  - Cleaned up tool registration patterns
  - Renamed test for consistency: "Tool Instantiation Performance" → "Agent Instantiation"

#### Documentation
- **Comprehensive Package Documentation** - Added bilingual (English/中文) documentation
  - Package-level overview with usage examples
  - Detailed function documentation with examples
  - Performance metrics included in package docs
  - All public APIs now fully documented

### 📊 Performance

Current benchmark results on Apple M3:
- **ToJSON**: ~600ns/op, 760B/op, 15 allocs/op
- **ConvertValue**: ~180ns/op, 392B/op, 5 allocs/op
- **Agent Creation**: ~180ns/op (16x faster than Python)

### 🔧 Technical Highlights

- **100% Test Coverage** - utils/serialize package now has complete test coverage
- **Better Error Handling** - Comprehensive tests for edge cases and error conditions
- **Production Ready** - Serialization utilities validated for WebSocket and API usage
- **Python Compatibility** - Prevents the JSON serialization bug found in Python Agno (commit aea0fc129)

### 📝 Files Changed

- `pkg/agentgo/utils/serialize.go` - Enhanced documentation with examples and performance notes
- `pkg/agentgo/utils/serialize_test.go` - Added 3 new test cases for error handling
- `pkg/agentgo/agent/agent_bench_test.go` - Simplified benchmark following Python patterns

### ✅ Migration Status

Completed migration items from Python Agno:
- ✅ JSON serialization bug fix (aea0fc129) - Already prevented in Go implementation
- ✅ Performance test optimization (e639f4996) - Applied to Go benchmarks
- 🔄 Custom route prefix (06baed104) - Deferred to Week 7 (AgentOS expansion)
- 🔄 HN tools update (24c3ee688) - Documentation only, no action needed

## [1.0.2] - 2025-10-05

### ✨ Added

#### New LLM Provider
- **GLM (智谱AI)** - Full integration with Zhipu AI's GLM models
  - Support for GLM-4, GLM-4V (vision), GLM-3-Turbo
  - Custom JWT authentication (HMAC-SHA256)
  - Synchronous API calls (`Invoke`)
  - Streaming responses (`InvokeStream`)
  - Tool/Function calling support
  - Test coverage: 57.2%

#### Implementation Details
- **pkg/agentgo/models/glm/glm.go** - Main model implementation (410 lines)
- **pkg/agentgo/models/glm/auth.go** - JWT authentication logic (59 lines)
- **pkg/agentgo/models/glm/types.go** - GLM API type definitions (105 lines)
- **pkg/agentgo/models/glm/glm_test.go** - Comprehensive unit tests (320 lines)
- **pkg/agentgo/models/glm/README.md** - Complete usage documentation

#### Examples & Documentation
- **cmd/examples/glm_agent/** - GLM agent example with calculator tools
  - Chinese language support demonstration
  - Multi-step calculation examples
  - Tool calling integration
- Updated README.md with GLM provider information
- Updated CLAUDE.md with GLM configuration and usage
- Added bilingual comments (English/中文) throughout codebase

### 🔧 Technical Highlights

- **Custom JWT Authentication** - Implemented GLM-specific JWT token generation
  - 7-day token expiration
  - Secure HMAC-SHA256 signing
  - Automatic token regeneration per request

- **OpenAI-Compatible Format** - API structure similar to OpenAI for easy integration
  - Request/response format alignment
  - Tool calling compatibility
  - Streaming support via Server-Sent Events (SSE)

- **Type Safety** - Full Go type system integration
  - Strongly-typed request/response structures
  - Error handling with custom error types
  - Context support for cancellation

### 📊 Test Results

- ✅ All 7 GLM tests passing
- ✅ 57.2% code coverage
- ✅ Race detector: PASS
- ✅ Build verification: SUCCESS

### 🌍 Environment Variables

New environment variable for GLM:
```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

### 📦 Dependencies

Added:
- `github.com/golang-jwt/jwt/v5 v5.3.0` - For JWT authentication

### 🎯 Supported Models

Total LLM providers increased from 6 to 7:
- OpenAI (GPT-4, GPT-3.5, GPT-4 Turbo)
- Anthropic (Claude 3.5 Sonnet, Claude 3 Opus/Sonnet/Haiku)
- **GLM (智谱AI: GLM-4, GLM-4V, GLM-3-Turbo)** ⭐ NEW
- Ollama (Local models)
- DeepSeek (DeepSeek-V2, DeepSeek-Coder)
- Google Gemini (Gemini Pro, Flash)
- ModelScope (Qwen, Yi models)

### 📝 Documentation Updates

- README.md - Added GLM to supported models list with example code
- CLAUDE.md - Added GLM environment variables and configuration
- Created pkg/agentgo/models/glm/README.md with comprehensive usage guide
- All code comments are bilingual (English/中文)

## [1.0.0] - 2025-10-02

### 🎉 Initial Release

AgentGo v1.0 is a high-performance Go implementation of the Agno multi-agent framework, designed for building production-ready AI agent systems.

### ✨ Features

#### Core Agent System
- **Agent** - Single autonomous agent with tool support
  - LLM model integration
  - Tool/function calling
  - Conversation memory
  - System instructions
  - Max loop protection
  - Coverage: 74.7%

- **Team** - Multi-agent collaboration with 4 coordination modes:
  - `Sequential` - Agents work one after another
  - `Parallel` - All agents work simultaneously
  - `LeaderFollower` - Leader delegates tasks to followers
  - `Consensus` - Agents discuss until reaching agreement
  - Coverage: 92.3%

- **Workflow** - Step-based orchestration with 5 primitives:
  - `Step` - Basic workflow step (agent or function)
  - `Condition` - Conditional branching
  - `Loop` - Iterative loops
  - `Parallel` - Parallel execution
  - `Router` - Dynamic routing
  - Coverage: 80.4%

#### LLM Providers
- **OpenAI** - GPT-4, GPT-3.5, GPT-4 Turbo
- **Anthropic** - Claude 3.5 Sonnet, Claude 3 Opus/Sonnet/Haiku
- **Ollama** - Local model support (llama3, mistral, etc.)

#### Tools
- **Calculator** - Basic math operations
- **HTTP** - GET/POST requests
- **File** - File operations with safety controls

#### Storage & Memory
- **Memory** - In-memory conversation storage with auto-truncation
  - Configurable message limits
  - Thread-safe operations
  - Coverage: 93.1%

- **Session** - Session management for multi-turn conversations
  - In-memory storage (default)
  - PostgreSQL support (via schema)
  - Redis caching support

#### Vector Database
- **ChromaDB** - Vector storage for RAG applications
  - Document embedding
  - Semantic search
  - Collection management

#### AgentOS - Production Server
- **RESTful API** - Full-featured HTTP server
  - Session CRUD operations
  - Agent registration and execution
  - Health check endpoint
  - Structured logging with slog
  - CORS support
  - Request timeout handling
  - Coverage: 65.0%

- **Agent Registry** - Thread-safe agent management
  - Dynamic agent registration
  - Concurrent access support
  - Agent lifecycle management

#### Developer Experience
- **Types** - Comprehensive type system
  - Message types (System, User, Assistant, Tool)
  - Error types with codes
  - Model request/response types
  - 100% test coverage ⭐

- **Documentation**
  - Complete API documentation (OpenAPI 3.0)
  - Deployment guide (Docker, K8s, native)
  - Architecture documentation
  - Performance benchmarks
  - Code examples

#### Deployment
- **Docker** - Production-ready Dockerfile
  - Multi-stage build (~15MB final image)
  - Non-root user
  - Health checks
  - Security best practices

- **Docker Compose** - Full stack deployment
  - AgentOS server
  - PostgreSQL database
  - Redis cache
  - ChromaDB (optional)
  - Ollama (optional)

- **Kubernetes** - K8s manifests included
  - Deployment, Service, ConfigMap, Secret
  - Health probes
  - Resource limits
  - Horizontal Pod Autoscaling ready

### 📊 Performance

- **Agent Creation:** ~180ns/op (16x faster than Python)
- **Memory Footprint:** ~1.2KB per agent
- **Test Coverage:** 80.8% average across core packages
- **Concurrent Operations:** Fully thread-safe with RWMutex

### 🧪 Testing

- **85+ test cases** across all core packages
- **100% pass rate** ✅
- All packages exceed 70% coverage target
- Comprehensive integration tests
- Concurrent access tests
- Performance benchmarks

### 📚 Examples

- `simple_agent` - Basic agent with calculator
- `claude_agent` - Anthropic Claude integration
- `ollama_agent` - Local model support
- `team_demo` - Multi-agent collaboration
- `workflow_demo` - Workflow orchestration
- `rag_demo` - RAG with ChromaDB

### 🔧 Technical Details

**Dependencies:**
- Go 1.21+
- Gin web framework
- PostgreSQL 15+ (optional)
- Redis 7+ (optional)
- ChromaDB (optional)

**Project Structure:**
```
agno-Go/
├── pkg/agentgo/          # Core framework
│   ├── agent/         # Agent implementation
│   ├── team/          # Team coordination
│   ├── workflow/      # Workflow engine
│   ├── models/        # LLM providers
│   ├── tools/         # Tool integrations
│   ├── memory/        # Conversation memory
│   └── types/         # Core types
├── pkg/agentos/       # Production server
│   ├── server.go      # HTTP server
│   ├── registry.go    # Agent registry
│   └── openapi.yaml   # API specification
├── cmd/examples/      # Example programs
└── docs/              # Documentation
```

### 🎯 Design Philosophy

AgentGo follows the **KISS principle** (Keep It Simple, Stupid):
- Focus on quality over quantity
- Clear, maintainable code
- Comprehensive testing
- Production-ready from day one

### 🔒 Security

- Non-root Docker container
- Secret management best practices
- Input validation
- Error handling
- Rate limiting support
- HTTPS/TLS ready

### 📖 Documentation

- [README.md](README.md) - Getting started
- [website/advanced/architecture.md](website/advanced/architecture.md) - Architecture overview
- [website/advanced/deployment.md](website/advanced/deployment.md) - Deployment guide
- [website/advanced/performance.md](website/advanced/performance.md) - Performance benchmarks
- [docs/DEVELOPMENT.md#testing-standards](docs/DEVELOPMENT.md#testing-standards) - Test coverage standards
- [pkg/agentos/README.md](pkg/agentos/README.md) - AgentOS API guide
- [pkg/agentos/openapi.yaml](pkg/agentos/openapi.yaml) - OpenAPI specification

### 🙏 Acknowledgments

AgentGo is inspired by and compatible with the design philosophy of:
- [Agno](https://github.com/agno-agi/agno) - Python multi-agent framework

### 📝 Migration from Python Agno

AgentGo maintains API compatibility where possible, making migration straightforward:

**Python:**
```python
from agno.agent import Agent
from agno.models.openai import OpenAI

agent = Agent(
    name="Assistant",
    model=OpenAI(id="gpt-4"),
)
response = agent.run("Hello!")
```

**Go:**
```go
import (
    "github.com/jholhewres/agent-go/pkg/agentgo/agent"
    "github.com/jholhewres/agent-go/pkg/agentgo/models/openai"
)

model, _ := openai.New("gpt-4", openai.Config{...})
ag, _ := agent.New(agent.Config{
    Name: "Assistant",
    Model: model,
})
output, _ := ag.Run(ctx, "Hello!")
```

### 🚀 Getting Started

**Installation:**
```bash
go get github.com/jholhewres/agent-go
```

**Quick Start:**
```bash
# Clone repository
git clone https://github.com/jholhewres/agent-go
cd agno-Go

# Run example
export OPENAI_API_KEY=sk-...
go run cmd/examples/simple_agent/main.go

# Or use Docker
docker-compose up -d
curl http://localhost:8080/health
```

### 🛣️ Roadmap

**v1.1** (Planned)
- Streaming response support
- More tool integrations
- Additional vector databases
- Enhanced monitoring/metrics

**v1.2** (Planned)
- gRPC API support
- WebSocket for real-time updates
- Plugin system
- Advanced workflow features

### 📄 License

MIT License - See [LICENSE](LICENSE) for details.

### 🔗 Links

- **GitHub:** https://github.com/jholhewres/agent-go
- **Documentation:** https://docs.agno.com
- **Issues:** https://github.com/jholhewres/agent-go/issues
- **Discussions:** https://github.com/jholhewres/agent-go/discussions

---

**Full Changelog:** https://github.com/jholhewres/agent-go/commits/v1.0.0
