package learning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jholhewres/agent-go/pkg/agno/types"
)

// Machine is the default implementation of LearningMachine
type Machine struct {
	storage   Storage
	extractor *Extractor
	enabled   bool
}

// NewMachine creates a new learning machine
func NewMachine(storage Storage) (*Machine, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage is required")
	}

	return &Machine{
		storage:   storage,
		extractor: NewExtractor(),
		enabled:   true,
	}, nil
}

// Learn extracts information from messages and stores it
func (m *Machine) Learn(ctx context.Context, userID string, messages []types.Message) error {
	if !m.enabled || userID == "" {
		return nil
	}

	// Ensure user profile exists
	profile, err := m.storage.GetUserProfile(ctx, userID)
	if err != nil {
		// Create new profile
		profile = &UserProfile{
			UserID:      userID,
			Preferences: make(map[string]interface{}),
			Context:     make(map[string]interface{}),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := m.storage.SaveUserProfile(ctx, profile); err != nil {
			return fmt.Errorf("failed to create user profile: %w", err)
		}
	}

	// Extract memories from messages
	memories := m.extractor.ExtractMemories(userID, messages)
	for _, memory := range memories {
		if err := m.storage.SaveUserMemory(ctx, &memory); err != nil {
			return fmt.Errorf("failed to save memory: %w", err)
		}
	}

	// Extract knowledge from messages
	knowledge := m.extractor.ExtractKnowledge(messages)
	for _, k := range knowledge {
		if err := m.storage.SaveKnowledge(ctx, &k); err != nil {
			return fmt.Errorf("failed to save knowledge: %w", err)
		}
	}

	// Update profile timestamp
	profile.UpdatedAt = time.Now()
	if err := m.storage.SaveUserProfile(ctx, profile); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	// Log learning event
	event := &LearningEvent{
		ID:     uuid.New().String(),
		UserID: userID,
		EventType: "learning_session",
		Data: map[string]interface{}{
			"message_count":  len(messages),
			"memories_count": len(memories),
			"knowledge_count": len(knowledge),
		},
		OccurredAt: time.Now(),
	}
	if err := m.storage.SaveLearningEvent(ctx, event); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to save learning event: %v\n", err)
	}

	return nil
}

// GetUserProfile returns the profile for a user
func (m *Machine) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	return m.storage.GetUserProfile(ctx, userID)
}

// GetUserMemories returns memories for a user
func (m *Machine) GetUserMemories(ctx context.Context, userID string, limit int) ([]UserMemory, error) {
	if limit <= 0 {
		limit = 10
	}
	return m.storage.GetUserMemories(ctx, userID, limit)
}

// GetLearnedKnowledge returns learned knowledge on a topic
func (m *Machine) GetLearnedKnowledge(ctx context.Context, topic string, limit int) ([]Knowledge, error) {
	if limit <= 0 {
		limit = 10
	}
	
	topic = strings.TrimSpace(strings.ToLower(topic))
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}
	
	return m.storage.GetKnowledge(ctx, topic, limit)
}

// DeleteUserData removes all data for a user (GDPR compliance)
func (m *Machine) DeleteUserData(ctx context.Context, userID string) error {
	return m.storage.DeleteUserData(ctx, userID)
}

// Enable enables the learning system
func (m *Machine) Enable() {
	m.enabled = true
}

// Disable disables the learning system
func (m *Machine) Disable() {
	m.enabled = false
}

// IsEnabled returns whether the learning system is enabled
func (m *Machine) IsEnabled() bool {
	return m.enabled
}

// Close closes the learning machine and its storage
func (m *Machine) Close() error {
	return m.storage.Close()
}
