package agent

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/run"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

func TestSpawn_BasicExecution(t *testing.T) {
	child, err := New(Config{
		Name:  "child",
		Model: &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
	})
	if err != nil {
		t.Fatalf("failed to create child agent: %v", err)
	}

	output, err := Spawn(context.Background(), child, "hello")
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	if output.Content != "Mock response" {
		t.Errorf("expected 'Mock response', got %q", output.Content)
	}
}

func TestSpawn_ParentRunIDLinked(t *testing.T) {
	var capturedRunCtx *run.RunContext
	child, _ := New(Config{
		Name: "child",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				rc, ok := run.FromContext(ctx)
				if ok {
					capturedRunCtx = rc
				}
				return &types.ModelResponse{Content: "ok"}, nil
			},
		},
	})

	parentRC := run.NewContext()
	parentRC.EnsureRunID()
	parentRC.UserID = "user-123"
	parentRC.SessionID = "session-456"
	parentCtx := run.WithContext(context.Background(), parentRC)

	_, err := Spawn(parentCtx, child, "test input")
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	if capturedRunCtx == nil {
		t.Fatal("child did not receive RunContext")
	}
	if capturedRunCtx.ParentRunID != parentRC.RunID {
		t.Errorf("expected ParentRunID %q, got %q", parentRC.RunID, capturedRunCtx.ParentRunID)
	}
	if capturedRunCtx.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %q", capturedRunCtx.UserID)
	}
	if capturedRunCtx.SessionID != "session-456" {
		t.Errorf("expected SessionID 'session-456', got %q", capturedRunCtx.SessionID)
	}
	if capturedRunCtx.RunID == parentRC.RunID {
		t.Error("child should have its own RunID, not the parent's")
	}
}

func TestSpawn_NoParentContext(t *testing.T) {
	child, _ := New(Config{
		Name:  "child",
		Model: &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
	})

	output, err := Spawn(context.Background(), child, "test")
	if err != nil {
		t.Fatalf("Spawn should work without parent RunContext: %v", err)
	}
	if output == nil {
		t.Fatal("expected non-nil output")
	}
}

func TestSpawn_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	child, _ := New(Config{
		Name:  "child",
		Model: &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
	})

	_, err := Spawn(ctx, child, "test")
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestSpawn_ChildError(t *testing.T) {
	child, _ := New(Config{
		Name: "child",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				return nil, fmt.Errorf("child model error")
			},
		},
	})

	_, err := Spawn(context.Background(), child, "test")
	if err == nil {
		t.Fatal("expected error from failing child")
	}
}

func TestSpawnAll_ConcurrentExecution(t *testing.T) {
	var callCount atomic.Int32

	makeChild := func(name string) *Agent {
		child, _ := New(Config{
			Name: name,
			Model: &MockModel{
				BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
				InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
					callCount.Add(1)
					return &types.ModelResponse{Content: "response"}, nil
				},
			},
		})
		return child
	}

	children := []SpawnConfig{
		{Agent: makeChild("child-1"), Input: "task 1"},
		{Agent: makeChild("child-2"), Input: "task 2"},
		{Agent: makeChild("child-3"), Input: "task 3"},
	}

	results := SpawnAll(context.Background(), children)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if int(callCount.Load()) != 3 {
		t.Errorf("expected 3 calls, got %d", callCount.Load())
	}

	for i, r := range results {
		if r.Error != nil {
			t.Errorf("result[%d] unexpected error: %v", i, r.Error)
		}
		if r.Output == nil {
			t.Errorf("result[%d] output is nil", i)
		}
	}
}

func TestSpawnAll_PreservesOrder(t *testing.T) {
	children := []SpawnConfig{
		{Agent: mustNewAgent("a"), Input: "input-a"},
		{Agent: mustNewAgent("b"), Input: "input-b"},
		{Agent: mustNewAgent("c"), Input: "input-c"},
	}

	results := SpawnAll(context.Background(), children)

	expectedIDs := []string{"agent-mock", "agent-mock", "agent-mock"}
	for i, r := range results {
		if r.AgentID != expectedIDs[i] {
			t.Errorf("result[%d] AgentID = %q, expected %q", i, r.AgentID, expectedIDs[i])
		}
	}
}

func TestSpawnAll_PartialFailure(t *testing.T) {
	successModel := &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}}
	failModel := &MockModel{
		BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return nil, fmt.Errorf("model error")
		},
	}

	child1, _ := New(Config{Name: "ok", Model: successModel})
	child2, _ := New(Config{Name: "fail", Model: failModel})
	child3, _ := New(Config{Name: "ok2", Model: successModel})

	results := SpawnAll(context.Background(), []SpawnConfig{
		{Agent: child1, Input: "a"},
		{Agent: child2, Input: "b"},
		{Agent: child3, Input: "c"},
	})

	if results[0].Error != nil {
		t.Errorf("result[0] should succeed")
	}
	if results[1].Error == nil {
		t.Errorf("result[1] should fail")
	}
	if results[2].Error != nil {
		t.Errorf("result[2] should succeed")
	}
}

func TestSpawnAll_Empty(t *testing.T) {
	results := SpawnAll(context.Background(), nil)
	if len(results) != 0 {
		t.Errorf("expected empty results for nil input, got %d", len(results))
	}
}

func mustNewAgent(name string) *Agent {
	a, err := New(Config{
		Name:  name,
		Model: &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
	})
	if err != nil {
		panic(err)
	}
	return a
}
