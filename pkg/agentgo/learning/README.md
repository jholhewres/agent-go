# learning

Persistent user-level learning and knowledge extraction for agents.

## Purpose

The `learning` package provides a `LearningMachine` interface and concrete implementations that extract structured information from conversation messages, build per-user profiles and memories, and accumulate shared knowledge — enabling agents to personalise responses over time and comply with data-deletion requirements (GDPR).

## Main Types

- `LearningMachine` — interface: `Learn`, `GetUserProfile`, `GetUserMemories`, `GetLearnedKnowledge`, `DeleteUserData`
- `UserProfile` — per-user preferences and contextual metadata
- `UserMemory` — timestamped memory entry linked to a user
- `Knowledge` — topic-scoped knowledge fact with source attribution
- `Extractor` — LLM-backed extractor that parses messages into memories/facts

## Minimal Example

```go
import (
    "github.com/jholhewres/agent-go/pkg/agentgo/learning"
    "github.com/jholhewres/agent-go/pkg/agentgo/learning/sqlite"
)

lm, err := sqlite.New("data/learning.db")
if err != nil {
    log.Fatal(err)
}

err = lm.Learn(ctx, "user-123", messages)

profile, _ := lm.GetUserProfile(ctx, "user-123")
mems, _ := lm.GetUserMemories(ctx, "user-123", 10)
```

## Status

**stable** — used by `pkg/agentgo/agent` and `cmd/examples/learning_agent`.
