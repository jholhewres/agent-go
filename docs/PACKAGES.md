# Package Index — pkg/agentgo

All top-level packages under `pkg/agentgo/`. Status legend: ✅ stable / 🚧 beta / 🧪 experimental / 📋 planned.

| Package | Purpose | Status |
|---|---|---|
| `agent` | Core `Agent` struct and `Run` loop with tool-call orchestration | ✅ stable |
| `cache` | Response and computation caching layer | 🚧 beta |
| `db` | Database adapters and batch helpers (SQLite, Postgres, etc.) | 🚧 beta |
| `debug` | Debug utilities and trace helpers for agent execution | 🚧 beta |
| `embeddings` | Text embedding interface and provider implementations (OpenAI, etc.) | ✅ stable |
| `experimental` | Packages with unstable APIs — see sub-directories | 🧪 experimental |
| `experimental/cloud` | Minimal `Deployer` interface for publishing agent artifacts | 🧪 experimental |
| `experimental/culture` | Tagged cultural-knowledge store for per-org agent configuration | 🧪 experimental |
| `experimental/eval` | LLM evaluation harness: scenarios, metrics, model comparison | 🧪 experimental |
| `experimental/integrations` | Thread-safe registry for third-party service integrations with health checks | 🧪 experimental |
| `guardrails` | Input/output guardrail hooks for safety and policy enforcement | ✅ stable |
| `hooks` | Tool execution lifecycle hooks (pre/post) | ✅ stable |
| `knowledge` | Knowledge base abstraction for RAG-style retrieval | 🚧 beta |
| `learning` | Persistent user learning, profile building, and knowledge extraction | ✅ stable |
| `mcp` | Model Context Protocol server and client integration | ✅ stable |
| `media` | Multimodal attachment normalisation (`Attachment`, `Normalize`) | ✅ stable |
| `memory` | Conversation history store with auto-truncation | ✅ stable |
| `models` | LLM provider interface and implementations (OpenAI, Anthropic, Groq, Ollama, GLM, etc.) | ✅ stable |
| `prompts` | Prompt composition utilities and `PromptComposer` | ✅ stable |
| `providers` | Provider-specific helpers and credential management | 🚧 beta |
| `reasoning` | Structured reasoning and chain-of-thought helpers | 🚧 beta |
| `run` | `RunContext` and per-run metadata/option types | ✅ stable |
| `session` | Session state persistence and management | 🚧 beta |
| `skills` | Skill registry for reusable agent capabilities | 🚧 beta |
| `structured` | Structured output parsing and schema validation | 🚧 beta |
| `team` | Multi-agent team orchestration (sequential, parallel, leader-follower, consensus) | ✅ stable |
| `tools` | Tool/toolkit system with built-in implementations (calculator, HTTP, file) | ✅ stable |
| `types` | Core types: `Message`, `ModelResponse`, error types | ✅ stable |
| `utils` | Shared utility functions | 🚧 beta |
| `vectordb` | Vector database adapters (ChromaDB, Redis) | 🚧 beta |
| `workflow` | Step-based workflow engine (Step, Condition, Loop, Parallel, Router) | ✅ stable |
