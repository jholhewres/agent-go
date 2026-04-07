package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jholhewres/agent-go/internal/session/dto"
	"github.com/jholhewres/agent-go/internal/session/store"
)

// Config captures the configuration required to instantiate a Postgres backed
// session store.
type Config struct {
	// DSN is the full Postgres connection string.
	DSN string
	// TableName allows overriding the default table used to store sessions.
	TableName string

	// Pool tuning — all fields are optional; zero values use sensible defaults.

	// MaxConns caps the total connections in the pool (default: 10).
	MaxConns int32
	// MinConns keeps at least this many connections open (default: 2).
	MinConns int32
	// MaxConnLifetime caps how long a connection may be reused (default: 30m).
	MaxConnLifetime time.Duration
	// MaxConnIdleTime closes connections idle longer than this (default: 5m).
	MaxConnIdleTime time.Duration
	// HealthCheckPeriod controls how often idle connections are health-checked
	// (default: 1m).
	HealthCheckPeriod time.Duration
	// ConnectTimeout is the maximum time allowed to establish a new connection
	// (default: 5s).
	ConnectTimeout time.Duration

	// Deprecated: use MaxConnLifetime. Kept for backwards compatibility.
	MaxConnLifeTime time.Duration
}

// defaults returns a copy of cfg with zero-value pool fields filled in with
// production-ready defaults.
func (c Config) defaults() Config {
	// Backwards-compat alias.
	if c.MaxConnLifetime == 0 && c.MaxConnLifeTime > 0 {
		c.MaxConnLifetime = c.MaxConnLifeTime
	}

	if c.MaxConns == 0 {
		c.MaxConns = 10
	}
	if c.MinConns == 0 {
		c.MinConns = 2
	}
	if c.MaxConnLifetime == 0 {
		c.MaxConnLifetime = 30 * time.Minute
	}
	if c.MaxConnIdleTime == 0 {
		c.MaxConnIdleTime = 5 * time.Minute
	}
	if c.HealthCheckPeriod == 0 {
		c.HealthCheckPeriod = time.Minute
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 5 * time.Second
	}
	if c.TableName == "" {
		c.TableName = "agno_sessions"
	}
	return c
}

// Store implements the session Store interface backed by Postgres.
type Store struct {
	pool      *pgxpool.Pool
	tableName string
}

// New constructs a new Postgres backed session store.
func New(ctx context.Context, cfg Config) (*Store, error) {
	cfg = cfg.defaults()

	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, errors.New("postgres store requires a DSN")
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	return &Store{pool: pool, tableName: cfg.TableName}, nil
}

// NewStore constructs a Postgres backed session store from a plain DSN string
// and a Config. The DSN is set on cfg before calling New, making it convenient
// when callers want to separate connection string from pool tuning.
func NewStore(ctx context.Context, dsn string, cfg Config) (*Store, error) {
	cfg.DSN = dsn
	return New(ctx, cfg)
}

