package session

import (
	"context"
	"errors"
	"sync"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
)

// Persister adapts a session.Storage to the agent.SessionPersister interface,
// allowing the agent to persist run outputs without importing the session package.
type Persister struct {
	storage Storage
	mu      sync.Mutex // Serializes concurrent PersistRun calls to prevent lost updates.
}

// NewPersister creates a new Persister wrapping the given Storage.
func NewPersister(storage Storage) *Persister {
	return &Persister{storage: storage}
}

// PersistRun saves a completed run output to the session.
// If the session doesn't exist, it creates a new one.
// Concurrent calls are serialized via a mutex to prevent lost updates.
func (p *Persister) PersistRun(ctx context.Context, sessionID, agentID, userID string, output *agent.RunOutput) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	sess, err := p.storage.Get(ctx, sessionID)
	if err != nil {
		if !errors.Is(err, ErrSessionNotFound) {
			return err // Real storage error — propagate it.
		}
		// Session not found — create a new one.
		sess = NewSession(sessionID, agentID)
		sess.UserID = userID
		sess.AddRun(output)
		return p.storage.Create(ctx, sess)
	}

	sess.AddRun(output)
	return p.storage.Update(ctx, sess)
}
