// Package main demonstrates how to initialise the Postgres session store,
// run schema migrations, and exercise basic CRUD operations.
//
// Usage:
//
//	DATABASE_URL="postgres://user:pass@localhost:5432/agno?sslmode=disable" \
//	  go run ./cmd/examples/postgres_storage/
//
// When DATABASE_URL is not set the program prints an informational message
// and exits with code 0 so CI pipelines are not broken.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jholhewres/agent-go/internal/session/dto"
	"github.com/jholhewres/agent-go/internal/session/store"
	postgresstore "github.com/jholhewres/agent-go/internal/session/store/postgres"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("DATABASE_URL not set — skipping postgres_storage example")
		log.Println("Set DATABASE_URL to a valid Postgres connection string and re-run.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Create the store with production-ready pool defaults.
	st, err := postgresstore.NewStore(ctx, dsn, postgresstore.Config{
		MaxConns:          10,
		MinConns:          2,
		MaxConnLifetime:   30 * time.Minute,
		MaxConnIdleTime:   5 * time.Minute,
		HealthCheckPeriod: time.Minute,
		ConnectTimeout:    5 * time.Second,
	})
	if err != nil {
		log.Fatalf("create store: %v", err)
	}
	defer st.Close()

	// 2. Run migrations — idempotent, safe to call on every startup.
	pool := st.Pool()
	if err := postgresstore.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	fmt.Println("migrations applied")

	// 3. Upsert a session.
	userID := "user-demo"
	record := &dto.SessionRecord{
		SessionID:   "demo-session-001",
		SessionType: dto.SessionTypeAgent,
		UserID:      &userID,
		SessionData: map[string]any{
			"session_name": "Demo Session",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	inserted, err := st.UpsertSession(ctx, record, false)
	if err != nil {
		log.Fatalf("upsert session: %v", err)
	}
	fmt.Printf("upserted session: %s\n", inserted.SessionID)

	// 4. List sessions for this tenant.
	sessions, total, err := st.ListByTenant(ctx, userID, postgresstore.ListByTenantOptions{
		SessionType: dto.SessionTypeAgent,
		Limit:       10,
		Page:        1,
	})
	if err != nil {
		log.Fatalf("list by tenant: %v", err)
	}
	fmt.Printf("tenant %q has %d session(s)\n", userID, total)
	for _, s := range sessions {
		fmt.Printf("  - %s (%s)\n", s.SessionID, s.SessionName())
	}

	// 5. List via standard interface to verify filtering.
	_, count, err := st.ListSessions(ctx, store.ListSessionsOptions{
		SessionType: dto.SessionTypeAgent,
		UserID:      userID,
	})
	if err != nil {
		log.Fatalf("list sessions: %v", err)
	}
	fmt.Printf("ListSessions count: %d\n", count)

	// 6. Delete the session.
	if err := st.DeleteSession(ctx, record.SessionID, dto.SessionTypeAgent); err != nil {
		log.Fatalf("delete session: %v", err)
	}
	fmt.Println("session deleted")
}