// Close terminates the underlying connection pool.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// Pool returns the underlying pgxpool.Pool. Callers may use it to run
// migrations (Migrate) or execute custom queries outside of the store API.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// UpsertSession inserts or updates a session record in the database.
func (s *Store) UpsertSession(ctx context.Context, record *dto.SessionRecord, preserveCreated bool) (*dto.SessionRecord, error) {
	if record == nil {
		return nil, errors.New("session record is required")
	}
	if err := record.SessionType.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	createdAt := record.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := record.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	}
	if preserveCreated {
		// When preserveCreated is true we want to avoid mutating the updatedAt if
		// the caller intentionally leaves it empty.
		if record.UpdatedAt.IsZero() {
			updatedAt = createdAt
		}
	}

	agentID := nullable(record.AgentID)
	teamID := nullable(record.TeamID)
	workflowID := nullable(record.WorkflowID)
	userID := nullable(record.UserID)

	query := fmt.Sprintf(`
INSERT INTO %s (
    session_id,
    session_type,
    agent_id,
    team_id,
    workflow_id,
    user_id,
    session_data,
    agent_data,
    team_data,
    workflow_data,
    metadata,
    runs,
    summary,
    created_at,
    updated_at
)
VALUES (
    $1, $2, $3, $4, $5, $6,
    $7::jsonb, $8::jsonb, $9::jsonb, $10::jsonb,
    $11::jsonb, $12::jsonb, $13::jsonb,
    $14, $15
)
ON CONFLICT (session_id) DO UPDATE SET
    session_type = EXCLUDED.session_type,
    agent_id = EXCLUDED.agent_id,
    team_id = EXCLUDED.team_id,
    workflow_id = EXCLUDED.workflow_id,
    user_id = EXCLUDED.user_id,
    session_data = EXCLUDED.session_data,
    agent_data = EXCLUDED.agent_data,
    team_data = EXCLUDED.team_data,
    workflow_data = EXCLUDED.workflow_data,
    metadata = EXCLUDED.metadata,
    runs = EXCLUDED.runs,
    summary = EXCLUDED.summary,
    updated_at = EXCLUDED.updated_at,
    created_at = COALESCE(%s.created_at, EXCLUDED.created_at)
RETURNING
    session_id,
    session_type,
    agent_id,
    team_id,
    workflow_id,
    user_id,
    session_data,
    agent_data,
    team_data,
    workflow_data,
    metadata,
    runs,
    summary,
    created_at,
    updated_at
`, s.tableName, s.tableName)

	sessionDataJSON, err := encodeJSON(record.SessionData)
	if err != nil {
		return nil, err
	}
	agentDataJSON, err := encodeJSON(record.AgentData)
	if err != nil {
		return nil, err
	}
	teamDataJSON, err := encodeJSON(record.TeamData)
	if err != nil {
		return nil, err
	}
	workflowDataJSON, err := encodeJSON(record.WorkflowData)
	if err != nil {
		return nil, err
	}
	metadataJSON, err := encodeJSON(record.Metadata)
	if err != nil {
		return nil, err
	}
	runsJSON, err := encodeJSON(record.Runs)
	if err != nil {
		return nil, err
	}
	summaryJSON, err := encodeJSON(record.Summary)
	if err != nil {
		return nil, err
	}

	args := []any{
		record.SessionID,
		string(record.SessionType),
		agentID,
		teamID,
		workflowID,
		userID,
		sessionDataJSON,
		agentDataJSON,
		teamDataJSON,
		workflowDataJSON,
		metadataJSON,
		runsJSON,
		summaryJSON,
		createdAt.Unix(),
		updatedAt.Unix(),
	}

	row := s.pool.QueryRow(ctx, query, args...)
	return scanRecord(row)
}

