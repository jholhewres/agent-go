package session

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

const defaultMaxContentLen = 200

// HistoryProvider implements agent.HistoryProvider by reading session history
// from a Storage backend and formatting it as a context string.
type HistoryProvider struct {
	storage       Storage
	maxContentLen int // max runes per run summary (0 uses defaultMaxContentLen)
}

// HistoryOption configures a HistoryProvider.
type HistoryOption func(*HistoryProvider)

// WithMaxContentLen sets the maximum character length for each run summary.
func WithMaxContentLen(n int) HistoryOption {
	return func(h *HistoryProvider) {
		if n > 0 {
			h.maxContentLen = n
		}
	}
}

// NewHistoryProvider creates a new HistoryProvider wrapping the given Storage.
func NewHistoryProvider(storage Storage, opts ...HistoryOption) *HistoryProvider {
	h := &HistoryProvider{storage: storage, maxContentLen: defaultMaxContentLen}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// GetHistory returns a formatted summary of previous runs in the session.
// maxRuns limits how many recent runs to include.
func (h *HistoryProvider) GetHistory(ctx context.Context, sessionID string, maxRuns int) (string, error) {
	sess, err := h.storage.Get(ctx, sessionID)
	if err != nil {
		return "", err
	}

	if len(sess.Runs) == 0 {
		return "", nil
	}

	// Take the last N runs.
	runs := sess.Runs
	if maxRuns > 0 && len(runs) > maxRuns {
		runs = runs[len(runs)-maxRuns:]
	}

	var parts []string
	parts = append(parts, "[Previous Conversation History]")

	for i, r := range runs {
		summary := truncateUTF8(r.Content, h.maxContentLen)
		parts = append(parts, fmt.Sprintf("Run %d: %s", i+1, summary))
	}

	return strings.Join(parts, "\n"), nil
}

// truncateUTF8 truncates s to at most maxRunes runes, appending "..." if truncated.
func truncateUTF8(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + "..."
}
