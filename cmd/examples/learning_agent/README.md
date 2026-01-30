# Learning Agent Example

This example demonstrates AgentGo's **Learning System** - agents that learn and improve over time.

## Features Demonstrated

- **User Profiles**: Persistent user information across sessions
- **User Memories**: Extracted facts, preferences, and context from conversations
- **Knowledge Transfer**: Learned knowledge that can be shared across users
- **SQLite Storage**: Lightweight local storage for learning data

## How It Works

1. **Enable Learning**: Set `Learning: true` and provide a `LearningMachine`
2. **Automatic Extraction**: After each conversation, the agent extracts:
   - User preferences ("I prefer short answers")
   - Factual information ("My name is John")
   - Contextual knowledge
3. **Memory Retrieval**: On subsequent interactions, the agent recalls learned information
4. **Knowledge Sharing**: Learned insights can benefit other users

## Run the Example

```bash
export OPENAI_API_KEY=your-api-key
go run cmd/examples/learning_agent/main.go
```

## Expected Output

```
=== First Interaction ===
Agent: Hello John! I'll keep my responses concise as you prefer.

=== Second Interaction ===
Agent: Your name is John.

=== Learned Memories ===
User Profile: {...}

User Memories (3):
  - [fact] My name is John
  - [preference] I prefer short, concise answers
  - [context] Hi! My name is John and I prefer short, concise answers.
```

## Code Walkthrough

```go
// Create learning storage
learningStorage, _ := sqlite.New("./learning.db")

// Create learning machine
learningMachine, _ := learning.NewMachine(learningStorage)

// Create agent with learning
ag, _ := agent.New(agent.Config{
    Name:            "Learning Assistant",
    Model:           model,
    UserID:          "user-123",      // Required for learning
    Learning:        true,             // Enable learning
    LearningMachine: learningMachine,
})
```

## Storage Options

AgentGo supports multiple storage backends for learning:

**SQLite** (local, lightweight):
```go
storage, _ := sqlite.New("./learning.db")
```

**PostgreSQL** (production, scalable):
```go
import "github.com/jholhewres/agent-go/pkg/agentgo/learning/postgres"

db, _ := sql.Open("postgres", "postgresql://localhost/agentgo")
storage, _ := postgres.New(db, "public")
```

## GDPR Compliance

Delete all user data:

```go
err := learningMachine.DeleteUserData(ctx, "user-123")
```

## Next Steps

- Try different conversation patterns
- Experiment with multiple users
- Integrate with PostgreSQL for production
- Build agents that transfer knowledge between users
