package session

import (
	"context"
	"strings"
	"testing"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
)

func TestNewHistoryProvider(t *testing.T) {
	storage := NewMemoryStorage()
	hp := NewHistoryProvider(storage)
	if hp == nil {
		t.Fatal("expected non-nil history provider")
	}
}

func TestHistoryProvider_NoSession(t *testing.T) {
	storage := NewMemoryStorage()
	hp := NewHistoryProvider(storage)

	_, err := hp.GetHistory(context.Background(), "nonexistent", 5)
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestHistoryProvider_EmptySession(t *testing.T) {
	storage := NewMemoryStorage()
	sess := NewSession("sess-1", "agent-1")
	storage.Create(context.Background(), sess)

	hp := NewHistoryProvider(storage)
	history, err := hp.GetHistory(context.Background(), "sess-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if history != "" {
		t.Errorf("expected empty history for session without runs, got %q", history)
	}
}

func TestHistoryProvider_WithRuns(t *testing.T) {
	storage := NewMemoryStorage()
	sess := NewSession("sess-1", "agent-1")
	sess.AddRun(&agent.RunOutput{RunID: "run-1", Content: "First response"})
	sess.AddRun(&agent.RunOutput{RunID: "run-2", Content: "Second response"})
	storage.Create(context.Background(), sess)

	hp := NewHistoryProvider(storage)
	history, err := hp.GetHistory(context.Background(), "sess-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(history, "Previous Conversation History") {
		t.Error("expected history header")
	}
	if !strings.Contains(history, "First response") {
		t.Error("expected first run content")
	}
	if !strings.Contains(history, "Second response") {
		t.Error("expected second run content")
	}
}

func TestHistoryProvider_MaxRunsLimit(t *testing.T) {
	storage := NewMemoryStorage()
	sess := NewSession("sess-1", "agent-1")
	sess.AddRun(&agent.RunOutput{RunID: "run-1", Content: "Old"})
	sess.AddRun(&agent.RunOutput{RunID: "run-2", Content: "Middle"})
	sess.AddRun(&agent.RunOutput{RunID: "run-3", Content: "Recent"})
	storage.Create(context.Background(), sess)

	hp := NewHistoryProvider(storage)
	history, err := hp.GetHistory(context.Background(), "sess-1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(history, "Old") {
		t.Error("oldest run should be excluded with maxRuns=2")
	}
	if !strings.Contains(history, "Middle") {
		t.Error("expected middle run")
	}
	if !strings.Contains(history, "Recent") {
		t.Error("expected recent run")
	}
}

func TestHistoryProvider_LongContentTruncated(t *testing.T) {
	storage := NewMemoryStorage()
	sess := NewSession("sess-1", "agent-1")
	longContent := strings.Repeat("x", 300)
	sess.AddRun(&agent.RunOutput{RunID: "run-1", Content: longContent})
	storage.Create(context.Background(), sess)

	hp := NewHistoryProvider(storage)
	history, err := hp.GetHistory(context.Background(), "sess-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(history, "...") {
		t.Error("expected truncation indicator")
	}
	// The summary should be ~200 chars + "..." not the full 300
	if strings.Contains(history, strings.Repeat("x", 300)) {
		t.Error("long content should be truncated")
	}
}
