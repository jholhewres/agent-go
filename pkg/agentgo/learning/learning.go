package learning

import (
	"context"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// LearningMachine is the interface for the learning system
type LearningMachine interface {
	// Learn extracts information from messages and stores it
	Learn(ctx context.Context, userID string, messages []types.Message) error
	
	// GetUserProfile returns the profile for a user
	GetUserProfile(ctx context.Context, userID string) (*UserProfile, error)
	
	// GetUserMemories returns memories for a user
	GetUserMemories(ctx context.Context, userID string, limit int) ([]UserMemory, error)
	
	// GetLearnedKnowledge returns learned knowledge on a topic
	GetLearnedKnowledge(ctx context.Context, topic string, limit int) ([]Knowledge, error)
	
	// DeleteUserData removes all data for a user (GDPR compliance)
	DeleteUserData(ctx context.Context, userID string) error
}

// UserProfile represents a user's profile
type UserProfile struct {
	UserID      string                 `json:"user_id"`
	Name        string                 `json:"name,omitempty"`
	Preferences map[string]interface{} `json:"preferences,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// UserMemory represents a memory associated with a user
type UserMemory struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Content   string                 `json:"content"`
	Type      MemoryType             `json:"type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// MemoryType represents the type of memory
type MemoryType string

const (
	MemoryTypeFact       MemoryType = "fact"
	MemoryTypePreference MemoryType = "preference"
	MemoryTypeContext    MemoryType = "context"
	MemoryTypeSkill      MemoryType = "skill"
)

// Knowledge represents learned knowledge that can be shared across users
type Knowledge struct {
	ID        string                 `json:"id"`
	Topic     string                 `json:"topic"`
	Content   string                 `json:"content"`
	Source    string                 `json:"source"` // "user", "system", "extracted"
	Relevance float64                `json:"relevance"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// LearningEvent represents a learning event for auditing
type LearningEvent struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	EventType  string                 `json:"event_type"`
	Data       map[string]interface{} `json:"data"`
	OccurredAt time.Time              `json:"occurred_at"`
}

// Storage is the interface for learning data persistence
type Storage interface {
	// User Profiles
	SaveUserProfile(ctx context.Context, profile *UserProfile) error
	GetUserProfile(ctx context.Context, userID string) (*UserProfile, error)
	
	// User Memories
	SaveUserMemory(ctx context.Context, memory *UserMemory) error
	GetUserMemories(ctx context.Context, userID string, limit int) ([]UserMemory, error)
	DeleteUserMemories(ctx context.Context, userID string) error
	
	// Learned Knowledge
	SaveKnowledge(ctx context.Context, knowledge *Knowledge) error
	GetKnowledge(ctx context.Context, topic string, limit int) ([]Knowledge, error)
	SearchKnowledge(ctx context.Context, query string, limit int) ([]Knowledge, error)
	
	// Learning Events
	SaveLearningEvent(ctx context.Context, event *LearningEvent) error
	GetLearningEvents(ctx context.Context, userID string, limit int) ([]LearningEvent, error)
	
	// Cleanup
	DeleteUserData(ctx context.Context, userID string) error
	
	// Close closes the storage connection
	Close() error
}