// ListSessions returns paginated sessions filtered according to the provided options.
func (s *Store) ListSessions(ctx context.Context, opts store.ListSessionsOptions) ([]*dto.SessionRecord, int, error) {
	if err := opts.SessionType.Validate(); err != nil {
		return nil, 0, err
	}

	var (
		whereClauses []string
		args         []any
	)

	whereClauses = append(whereClauses, "session_type = $1")
	args = append(args, string(opts.SessionType))
	argIndex := 2

	if opts.UserID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, opts.UserID)
		argIndex++
	}

	if opts.ComponentID != "" {
		column, err := opts.SessionType.ComponentColumn()
		if err != nil {
			return nil, 0, err
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", column, argIndex))
		args = append(args, opts.ComponentID)
		argIndex++
	}

	if opts.SessionName != "" {
		whereClauses = append(whereClauses,
			fmt.Sprintf("lower(session_data->>'session_name') LIKE lower('%%' || $%d || '%%')", argIndex),
		)
		args = append(args, opts.SessionName)
		argIndex++
	}

	allowedSortFields := map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
	}
	sortField := allowedSortFields[strings.ToLower(strings.TrimSpace(opts.SortBy))]
	if sortField == "" {
		sortField = "updated_at"
	}
	sortOrder := strings.ToUpper(strings.TrimSpace(opts.SortOrder))
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	builder := strings.Builder{}
	builder.WriteString("SELECT \n")
	builder.WriteString("    session_id, session_type, agent_id, team_id, workflow_id, user_id,\n")
	builder.WriteString("    session_data, agent_data, team_data, workflow_data, metadata, runs, summary,\n")
	builder.WriteString("    created_at, updated_at,\n")
	builder.WriteString("    COUNT(*) OVER() AS total_count\n")
	builder.WriteString(fmt.Sprintf("FROM %s\n", s.tableName))
	builder.WriteString("WHERE ")
	builder.WriteString(strings.Join(whereClauses, " AND "))
	builder.WriteString(fmt.Sprintf("\nORDER BY %s %s\n", sortField, sortOrder))

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}
	if limit > 0 {
		builder.WriteString(fmt.Sprintf("LIMIT $%d\n", argIndex))
		args = append(args, limit)
		argIndex++
		offset := (page - 1) * limit
		builder.WriteString(fmt.Sprintf("OFFSET $%d\n", argIndex))
		args = append(args, offset)
	}

	rows, err := s.pool.Query(ctx, builder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sessions query: %w", err)
	}
	defer rows.Close()

	var (
		results    []*dto.SessionRecord
		totalCount int
	)

	for rows.Next() {
		record, count, err := scanRecordWithCount(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, record)
		totalCount = count
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return results, totalCount, nil
}

// GetSession retrieves a session by identifier and type.
func (s *Store) GetSession(ctx context.Context, sessionID string, sessionType dto.SessionType) (*dto.SessionRecord, error) {
	query := fmt.Sprintf(`
SELECT
    session_id,
    session_type,
    agent_id,
    team_id,
    workflow_id,
    user_id,
    session_data,
    agent_data,
    team_data,
    workflow_data,
    metadata,
    runs,
    summary,
    created_at,
    updated_at
FROM %s
WHERE session_id = $1 AND session_type = $2
`, s.tableName)

	row := s.pool.QueryRow(ctx, query, sessionID, string(sessionType))
	record, err := scanRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return record, err
}

// DeleteSession removes a session by identifier and type.
func (s *Store) DeleteSession(ctx context.Context, sessionID string, sessionType dto.SessionType) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE session_id = $1 AND session_type = $2", s.tableName)
	commandTag, err := s.pool.Exec(ctx, query, sessionID, string(sessionType))
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// RenameSession updates the session name stored within the session_data JSON payload.
func (s *Store) RenameSession(ctx context.Context, sessionID string, sessionType dto.SessionType, sessionName string) (*dto.SessionRecord, error) {
	query := fmt.Sprintf(`
UPDATE %s
SET
    session_data = jsonb_set(COALESCE(session_data::jsonb, '{}'::jsonb), '{session_name}', to_jsonb($3::text), true),
    updated_at = EXTRACT(EPOCH FROM now())::bigint
WHERE session_id = $1 AND session_type = $2
RETURNING
    session_id,
    session_type,
    agent_id,
    team_id,
    workflow_id,
    user_id,
    session_data,
    agent_data,
    team_data,
    workflow_data,
    metadata,
    runs,
    summary,
    created_at,
    updated_at
`, s.tableName)

	row := s.pool.QueryRow(ctx, query, sessionID, string(sessionType), sessionName)
	record, err := scanRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return record, err
}

// ListByTenantOptions controls filtering and pagination for ListByTenant.
type ListByTenantOptions struct {
	// SessionType restricts results to a single session category. Required.
	SessionType dto.SessionType
	// Limit is the maximum number of records per page (default: 50).
	Limit int
	// Page is 1-based page number (default: 1).
	Page int
}

