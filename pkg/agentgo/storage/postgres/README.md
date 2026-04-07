# Postgres Session Store

The Postgres storage backend persists agent, team, and workflow sessions to a
PostgreSQL database. It is the recommended backend for production workloads
that require durability and multi-tenant isolation.

> **Note**: The implementation lives in
> `internal/session/store/postgres/`.  This directory contains only
> documentation; import the internal package directly.

---

## When to use this backend

| Scenario | Recommendation |
|---|---|
| Single-process development / testing | In-memory store (no external dependency) |
| Production, single tenant | Postgres — durable, queryable |
| Production, multi-tenant SaaS | Postgres — built-in `user_id` isolation |
| High-throughput read path (cached) | Postgres + Redis read-through cache |

---

## Schema

```
agno_schema_migrations          (migration tracking)
┌─────────────┬─────────────────────────────┐
│ version     │ TEXT PRIMARY KEY            │
│ applied_at  │ TIMESTAMPTZ NOT NULL        │
└─────────────┴─────────────────────────────┘

agno_sessions                   (session records)
┌──────────────┬──────────────────────────────────────────────────┐
│ session_id   │ TEXT PRIMARY KEY                                  │
│ session_type │ TEXT NOT NULL  ("agent" | "team" | "workflow")    │
│ agent_id     │ TEXT                                              │
│ team_id      │ TEXT                                              │
│ workflow_id  │ TEXT                                              │
│ user_id      │ TEXT                    ← tenant key              │
│ session_data │ JSONB                                             │
│ agent_data   │ JSONB                                             │
│ team_data    │ JSONB                                             │
│ workflow_data│ JSONB                                             │
│ metadata     │ JSONB                                             │
│ runs         │ JSONB                                             │
│ summary      │ JSONB                                             │
│ created_at   │ BIGINT NOT NULL         (Unix epoch seconds)      │
│ updated_at   │ BIGINT                  (Unix epoch seconds)      │
└──────────────┴──────────────────────────────────────────────────┘
```

### Indexes

| Index name | Columns | Purpose |
|---|---|---|
| `idx_agno_sessions_user_updated` | `(user_id, updated_at DESC)` | List sessions per tenant ordered by recency |
| `idx_agno_sessions_type_agent` | `(session_type, agent_id)` | Filter by session type + agent |
| `idx_agno_sessions_workflow` | `(workflow_id) WHERE NOT NULL` | Workflow lookup (partial index) |

---

## Connection pool tuning

| Field | Default | Notes |
|---|---|---|
| `MaxConns` | 10 | Increase for high-concurrency services |
| `MinConns` | 2 | Keep warm connections ready |
| `MaxConnLifetime` | 30m | Rotates credentials on re-connect |
| `MaxConnIdleTime` | 5m | Closes unused connections |
| `HealthCheckPeriod` | 1m | Detects stale idle connections |
| `ConnectTimeout` | 5s | Fail fast on network partition |

A good starting point for a medium-traffic API:

```go
postgresstore.Config{
    MaxConns:          20,
    MinConns:          5,
    MaxConnLifetime:   1 * time.Hour,
    MaxConnIdleTime:   10 * time.Minute,
    HealthCheckPeriod: 30 * time.Second,
    ConnectTimeout:    5 * time.Second,
}
```

---

## Migration workflow

Migrations are embedded SQL files located in
`internal/session/store/postgres/migrations/`.

Naming convention: `<version>_<name>.up.sql` / `<version>_<name>.down.sql`
(e.g. `001_init.up.sql`).

`Migrate` applies pending up-migrations inside individual transactions and
records each applied version in `agno_schema_migrations`. Calling it multiple
times is safe — already-applied migrations are skipped.

To add a new migration:
1. Create `002_add_column.up.sql` and `002_add_column.down.sql`.
2. Calling `Migrate` on next startup applies it automatically.

---

## Initialisation example

```go
package main

import (
    "context"
    "log"
    "time"
    "os"

    postgresstore "github.com/jholhewres/agent-go/internal/session/store/postgres"
    "github.com/jholhewres/agent-go/internal/session/dto"
)

func main() {
    ctx := context.Background()

    st, err := postgresstore.NewStore(ctx, os.Getenv("DATABASE_URL"), postgresstore.Config{
        MaxConns:          10,
        MinConns:          2,
        MaxConnLifetime:   30 * time.Minute,
        MaxConnIdleTime:   5 * time.Minute,
        HealthCheckPeriod: time.Minute,
        ConnectTimeout:    5 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer st.Close()

    // Apply migrations on startup (idempotent).
    if err := postgresstore.Migrate(ctx, st.Pool()); err != nil {
        log.Fatal(err)
    }

    // List sessions for a tenant.
    sessions, total, err := st.ListByTenant(ctx, "user-123", postgresstore.ListByTenantOptions{
        SessionType: dto.SessionTypeAgent,
        Limit:       20,
        Page:        1,
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("%d sessions (page 1 of %d)", len(sessions), total)
}
```

---

## Backup & restore

### Logical backup (recommended)

```bash
# Backup
pg_dump -Fc -d "$DATABASE_URL" -t agno_sessions -t agno_schema_migrations \
  -f agno_sessions_$(date +%Y%m%d).dump

# Restore to a new database
pg_restore -d "$TARGET_DATABASE_URL" agno_sessions_20260101.dump
```

### Continuous archiving (WAL)

For zero-RPO requirements configure WAL archiving with
`archive_mode = on` and `archive_command` in `postgresql.conf`. Tools such as
pgBackRest or Barman provide managed WAL streaming and point-in-time recovery.

### Snapshot-based (cloud)

Cloud-managed Postgres (RDS, Cloud SQL, AlloyDB) supports automated daily
snapshots and point-in-time recovery — enable these at the database
instance level rather than at the application level.
