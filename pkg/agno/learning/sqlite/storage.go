package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jholhewres/agent-go/pkg/agno/learning"
	_ "modernc.org/sqlite"
)

// Storage implements learning.Storage for SQLite
type Storage struct {
	db *sql.DB
}

// New creates a new SQLite storage for learning
func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &Storage{db: db}

	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return storage, nil
}

// migrate runs database migrations
func (s *Storage) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS learning_user_profiles (
			user_id TEXT PRIMARY KEY,
			name TEXT,
			preferences TEXT DEFAULT '{}',
			context TEXT DEFAULT '{}',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS learning_user_memories (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			content TEXT NOT NULL,
			type TEXT NOT NULL,
			metadata TEXT DEFAULT '{}',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES learning_user_profiles(user_id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS learning_knowledge (
			id TEXT PRIMARY KEY,
			topic TEXT NOT NULL,
			content TEXT NOT NULL,
			source TEXT NOT NULL,
			relevance REAL DEFAULT 0.5,
			metadata TEXT DEFAULT '{}',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS learning_events (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			data TEXT DEFAULT '{}',
			occurred_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_learning_memories_user_id ON learning_user_memories(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_learning_knowledge_topic ON learning_knowledge(topic)`,
	}

	for _, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// SaveUserProfile saves or updates a user profile
func (s *Storage) SaveUserProfile(ctx context.Context, profile *learning.UserProfile) error {
	preferencesJSON, _ := json.Marshal(profile.Preferences)
	contextJSON, _ := json.Marshal(profile.Context)

	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO learning_user_profiles (user_id, name, preferences, context, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, profile.UserID, profile.Name, string(preferencesJSON), string(contextJSON), profile.CreatedAt, profile.UpdatedAt)

	return err
}

// GetUserProfile retrieves a user profile
func (s *Storage) GetUserProfile(ctx context.Context, userID string) (*learning.UserProfile, error) {
	var profile learning.UserProfile
	var preferencesJSON, contextJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, name, preferences, context, created_at, updated_at
		FROM learning_user_profiles WHERE user_id = ?
	`, userID).Scan(&profile.UserID, &profile.Name, &preferencesJSON, &contextJSON, &profile.CreatedAt, &profile.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user profile not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(preferencesJSON), &profile.Preferences)
	json.Unmarshal([]byte(contextJSON), &profile.Context)

	return &profile, nil
}

// SaveUserMemory saves a user memory
func (s *Storage) SaveUserMemory(ctx context.Context, memory *learning.UserMemory) error {
	metadataJSON, _ := json.Marshal(memory.Metadata)

	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO learning_user_memories (id, user_id, content, type, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, memory.ID, memory.UserID, memory.Content, string(memory.Type), string(metadataJSON), memory.CreatedAt)

	return err
}

// GetUserMemories retrieves user memories
func (s *Storage) GetUserMemories(ctx context.Context, userID string, limit int) ([]learning.UserMemory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, content, type, metadata, created_at
		FROM learning_user_memories WHERE user_id = ?
		ORDER BY created_at DESC LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []learning.UserMemory
	for rows.Next() {
		var memory learning.UserMemory
		var memoryType, metadataJSON string

		if err := rows.Scan(&memory.ID, &memory.UserID, &memory.Content, &memoryType, &metadataJSON, &memory.CreatedAt); err != nil {
			return nil, err
		}

		memory.Type = learning.MemoryType(memoryType)
		json.Unmarshal([]byte(metadataJSON), &memory.Metadata)
		memories = append(memories, memory)
	}

	return memories, rows.Err()
}

// DeleteUserMemories deletes all memories for a user
func (s *Storage) DeleteUserMemories(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM learning_user_memories WHERE user_id = ?`, userID)
	return err
}

// SaveKnowledge saves learned knowledge
func (s *Storage) SaveKnowledge(ctx context.Context, knowledge *learning.Knowledge) error {
	metadataJSON, _ := json.Marshal(knowledge.Metadata)

	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO learning_knowledge (id, topic, content, source, relevance, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, knowledge.ID, knowledge.Topic, knowledge.Content, knowledge.Source, knowledge.Relevance, string(metadataJSON), knowledge.CreatedAt)

	return err
}

// GetKnowledge retrieves knowledge by topic
func (s *Storage) GetKnowledge(ctx context.Context, topic string, limit int) ([]learning.Knowledge, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, topic, content, source, relevance, metadata, created_at
		FROM learning_knowledge WHERE topic = ?
		ORDER BY relevance DESC, created_at DESC LIMIT ?
	`, topic, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var knowledgeList []learning.Knowledge
	for rows.Next() {
		var k learning.Knowledge
		var metadataJSON string

		if err := rows.Scan(&k.ID, &k.Topic, &k.Content, &k.Source, &k.Relevance, &metadataJSON, &k.CreatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(metadataJSON), &k.Metadata)
		knowledgeList = append(knowledgeList, k)
	}

	return knowledgeList, rows.Err()
}

// SearchKnowledge searches knowledge by content
func (s *Storage) SearchKnowledge(ctx context.Context, query string, limit int) ([]learning.Knowledge, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, topic, content, source, relevance, metadata, created_at
		FROM learning_knowledge WHERE content LIKE ? OR topic LIKE ?
		ORDER BY relevance DESC, created_at DESC LIMIT ?
	`, "%"+query+"%", "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var knowledgeList []learning.Knowledge
	for rows.Next() {
		var k learning.Knowledge
		var metadataJSON string

		if err := rows.Scan(&k.ID, &k.Topic, &k.Content, &k.Source, &k.Relevance, &metadataJSON, &k.CreatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(metadataJSON), &k.Metadata)
		knowledgeList = append(knowledgeList, k)
	}

	return knowledgeList, rows.Err()
}

// SaveLearningEvent saves a learning event
func (s *Storage) SaveLearningEvent(ctx context.Context, event *learning.LearningEvent) error {
	dataJSON, _ := json.Marshal(event.Data)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO learning_events (id, user_id, event_type, data, occurred_at)
		VALUES (?, ?, ?, ?, ?)
	`, event.ID, event.UserID, event.EventType, string(dataJSON), event.OccurredAt)

	return err
}

// GetLearningEvents retrieves learning events for a user
func (s *Storage) GetLearningEvents(ctx context.Context, userID string, limit int) ([]learning.LearningEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, event_type, data, occurred_at
		FROM learning_events WHERE user_id = ?
		ORDER BY occurred_at DESC LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []learning.LearningEvent
	for rows.Next() {
		var event learning.LearningEvent
		var dataJSON string

		if err := rows.Scan(&event.ID, &event.UserID, &event.EventType, &dataJSON, &event.OccurredAt); err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(dataJSON), &event.Data)
		events = append(events, event)
	}

	return events, rows.Err()
}

// DeleteUserData deletes all data for a user (GDPR compliance)
func (s *Storage) DeleteUserData(ctx context.Context, userID string) error {
	queries := []string{
		`DELETE FROM learning_events WHERE user_id = ?`,
		`DELETE FROM learning_user_memories WHERE user_id = ?`,
		`DELETE FROM learning_user_profiles WHERE user_id = ?`,
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
