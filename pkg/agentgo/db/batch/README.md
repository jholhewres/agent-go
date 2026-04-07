# Batch Operations for PostgreSQL

High-performance PostgreSQL batch session write implementation using COPY protocol and temporary table strategy.

## Features

- **High Performance**: Uses PostgreSQL COPY protocol, supports >10,000 records/sec
- **Atomicity**: All operations execute in transactions, ensuring data consistency
- **Flexibility**: Supports preserving or auto-updating `updated_at` timestamps
- **Configurable**: Batch size, retry count, timeout are all customizable
- **Test Coverage**: 80.9% test coverage with comprehensive unit and integration tests

## Architecture

### COPY + Temporary Table Strategy

```
1. CREATE TEMPORARY TABLE temp_sessions
2. COPY data INTO temp_sessions (Bulk import)
3. INSERT INTO sessions ... FROM temp_sessions ON CONFLICT DO UPDATE
4. DROP temp_sessions (Auto on transaction end)
```

### Benefits

- **Fast**: COPY is 10-100x faster than individual INSERTs
- **Memory Optimized**: Temporary tables auto-cleanup after transaction
- **UPSERT Support**: Automatically handles both insert and update

## Installation

```bash
go get github.com/jholhewres/agent-go
```

Dependencies:
- `github.com/lib/pq` - PostgreSQL driver
- Go 1.21+

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "database/sql"
    "log"

    _ "github.com/lib/pq"
    "github.com/jholhewres/agent-go/pkg/agentgo/db/batch"
    "github.com/jholhewres/agent-go/pkg/agentgo/session"
)

func main() {
    // 1. Connect to database
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/agno?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // 2. Create batch writer
    writer, err := batch.NewPostgresBatchWriter(db, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    // 3. Prepare data
    sessions := []*session.Session{
        {
            SessionID: "session-1",
            AgentID:   "agent-001",
            UserID:    "user-001",
            Name:      "Example Session",
            // ... other fields
        },
    }

    // 4. Batch upsert
    ctx := context.Background()
    if err := writer.UpsertSessions(ctx, sessions, false); err != nil {
        log.Fatal(err)
    }

    log.Println("Success!")
}
```

### Custom Configuration

```go
config := &batch.Config{
    BatchSize:        1000,               // Process 1000 records per batch
    MinBatchSize:     200,                // Minimum batch size when shrinking
    MaxRetries:       5,                  // Retry up to 5 times on failure
    TimeoutSeconds:   60,                 // 60 seconds timeout per batch
    ThrottleInterval: 100 * time.Millisecond, // Sleep 100ms between batches
}

writer, err := batch.NewPostgresBatchWriter(db, config)
```

### Preserve Timestamps (Data Migration)

```go
// When migrating historical data, preserve original updated_at timestamps
err := writer.UpsertSessions(ctx, sessions, true) // preserveUpdatedAt=true
```

### Auto-Update Timestamps (Normal Operations)

```go
// For normal insert/update, auto-set updated_at to current time
err := writer.UpsertSessions(ctx, sessions, false) // preserveUpdatedAt=false
```

## API Documentation

### BatchWriter Interface

```go
type BatchWriter interface {
    // UpsertSessions batch inserts or updates sessions
    UpsertSessions(ctx context.Context, sessions []*session.Session, preserveUpdatedAt bool) error

    // Close closes the batch writer and releases resources
    Close() error
}
```

### Config

```go
type Config struct {
    BatchSize        int           // Batch size, default 5000
    MinBatchSize     int           // Minimum batch size, default 500
    MaxRetries       int           // Max retries, default 3
    TimeoutSeconds   int           // Timeout (seconds), default 30
    ThrottleInterval time.Duration // Sleep duration between batches, default 0
}
```

### NewPostgresBatchWriter

```go
func NewPostgresBatchWriter(db *sql.DB, config *Config) (*PostgresBatchWriter, error)
```

Creates a PostgreSQL batch writer. If `config` is `nil`, uses default configuration.

## Performance Benchmarks

### Test Environment
- PostgreSQL 15
- Table: 13 columns (5 JSONB, 2 timestamps)
- Network: localhost

### Results

| Records | Time | Throughput |
|---------|------|------------|
| 1,000   | ~80ms | 12,500 records/sec |
| 5,000   | ~350ms | 14,285 records/sec |
| 10,000  | ~680ms | 14,706 records/sec |

> Actual performance depends on network latency, table complexity, database load, etc.

## Database Schema

```sql
CREATE TABLE sessions (
    session_id VARCHAR(255) PRIMARY KEY,
    agent_id VARCHAR(255),
    team_id VARCHAR(255),
    workflow_id VARCHAR(255),
    user_id VARCHAR(255),
    name VARCHAR(255),
    metadata JSONB,
    state JSONB,
    agent_data JSONB,
    runs JSONB,
    summary JSONB,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);
```

## Examples

Complete example code is in [`examples/batch_upsert/`](../../../examples/batch_upsert/) directory.

Run Example:

```bash
# 1. Start PostgreSQL
docker-compose up -d postgres

# 2. Run example
go run examples/batch_upsert/main.go
```

## Testing

### Unit Tests

```bash
go test ./pkg/agentgo/db/batch/...
```

### Test Coverage

```bash
go test -cover ./pkg/agentgo/db/batch/...
# coverage: 80.9% of statements
```

### Integration Tests

```bash
# Requires running PostgreSQL instance
go test -tags=integration ./pkg/agentgo/db/batch/...
```

### Race Detection

```bash
go test -race ./pkg/agentgo/db/batch/...
```

## Error Handling

All errors are wrapped using `fmt.Errorf`, and you can use `errors.Unwrap` to get the original error.

```go
err := writer.UpsertSessions(ctx, sessions, false)
if err != nil {
    // Error already includes context
    log.Printf("Failed to upsert: %v", err)

    // Can check for specific errors
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Operation timed out")
    }
}
```

## Best Practices

### 1. Batch Size

- **Default 5000** works for most scenarios
- Increase for high network latency
- Decrease for limited memory

### 2. Transaction Timeout

```go
// For large batches, increase timeout
config := &batch.Config{
    TimeoutSeconds: 120, // 2 minutes
}
```

### 3. Error Retry

```go
// Increase retries for unstable networks
config := &batch.Config{
    MaxRetries: 5,
}
```

### 4. Context Cancellation

```go
// Use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := writer.UpsertSessions(ctx, sessions, false)
```

## Notes

1. **Connection Pool**: Ensure database connection pool is properly configured
2. **Index**: `session_id` must be primary key or unique index
3. **JSONB**: Using JSONB type allows direct JSON field queries in PostgreSQL
4. **Transaction**: All operations in single transaction, auto-rollback on failure

## Troubleshooting

### Error: "db cannot be nil"

Ensure a valid `*sql.DB` instance is passed.

### Error: "failed to create temp table"

Check if database user has permission to create temporary tables.

### Performance Below Expectations

1. Check network latency
2. Check database load
3. Optimize table indexes
4. Adjust batch size

## Contributing

Issues and Pull Requests are welcome!

## License

MIT License - See [LICENSE](../../../../LICENSE) file
