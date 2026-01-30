package learning

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jholhewres/agent-go/pkg/agno/types"
)

// Extractor extracts learning data from messages
type Extractor struct {
	// Future: could use LLM to extract structured information
}

// NewExtractor creates a new extractor
func NewExtractor() *Extractor {
	return &Extractor{}
}

// ExtractMemories extracts memories from messages
func (e *Extractor) ExtractMemories(userID string, messages []types.Message) []UserMemory {
	var memories []UserMemory

	for _, msg := range messages {
		if msg.Role != types.MessageRoleUser {
			continue
		}

		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}

		// Extract preferences (simple heuristic-based extraction)
		if e.containsPreference(content) {
			memories = append(memories, UserMemory{
				ID:        uuid.New().String(),
				UserID:    userID,
				Content:   content,
				Type:      MemoryTypePreference,
				Metadata:  map[string]interface{}{"extracted_at": time.Now()},
				CreatedAt: time.Now(),
			})
		}

		// Extract factual information
		if e.containsFact(content) {
			memories = append(memories, UserMemory{
				ID:        uuid.New().String(),
				UserID:    userID,
				Content:   content,
				Type:      MemoryTypeFact,
				Metadata:  map[string]interface{}{"extracted_at": time.Now()},
				CreatedAt: time.Now(),
			})
		}

		// Store as context if it's substantial
		if len(content) > 50 {
			memories = append(memories, UserMemory{
				ID:        uuid.New().String(),
				UserID:    userID,
				Content:   content,
				Type:      MemoryTypeContext,
				Metadata:  map[string]interface{}{"extracted_at": time.Now()},
				CreatedAt: time.Now(),
			})
		}
	}

	return memories
}

// ExtractKnowledge extracts transferable knowledge from messages
func (e *Extractor) ExtractKnowledge(messages []types.Message) []Knowledge {
	var knowledge []Knowledge

	for _, msg := range messages {
		if msg.Role != types.MessageRoleAssistant {
			continue
		}

		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}

		// Extract knowledge if message contains explanations or factual info
		if e.containsKnowledge(content) {
			topic := e.extractTopic(content)
			knowledge = append(knowledge, Knowledge{
				ID:        uuid.New().String(),
				Topic:     topic,
				Content:   content,
				Source:    "extracted",
				Relevance: 0.8, // Default relevance
				Metadata:  map[string]interface{}{"extracted_at": time.Now()},
				CreatedAt: time.Now(),
			})
		}
	}

	return knowledge
}

// containsPreference checks if content contains preference indicators
func (e *Extractor) containsPreference(content string) bool {
	lower := strings.ToLower(content)
	preferenceKeywords := []string{
		"i prefer", "i like", "i love", "i enjoy",
		"my favorite", "i usually", "i always",
		"i don't like", "i hate", "i dislike",
	}

	for _, keyword := range preferenceKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// containsFact checks if content contains factual information
func (e *Extractor) containsFact(content string) bool {
	lower := strings.ToLower(content)
	factKeywords := []string{
		"i am", "my name is", "i work at",
		"i'm from", "i live in", "my job is",
		"i have", "i own", "i use",
	}

	for _, keyword := range factKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// containsKnowledge checks if content contains transferable knowledge
func (e *Extractor) containsKnowledge(content string) bool {
	lower := strings.ToLower(content)
	knowledgeKeywords := []string{
		"here's how", "to do this", "you can",
		"the way to", "this works by", "this means",
		"in other words", "essentially", "basically",
		"for example", "such as", "definition",
	}

	for _, keyword := range knowledgeKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	// Knowledge is usually longer explanations
	return len(content) > 100
}

// extractTopic extracts a topic from content (simple heuristic)
func (e *Extractor) extractTopic(content string) string {
	// Take first significant words as topic
	words := strings.Fields(content)
	if len(words) == 0 {
		return "general"
	}

	// Take up to 5 words
	topicWords := []string{}
	for i := 0; i < len(words) && i < 5; i++ {
		word := strings.ToLower(strings.Trim(words[i], ".,!?;:"))
		if len(word) > 3 { // Skip small words
			topicWords = append(topicWords, word)
		}
	}

	if len(topicWords) == 0 {
		return "general"
	}

	return strings.Join(topicWords, "_")
}
