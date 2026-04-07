package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const trackingTable = "agno_schema_migrations"

// migration holds the parsed metadata for a single migration file.
type migration struct {
	version string // e.g. "001"
	name    string // full filename stem, e.g. "001_init"
	sql     string
}

// Migrate applies all pending up-migrations idempotently. It creates the
// tracking table agno_schema_migrations on first run and records each applied
// version so subsequent calls are no-ops for already-applied versions.
//
// Only files matching the pattern migrations/<version>_*.up.sql are processed;
// down files are embedded for completeness but never run by this function.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ensureTrackingTable(ctx, pool); err != nil {
		return fmt.Errorf("migrate: ensure tracking table: %w", err)
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return fmt.Errorf("migrate: list applied versions: %w", err)
	}

	pending, err := pendingMigrations(applied)
	if err != nil {
		return fmt.Errorf("migrate: list pending migrations: %w", err)
	}

	for _, m := range pending {
		if err := applyMigration(ctx, pool, m); err != nil {
			return fmt.Errorf("migrate: apply %s: %w", m.name, err)
		}
	}
	return nil
}

func ensureTrackingTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    version    TEXT        PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`, trackingTable))
	return err
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]struct{}, error) {
	rows, err := pool.Query(ctx, fmt.Sprintf("SELECT version FROM %s", trackingTable))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = struct{}{}
	}
	return applied, rows.Err()
}

func pendingMigrations(applied map[string]struct{}) ([]migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}

	var pending []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		// Filename format: <version>_<name>.up.sql  (e.g. 001_init.up.sql)
		version, _, ok := parseFilename(e.Name())
		if !ok {
			continue
		}
		if _, done := applied[version]; done {
			continue
		}
		content, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration file %s: %w", e.Name(), err)
		}
		stem := strings.TrimSuffix(e.Name(), ".up.sql")
		pending = append(pending, migration{
			version: version,
			name:    stem,
			sql:     string(content),
		})
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].version < pending[j].version
	})
	return pending, nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, m migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, m.sql); err != nil {
		return fmt.Errorf("execute SQL: %w", err)
	}
	if _, err := tx.Exec(ctx,
		fmt.Sprintf("INSERT INTO %s (version, applied_at) VALUES ($1, $2) ON CONFLICT DO NOTHING", trackingTable),
		m.version, time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("record version: %w", err)
	}
	return tx.Commit(ctx)
}

// parseFilename splits "001_init.up.sql" into version="001" and name="init".
// Returns ok=false when the filename does not match the expected pattern.
func parseFilename(filename string) (version, name string, ok bool) {
	base := strings.TrimSuffix(filename, ".up.sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
