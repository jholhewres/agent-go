package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jholhewres/agent-go/pkg/agno/learning"
)

// Storage implements learning.Storage for PostgreSQL
type Storage struct {
	db     *sql.DB
	schema string
}

// New creates a new PostgreSQL storage for learning
func New(db *sql.DB, schema string) (*Storage, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	if schema == "" {
		schema = "public"
	}

	storage := &Storage{
		db:     db,
		schema: schema,
	}

	// Run migrations
	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return storage, nil
}

// migrate runs database migrations
func (s *Storage) migrate() error {
	ctx := context.Background()

	migrations := []string{
		// User profiles table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.learning_user_profiles (
				user_id VARCHAR(255) PRIMARY KEY,
				name VARCHAR(255),
				preferences JSONB DEFAULT '{}'::jsonb,
				context JSONB DEFAULT '{}'::jsonb,
				created_at TIMESTAMP DEFAULT NOW(),
				updated_at TIMESTAMP DEFAULT NOW()
			)
		`, s.schema),

		// User memories table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.learning_user_memories (
				id VARCHAR(255) PRIMARY KEY,
				user_id VARCHAR(255) NOT NULL,
				content TEXT NOT NULL,
				type VARCHAR(50) NOT NULL,
				metadata JSONB DEFAULT '{}'::jsonb,
				created_at TIMESTAMP DEFAULT NOW(),
				FOREIGN KEY (user_id) REFERENCES %s.learning_user_profiles(user_id) ON DELETE CASCADE
			)
		`, s.schema, s.schema),

		// Learned knowledge table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.learning_knowledge (
				id VARCHAR(255) PRIMARY KEY,
				topic VARCHAR(255) NOT NULL,
				content TEXT NOT NULL,
				source VARCHAR(50) NOT NULL,
				relevance FLOAT DEFAULT 0.5,
				metadata JSONB DEFAULT '{}'::jsonb,
				created_at TIMESTAMP DEFAULT NOW()
			)
		`, s.schema),

		// Learning events table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.learning_events (
				id VARCHAR(255) PRIMARY KEY,
				user_id VARCHAR(255) NOT NULL,
				event_type VARCHAR(100) NOT NULL,
				data JSONB DEFAULT '{}'::jsonb,
				occurred_at TIMESTAMP DEFAULT NOW()
			)
		`, s.schema),

		// Indexes
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_learning_memories_user_id ON %s.learning_user_memories(user_id)`, s.schema),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_learning_memories_type ON %s.learning_user_memories(type)`, s.schema),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_learning_knowledge_topic ON %s.learning_knowledge(topic)`, s.schema),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_learning_events_user_id ON %s.learning_events(user_id)`, s.schema),
	}

	for _, migration := range migrations {
		if _, err := s.db.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// SaveUserProfile saves or updates a user profile
func (s *Storage) SaveUserProfile(ctx context.Context, profile *learning.UserProfile) error {
	preferencesJSON, err := json.Marshal(profile.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	contextJSON, err := json.Marshal(profile.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.learning_user_profiles (user_id, name, preferences, context, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE
		SET name = EXCLUDED.name,
		    preferences = EXCLUDED.preferences,
		    context = EXCLUDED.context,
		    updated_at = EXCLUDED.updated_at
	`, s.schema)

	_, err = s.db.ExecContext(ctx, query,
		profile.UserID,
		profile.Name,
		preferencesJSON,
		contextJSON,
		profile.CreatedAt,
		profile.UpdatedAt,
	)

	return err
}

// GetUserProfile retrieves a user profile
func (s *Storage) GetUserProfile(ctx context.Context, userID string) (*learning.UserProfile, error) {
	query := fmt.Sprintf(`
		SELECT user_id, name, preferences, context, created_at, updated_at
		FROM %s.learning_user_profiles
		WHERE user_id = $1
	`, s.schema)

	var profile learning.UserProfile
	var preferencesJSON, contextJSON []byte

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.UserID,
		&profile.Name,
		&preferencesJSON,
		&contextJSON,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user profile not found")
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(preferencesJSON, &profile.Preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	if err := json.Unmarshal(contextJSON, &profile.Context); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	return &profile, nil
}

// SaveUserMemory saves a user memory
func (s *Storage) SaveUserMemory(ctx context.Context, memory *learning.UserMemory) error {
	metadataJSON, err := json.Marshal(memory.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.learning_user_memories (id, user_id, content, type, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`, s.schema)

	_, err = s.db.ExecContext(ctx, query,
		memory.ID,
		memory.UserID,
		memory.Content,
		string(memory.Type),
		metadataJSON,
		memory.CreatedAt,
	)

	return err
}

// GetUserMemories retrieves user memories
func (s *Storage) GetUserMemories(ctx context.Context, userID string, limit int) ([]learning.UserMemory, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, content, type, metadata, created_at
		FROM %s.learning_user_memories
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, s.schema)

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []learning.UserMemory
	for rows.Next() {
		var memory learning.UserMemory
		var memoryType string
		var metadataJSON []byte

		err := rows.Scan(
			&memory.ID,
			&memory.UserID,
			&memory.Content,
			&memoryType,
			&metadataJSON,
			&memory.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		memory.Type = learning.MemoryType(memoryType)

		if err := json.Unmarshal(metadataJSON, &memory.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		memories = append(memories, memory)
	}

	return memories, rows.Err()
}

// DeleteUserMemories deletes all memories for a user
func (s *Storage) DeleteUserMemories(ctx context.Context, userID string) error {
	query := fmt.Sprintf(`DELETE FROM %s.learning_user_memories WHERE user_id = $1`, s.schema)
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// SaveKnowledge saves learned knowledge
func (s *Storage) SaveKnowledge(ctx context.Context, knowledge *learning.Knowledge) error {
	metadataJSON, err := json.Marshal(knowledge.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.learning_knowledge (id, topic, content, source, relevance, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, s.schema)

	_, err = s.db.ExecContext(ctx, query,
		knowledge.ID,
		knowledge.Topic,
		knowledge.Content,
		knowledge.Source,
		knowledge.Relevance,
		metadataJSON,
		knowledge.CreatedAt,
	)

	return err
}

// GetKnowledge retrieves knowledge by topic
func (s *Storage) GetKnowledge(ctx context.Context, topic string, limit int) ([]learning.Knowledge, error) {
	query := fmt.Sprintf(`
		SELECT id, topic, content, source, relevance, metadata, created_at
		FROM %s.learning_knowledge
		WHERE topic = $1
		ORDER BY relevance DESC, created_at DESC
		LIMIT $2
	`, s.schema)

	return s.queryKnowledge(ctx, query, topic, limit)
}

// SearchKnowledge searches knowledge by content
func (s *Storage) SearchKnowledge(ctx context.Context, query string, limit int) ([]learning.Knowledge, error) {
	searchQuery := fmt.Sprintf(`
		SELECT id, topic, content, source, relevance, metadata, created_at
		FROM %s.learning_knowledge
		WHERE content ILIKE $1 OR topic ILIKE $1
		ORDER BY relevance DESC, created_at DESC
		LIMIT $2
	`, s.schema)

	return s.queryKnowledge(ctx, searchQuery, "%"+query+"%", limit)
}

func (s *Storage) queryKnowledge(ctx context.Context, query string, arg interface{}, limit int) ([]learning.Knowledge, error) {
	rows, err := s.db.QueryContext(ctx, query, arg, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var knowledgeList []learning.Knowledge
	for rows.Next() {
		var k learning.Knowledge
		var metadataJSON []byte

		err := rows.Scan(
			&k.ID,
			&k.Topic,
			&k.Content,
			&k.Source,
			&k.Relevance,
			&metadataJSON,
			&k.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataJSON, &k.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		knowledgeList = append(knowledgeList, k)
	}

	return knowledgeList, rows.Err()
}

// SaveLearningEvent saves a learning event
func (s *Storage) SaveLearningEvent(ctx context.Context, event *learning.LearningEvent) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.learning_events (id, user_id, event_type, data, occurred_at)
		VALUES ($1, $2, $3, $4, $5)
	`, s.schema)

	_, err = s.db.ExecContext(ctx, query,
		event.ID,
		event.UserID,
		event.EventType,
		dataJSON,
		event.OccurredAt,
	)

	return err
}

// GetLearningEvents retrieves learning events for a user
func (s *Storage) GetLearningEvents(ctx context.Context, userID string, limit int) ([]learning.LearningEvent, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, event_type, data, occurred_at
		FROM %s.learning_events
		WHERE user_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2
	`, s.schema)

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []learning.LearningEvent
	for rows.Next() {
		var event learning.LearningEvent
		var dataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.UserID,
			&event.EventType,
			&dataJSON,
			&event.OccurredAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

// DeleteUserData deletes all data for a user (GDPR compliance)
func (s *Storage) DeleteUserData(ctx context.Context, userID string) error {
	// Delete in order (foreign key constraints)
	queries := []string{
		fmt.Sprintf(`DELETE FROM %s.learning_events WHERE user_id = $1`, s.schema),
		fmt.Sprintf(`DELETE FROM %s.learning_user_memories WHERE user_id = $1`, s.schema),
		fmt.Sprintf(`DELETE FROM %s.learning_user_profiles WHERE user_id = $1`, s.schema),
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query, userID); err != nil {
			return fmt.Errorf("failed to delete user data: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}
