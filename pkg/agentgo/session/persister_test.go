package session

import (
	"context"
	"testing"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
)

func TestNewPersister(t *testing.T) {
	storage := NewMemoryStorage()
	p := NewPersister(storage)
	if p == nil {
		t.Fatal("expected non-nil persister")
	}
}

func TestPersister_PersistRun_NewSession(t *testing.T) {
	storage := NewMemoryStorage()
	p := NewPersister(storage)

	output := &agent.RunOutput{
		RunID:   "run-1",
		Content: "hello world",
	}

	err := p.PersistRun(context.Background(), "sess-1", "agent-1", "user-1", output)
	if err != nil {
		t.Fatalf("PersistRun failed: %v", err)
	}

	// Verify session was created
	sess, err := storage.Get(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("Get session failed: %v", err)
	}
	if sess.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got %q", sess.AgentID)
	}
	if sess.UserID != "user-1" {
		t.Errorf("expected user ID 'user-1', got %q", sess.UserID)
	}
	if len(sess.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(sess.Runs))
	}
	if sess.Runs[0].Content != "hello world" {
		t.Errorf("expected content 'hello world', got %q", sess.Runs[0].Content)
	}
}

func TestPersister_PersistRun_ExistingSession(t *testing.T) {
	storage := NewMemoryStorage()
	p := NewPersister(storage)

	// Create initial session with a run.
	err := p.PersistRun(context.Background(), "sess-1", "agent-1", "user-1", &agent.RunOutput{
		RunID:   "run-1",
		Content: "first",
	})
	if err != nil {
		t.Fatalf("first PersistRun failed: %v", err)
	}

	// Add second run to same session.
	err = p.PersistRun(context.Background(), "sess-1", "agent-1", "user-1", &agent.RunOutput{
		RunID:   "run-2",
		Content: "second",
	})
	if err != nil {
		t.Fatalf("second PersistRun failed: %v", err)
	}

	sess, _ := storage.Get(context.Background(), "sess-1")
	if len(sess.Runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(sess.Runs))
	}
	if sess.Runs[1].Content != "second" {
		t.Errorf("expected second run content 'second', got %q", sess.Runs[1].Content)
	}
}

func TestPersister_PersistRun_ContextCancellation(t *testing.T) {
	storage := NewMemoryStorage()
	p := NewPersister(storage)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.PersistRun(ctx, "sess-1", "agent-1", "user-1", &agent.RunOutput{
		Content: "test",
	})
	// Cancelled context should cause error from storage
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}