// ListByTenant returns sessions belonging to the given tenant (user_id),
// ordered by updated_at DESC. It is a thin convenience wrapper over
// ListSessions that enforces the user_id filter.
func (s *Store) ListByTenant(ctx context.Context, tenantID string, opts ListByTenantOptions) ([]*dto.SessionRecord, int, error) {
	if tenantID == "" {
		return nil, 0, errors.New("tenantID is required")
	}
	return s.ListSessions(ctx, store.ListSessionsOptions{
		SessionType: opts.SessionType,
		UserID:      tenantID,
		Limit:       opts.Limit,
		Page:        opts.Page,
		SortBy:      "updated_at",
		SortOrder:   "DESC",
	})
}

func scanRecord(row pgx.Row) (*dto.SessionRecord, error) {
	record, _, err := scanIntoRecord(row, false)
	return record, err
}

func scanRecordWithCount(rows pgx.Rows) (*dto.SessionRecord, int, error) {
	return scanIntoRecord(rows, true)
}

type pgScanner interface {
	Scan(dest ...any) error
}

func scanIntoRecord(scanner pgScanner, includeCount bool) (*dto.SessionRecord, int, error) {
	var (
		record       dto.SessionRecord
		sessionType  string
		agentID      pgtype.Text
		teamID       pgtype.Text
		workflowID   pgtype.Text
		userID       pgtype.Text
		sessionData  []byte
		agentData    []byte
		teamData     []byte
		workflowData []byte
		metadata     []byte
		runs         []byte
		summary      []byte
		createdAt    pgtype.Int8
		updatedAt    pgtype.Int8
		totalCount   pgtype.Int8
	)

	dest := []any{
		&record.SessionID,
		&sessionType,
		&agentID,
		&teamID,
		&workflowID,
		&userID,
		&sessionData,
		&agentData,
		&teamData,
		&workflowData,
		&metadata,
		&runs,
		&summary,
		&createdAt,
		&updatedAt,
	}
	if includeCount {
		dest = append(dest, &totalCount)
	}

	if err := scanner.Scan(dest...); err != nil {
		return nil, 0, err
	}

	record.SessionType = dto.SessionType(sessionType)
	record.AgentID = textPtr(agentID)
	record.TeamID = textPtr(teamID)
	record.WorkflowID = textPtr(workflowID)
	record.UserID = textPtr(userID)

	if createdAt.Valid {
		record.CreatedAt = time.Unix(createdAt.Int64, 0).UTC()
	}
	if updatedAt.Valid {
		record.UpdatedAt = time.Unix(updatedAt.Int64, 0).UTC()
	} else {
		record.UpdatedAt = record.CreatedAt
	}

	var err error
	if record.SessionData, err = decodeJSONMap(sessionData); err != nil {
		return nil, 0, err
	}
	if record.AgentData, err = decodeJSONMap(agentData); err != nil {
		return nil, 0, err
	}
	if record.TeamData, err = decodeJSONMap(teamData); err != nil {
		return nil, 0, err
	}
	if record.WorkflowData, err = decodeJSONMap(workflowData); err != nil {
		return nil, 0, err
	}
	if record.Metadata, err = decodeJSONMap(metadata); err != nil {
		return nil, 0, err
	}
	if record.Summary, err = decodeJSONMap(summary); err != nil {
		return nil, 0, err
	}
	if record.Runs, err = decodeJSONSlice(runs); err != nil {
		return nil, 0, err
	}

	count := 0
	if includeCount && totalCount.Valid {
		count = int(totalCount.Int64)
	}

	return &record, count, nil
}

func decodeJSONMap(payload []byte) (map[string]any, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func decodeJSONSlice(payload []byte) ([]map[string]any, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var result []map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func encodeJSON(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	buffer, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return string(buffer), nil
}

func nullable(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func textPtr(text pgtype.Text) *string {
	if !text.Valid {
		return nil
	}
	value := text.String
	return &value
}
