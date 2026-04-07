package memory

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

const (
	defaultThreshold        = 50
	defaultPreserveLast     = 10
	defaultMaxSummaryTokens = 500
	defaultSummaryTag       = "[Conversation Summary]"
	defaultSummaryPrompt    = "You are a conversation summarizer. Summarize the following conversation messages concisely, preserving key facts, decisions, and context. Be brief but complete."
)

// SummarizingConfig configures the SummarizingMemory behavior.
type SummarizingConfig struct {
	// Inner is the wrapped memory instance (required).
	Inner Memory

	// Model is the LLM used to generate summaries (required).
	Model models.Model

	// Threshold triggers compaction when len(messages) > Threshold (default 50).
	Threshold int

	// PreserveLast always keeps the last N messages verbatim (default 10).
	PreserveLast int

	// MaxSummaryTokens is the upper bound on summary length in tokens (default 500).
	MaxSummaryTokens int

	// SummaryPrompt overrides the default system prompt used during summarization.
	SummaryPrompt string

	// SummaryTag is prepended to the synthesized summary message (default "[Conversation Summary]").
	SummaryTag string
}

// SummarizingMemory wraps any Memory and condenses old messages via an LLM
// when the message count exceeds a configurable threshold.
type SummarizingMemory struct {
	inner Memory
	cfg   SummarizingConfig
	mu    sync.Mutex
}

// NewSummarizingMemory creates a new SummarizingMemory with the given config.
// Returns an error if Inner or Model are not provided.
func NewSummarizingMemory(cfg SummarizingConfig) (*SummarizingMemory, error) {
	if cfg.Inner == nil {
		return nil, fmt.Errorf("SummarizingConfig.Inner is required")
	}
	if cfg.Model == nil {
		return nil, fmt.Errorf("SummarizingConfig.Model is required")
	}
	if cfg.Threshold <= 0 {
		cfg.Threshold = defaultThreshold
	}
	if cfg.PreserveLast <= 0 {
		cfg.PreserveLast = defaultPreserveLast
	}
	if cfg.MaxSummaryTokens <= 0 {
		cfg.MaxSummaryTokens = defaultMaxSummaryTokens
	}
	if cfg.SummaryPrompt == "" {
		cfg.SummaryPrompt = defaultSummaryPrompt
	}
	if cfg.SummaryTag == "" {
		cfg.SummaryTag = defaultSummaryTag
	}
	return &SummarizingMemory{
		inner: cfg.Inner,
		cfg:   cfg,
	}, nil
}

// Add appends a message to the inner memory and triggers compaction when the
// threshold is exceeded. Compaction is synchronous; on failure the original
// state is preserved.
// TODO(future): make compaction asynchronous for lower-latency writes.
func (s *SummarizingMemory) Add(message *types.Message, userID ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.inner.Add(message, userID...)

	if s.inner.Size(userID...) > s.cfg.Threshold {
		if err := s.compact(context.Background(), userID...); err != nil {
			log.Printf("summarizing_memory: compact failed, keeping original state: %v", err)
		}
	}
}

// GetMessages delegates to the inner memory.
func (s *SummarizingMemory) GetMessages(userID ...string) []*types.Message {
	return s.inner.GetMessages(userID...)
}

// Clear delegates to the inner memory.
func (s *SummarizingMemory) Clear(userID ...string) {
	s.inner.Clear(userID...)
}

// Size delegates to the inner memory.
func (s *SummarizingMemory) Size(userID ...string) int {
	return s.inner.Size(userID...)
}

// compact condenses messages older than the last PreserveLast into a single
// System summary message. If the LLM call fails the inner state is untouched.
func (s *SummarizingMemory) compact(ctx context.Context, userID ...string) error {
	all := s.inner.GetMessages(userID...)

	// Not enough messages to summarize after preserving the tail.
	if len(all) <= s.cfg.PreserveLast {
		return nil
	}

	toSummarize := all[:len(all)-s.cfg.PreserveLast]
	keep := all[len(all)-s.cfg.PreserveLast:]

	// Build the summarization prompt content.
	var sb strings.Builder
	for _, msg := range toSummarize {
		sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	promptMessages := []*types.Message{
		types.NewSystemMessage(s.cfg.SummaryPrompt),
		types.NewUserMessage(sb.String()),
	}

	resp, err := s.cfg.Model.Invoke(ctx, &models.InvokeRequest{
		Messages:  promptMessages,
		MaxTokens: s.cfg.MaxSummaryTokens,
	})
	if err != nil {
		return fmt.Errorf("summarizing model invoke: %w", err)
	}

	summaryContent := s.cfg.SummaryTag + " " + resp.Content
	summaryMsg := types.NewSystemMessage(summaryContent)

	// Atomically replace inner memory with [summary, ...keep].
	s.inner.Clear(userID...)
	s.inner.Add(summaryMsg, userID...)
	for _, msg := range keep {
		s.inner.Add(msg, userID...)
	}

	return nil
}
