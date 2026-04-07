# Memory Package

The `memory` package provides conversation history management for AgentGo agents.
Three implementations cover a range of needs, from simple prototyping to production
workloads with large conversation histories.

## Implementations

### InMemory

Simple in-memory store with optional auto-truncation. Messages are keyed per user
for multi-tenant support.

```go
mem := memory.NewInMemory(100) // keep at most 100 messages
mem.Add(types.NewUserMessage("hello"))
msgs := mem.GetMessages()
```

**Best for**: Prototyping, short conversations, unit tests.

---

### HybridMemory

Combines a fast in-memory short-term buffer with a vector database for long-term
storage. Older messages are automatically migrated to the vector DB, enabling
semantic search over the full history.

```go
hybrid, err := memory.NewHybridMemory(memory.HybridMemoryConfig{
    VectorDB:             myChromaDB,
    Embedder:             myEmbedder,
    MaxShortTermMessages: 50,
    LongTermThreshold:    20,
})
results, err := hybrid.Search(ctx, "what did we discuss about payments?", 5)
```

**Best for**: Long-running agents that need semantic recall over their full history.

---

### Summarizing Memory

Wraps any existing `Memory` implementation. When the number of stored messages
exceeds a configurable `Threshold`, the oldest messages (all except the last
`PreserveLast`) are condensed into a single System message via an LLM call.
This keeps context windows small while preserving the gist of prior conversation.

```go
package main

import (
    "context"
    "fmt"

    "github.com/jholhewres/agent-go/pkg/agentgo/memory"
    "github.com/jholhewres/agent-go/pkg/agentgo/models"
    "github.com/jholhewres/agent-go/pkg/agentgo/models/openai"
    "github.com/jholhewres/agent-go/pkg/agentgo/types"
)

func main() {
    inner := memory.NewInMemory(200)

    llm, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: "sk-..."})

    sm, err := memory.NewSummarizingMemory(memory.SummarizingConfig{
        Inner:            inner,          // any Memory impl, including HybridMemory
        Model:            llm,
        Threshold:        50,             // compact when > 50 messages
        PreserveLast:     10,             // always keep last 10 verbatim
        MaxSummaryTokens: 500,
        SummaryTag:       "[Summary]",
    })
    if err != nil {
        panic(err)
    }

    for i := 0; i < 60; i++ {
        sm.Add(types.NewUserMessage(fmt.Sprintf("turn %d", i)))
    }

    // After compaction: 1 summary System message + 10 recent messages = 11 total.
    msgs := sm.GetMessages()
    fmt.Println("messages after compaction:", len(msgs)) // 11
    fmt.Println("first message role:", msgs[0].Role)     // system
    _ = context.Background()
}
```

**Configuration options**:

| Field              | Default                     | Description                                        |
|--------------------|-----------------------------|----------------------------------------------------|
| `Inner`            | *(required)*                | The underlying Memory to wrap                      |
| `Model`            | *(required)*                | LLM used to generate summaries                     |
| `Threshold`        | `50`                        | Triggers compaction when `len(messages) > Threshold` |
| `PreserveLast`     | `10`                        | Number of recent messages kept verbatim            |
| `MaxSummaryTokens` | `500`                       | Token budget for the summary response              |
| `SummaryPrompt`    | built-in instruction        | Override the summarizer system prompt              |
| `SummaryTag`       | `[Conversation Summary]`    | Prefix prepended to the generated summary          |

**Best for**: Agents with long conversations where you want bounded context
without losing conversational continuity.

---

## Comparison

| Feature              | InMemory      | HybridMemory          | SummarizingMemory            |
|----------------------|---------------|-----------------------|------------------------------|
| Setup complexity     | None          | Requires VectorDB     | Requires LLM                 |
| Semantic search      | No            | Yes                   | No                           |
| Context compression  | Truncation    | No (offloads to DB)   | LLM summarization            |
| Exact recall         | Yes (recent)  | Yes (via search)      | Approximate (in summary)     |
| Multi-tenant         | Yes           | Yes                   | Delegates to inner           |
| External deps        | None          | Vector DB + Embedder  | LLM API                      |
| Best for             | Prototyping   | Long-term recall      | Long conversations, RAG-less |

## Error Handling

`SummarizingMemory.Add` never returns an error. If the LLM summarization fails
(network error, rate limit, etc.), the compaction is skipped and the original
messages are preserved. The error is logged via the standard `log` package.
