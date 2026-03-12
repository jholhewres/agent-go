package agenttool

import (
	"context"
	"fmt"
	"testing"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// mockModel implements models.Model for testing.
type mockModel struct {
	models.BaseModel
	invokeFunc func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
}

func (m *mockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, req)
	}
	return &types.ModelResponse{Content: "mock response"}, nil
}

func (m *mockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func TestNew(t *testing.T) {
	ag, err := agent.New(agent.Config{
		Name:  "researcher",
		Model: &mockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	tk := New(ag, "Ask the research expert")

	if tk.Name() != "agent_researcher" {
		t.Errorf("expected toolkit name 'agent_researcher', got %q", tk.Name())
	}

	fns := tk.Functions()
	if _, ok := fns["ask_researcher"]; !ok {
		t.Error("expected function 'ask_researcher' to be registered")
	}
}

func TestAgentToolkit_Invoke(t *testing.T) {
	ag, _ := agent.New(agent.Config{
		Name: "helper",
		Model: &mockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			invokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				return &types.ModelResponse{Content: "42"}, nil
			},
		},
	})

	tk := New(ag, "Ask the helper")
	fn := tk.Functions()["ask_helper"]

	result, err := fn.Handler(context.Background(), map[string]interface{}{
		"input": "what is the answer?",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "42" {
		t.Errorf("expected '42', got %v", result)
	}
}

func TestAgentToolkit_MissingInput(t *testing.T) {
	ag, _ := agent.New(agent.Config{
		Name:  "helper",
		Model: &mockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
	})

	tk := New(ag, "Ask the helper")
	fn := tk.Functions()["ask_helper"]

	_, err := fn.Handler(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}

func TestAgentToolkit_EmptyInput(t *testing.T) {
	ag, _ := agent.New(agent.Config{
		Name:  "helper",
		Model: &mockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
	})

	tk := New(ag, "Ask the helper")
	fn := tk.Functions()["ask_helper"]

	_, err := fn.Handler(context.Background(), map[string]interface{}{
		"input": "",
	})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestAgentToolkit_AgentError(t *testing.T) {
	ag, _ := agent.New(agent.Config{
		Name: "failer",
		Model: &mockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			invokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				return nil, fmt.Errorf("model crashed")
			},
		},
	})

	tk := New(ag, "Ask the failer")
	fn := tk.Functions()["ask_failer"]

	_, err := fn.Handler(context.Background(), map[string]interface{}{
		"input": "test",
	})
	if err == nil {
		t.Fatal("expected error from failing agent")
	}
}

func TestAgentToolkit_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ag, _ := agent.New(agent.Config{
		Name:  "helper",
		Model: &mockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
	})

	tk := New(ag, "Ask the helper")
	fn := tk.Functions()["ask_helper"]

	_, err := fn.Handler(ctx, map[string]interface{}{
		"input": "test",
	})
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}
