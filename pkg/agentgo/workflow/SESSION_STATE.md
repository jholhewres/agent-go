# Workflow Session State Management

## Overview

SessionState provides session-level state management across workflow steps, specifically solving race condition issues during parallel step execution.

## Core Features

- **Thread-safe** - Uses sync.RWMutex to protect concurrent access
- **Deep copy** - Creates independent copies for parallel branches
- **Smart merging** - Only applies actual changes
- **Session isolation** - Supports SessionID and UserID

## Quick Start

### 1. Create Context with Session State

```go
// Basic context
ctx := workflow.NewExecutionContext("input text")

// Context with session info
ctx := workflow.NewExecutionContextWithSession(
    "input text",
    "session-123",
    "user-456",
)
```

### 2. Use Session State

```go
// Set value
ctx.SetSessionState("counter", 1)
ctx.SetSessionState("user_name", "Alice")

// Get value
if val, ok := ctx.GetSessionState("counter"); ok {
    counter := val.(int)
    fmt.Println("Counter:", counter)
}

// Direct access
ctx.SessionState.Set("key", "value")
value, exists := ctx.SessionState.Get("key")
```

### 3. Session State in Parallel Steps

```go
// Create parallel node
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
    ID: "parallel-processing",
    Nodes: []workflow.Node{
        step1, // Each step operates on its own SessionState clone
        step2,
        step3,
    },
})

// States are auto-merged after execution
result, _ := parallel.Execute(ctx, execCtx)
```

## Problem Solved

### Issue in Python Version

**Problem**: Parallel steps sharing the same `session_state` dict cause race conditions.

```python
# Python - Problem code
def execute_parallel(steps, session_state):
    # All steps share the same dict!
    for step in steps:
        await step.run(session_state)  # Race condition
```

### Solution in Go Version

**Solution**: Create independent SessionState clone for each parallel branch.

```go
// Go - Solution
sessionStateCopies := make([]*SessionState, len(nodes))
for i := range nodes {
    if execCtx.SessionState != nil {
        sessionStateCopies[i] = execCtx.SessionState.Clone() // Independent copy
    }
}

// Merge after execution
execCtx.SessionState = MergeParallelSessionStates(
    originalSessionState,
    modifiedSessionStates,
)
```

## Advanced Usage

### Clone Session State

```go
// Deep copy session state
cloned := sessionState.Clone()

// Modifying clone doesn't affect original
cloned.Set("key", "new value")
```

### Merge Multiple States

```go
// Merge another state
sessionState.Merge(anotherState)

// Merge parallel branch states
merged := workflow.MergeParallelSessionStates(
    originalState,
    []SessionState{branch1State, branch2State, branch3State},
)
```

### Get All Data

```go
// Get copy of all data
allData := sessionState.GetAll()

// Convert to plain map
dataMap := sessionState.ToMap()
```

## Concurrency Safety Guarantees

### RWMutex Mechanism

```go
type SessionState struct {
    mu   sync.RWMutex              // Read-write lock
    data map[string]interface{}
}

// Read operations use read lock
func (ss *SessionState) Get(key string) (interface{}, bool) {
    ss.mu.RLock()                  // Multiple readers allowed
    defer ss.mu.RUnlock()
    // ...
}

// Write operations use write lock
func (ss *SessionState) Set(key string, value interface{}) {
    ss.mu.Lock()                   // Exclusive write access
    defer ss.mu.Unlock()
    // ...
}
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"

    "github.com/jholhewres/agent-go/pkg/agentgo/workflow"
)

func main() {
    // Create workflow context
    execCtx := workflow.NewExecutionContextWithSession(
        "Process data",
        "session-abc-123",
        "user-xyz-789",
    )

    // Initialize session state
    execCtx.SetSessionState("total_processed", 0)
    execCtx.SetSessionState("errors", []string{})

    // Create parallel processing steps
    parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
        ID: "data-processing",
        Nodes: []workflow.Node{
            // Step 1: Process batch 1
            step1,
            // Step 2: Process batch 2
            step2,
            // Step 3: Process batch 3
            step3,
        },
    })

    // Execute parallel steps
    result, err := parallel.Execute(context.Background(), execCtx)
    if err != nil {
        panic(err)
    }

    // Check merged state
    if total, ok := result.GetSessionState("total_processed"); ok {
        fmt.Printf("Total processed: %v\n", total)
    }

    if errors, ok := result.GetSessionState("errors"); ok {
        fmt.Printf("Errors: %v\n", errors)
    }
}
```

## Best Practices

### 1. Use Descriptive Key Names

```go
// Good
ctx.SetSessionState("user_authentication_token", token)
ctx.SetSessionState("workflow_start_time", time.Now())

// Bad
ctx.SetSessionState("t", token)
ctx.SetSessionState("x", time.Now())
```

### 2. Check Value Existence

```go
// Good
if val, ok := ctx.GetSessionState("key"); ok {
    // Use val
}

// Bad
val := ctx.GetSessionState("key")  // Doesn't compile
```

### 3. Avoid Storing Large Objects

```go
// Good - Store reference or ID
ctx.SetSessionState("document_id", docID)

// Bad - Store large object
ctx.SetSessionState("full_document", largeDocument) // Will be deep-copied!
```

### 4. Avoid Deleting Shared Keys in Parallel

```go
// Careful - Deleting keys in parallel steps
// May lead to unpredictable results during merge
ctx.SessionState.Delete("shared_key")
```

## Performance Considerations

- **Cloning overhead**: O(n) where n = number of keys
- **Merging overhead**: O(m*n) where m = branches, n = keys
- **Deep copy**: Uses JSON serialization (can be optimized)
- **Recommendation**: Limit session state size

## API Reference

### SessionState Methods

| Method | Description |
|--------|-------------|
| `Set(key, value)` | Set value |
| `Get(key)` | Get value |
| `GetAll()` | Get all data copy |
| `Delete(key)` | Delete key |
| `Clear()` | Clear all data |
| `Clone()` | Deep copy |
| `Merge(other)` | Merge another state |
| `ToMap()` | Convert to plain map |

### ExecutionContext Methods

| Method | Description |
|--------|-------------|
| `SetSessionState(key, value)` | Set session state value |
| `GetSessionState(key)` | Get session state value |
| `Set(key, value)` | Set context data |
| `Get(key)` | Get context data |

## Compatibility with Python Version

This implementation is compatible with Python Agno v2.1.2 session state management, fixing the same concurrency issues.

## License

Apache License 2.0
