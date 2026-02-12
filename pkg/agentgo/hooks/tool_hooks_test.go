package hooks

import (
	"context"
	"errors"
	"testing"
)

func TestToolHookInput_Phases(t *testing.T) {
	input := NewToolHookInput("agent-1", "call-1", "test_func", map[string]interface{}{"arg": "value"})

	if !input.IsPre() {
		t.Error("Expected IsPre() to be true for new input")
	}
	if input.IsPost() {
		t.Error("Expected IsPost() to be false for new input")
	}

	input.WithResult("result", nil)

	if input.IsPre() {
		t.Error("Expected IsPre() to be false after WithResult")
	}
	if !input.IsPost() {
		t.Error("Expected IsPost() to be true after WithResult")
	}
}

func TestToolHookInput_Success(t *testing.T) {
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)

	if input.Success() {
		t.Error("Expected Success() to be false for pre-phase input")
	}

	input.WithResult("result", nil)
	if !input.Success() {
		t.Error("Expected Success() to be true for successful execution")
	}

	input = NewToolHookInput("agent-1", "call-1", "test_func", nil)
	input.WithResult(nil, errors.New("test error"))
	if input.Success() {
		t.Error("Expected Success() to be false for failed execution")
	}
}

func TestToolHookInput_Failed(t *testing.T) {
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)
	input.WithResult(nil, errors.New("test error"))

	if !input.Failed() {
		t.Error("Expected Failed() to be true for failed execution")
	}
}

func TestToolHookInput_WithMetadata(t *testing.T) {
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)
	input.WithMetadata(map[string]interface{}{"key": "value"})

	if input.Metadata["key"] != "value" {
		t.Error("Expected metadata to be set")
	}
}

// Mock ToolHooker implementation
type mockToolHooker struct {
	preErr  error
	postErr error
	preCnt  int
	postCnt int
}

func (m *mockToolHooker) OnToolPre(ctx context.Context, input *ToolHookInput) error {
	m.preCnt++
	return m.preErr
}

func (m *mockToolHooker) OnToolPost(ctx context.Context, input *ToolHookInput) error {
	m.postCnt++
	return m.postErr
}

func TestExecuteToolPreHook_WithToolHooker(t *testing.T) {
	hooker := &mockToolHooker{preErr: nil}
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)

	err := ExecuteToolPreHook(context.Background(), hooker, input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if hooker.preCnt != 1 {
		t.Errorf("Expected preCnt to be 1, got %d", hooker.preCnt)
	}
}

func TestExecuteToolPreHook_WithToolHooker_Blocking(t *testing.T) {
	hooker := &mockToolHooker{preErr: errors.New("blocked")}
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)

	err := ExecuteToolPreHook(context.Background(), hooker, input)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "blocked" {
		t.Errorf("Expected 'blocked', got %v", err)
	}
}

func TestExecuteToolPreHook_WithFunction(t *testing.T) {
	called := false
	hook := ToolHookFunc(func(ctx context.Context, input *ToolHookInput) error {
		called = true
		return nil
	})
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)

	err := ExecuteToolPreHook(context.Background(), hook, input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestExecuteToolPostHook_WithToolHooker(t *testing.T) {
	hooker := &mockToolHooker{postErr: nil}
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)
	input.WithResult("result", nil)

	err := ExecuteToolPostHook(context.Background(), hooker, input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if hooker.postCnt != 1 {
		t.Errorf("Expected postCnt to be 1, got %d", hooker.postCnt)
	}
}

func TestExecuteToolPreHooks_AllPass(t *testing.T) {
	hooks := []ToolHook{
		ToolHookFunc(func(ctx context.Context, input *ToolHookInput) error { return nil }),
		ToolHookFunc(func(ctx context.Context, input *ToolHookInput) error { return nil }),
	}
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)

	err := ExecuteToolPreHooks(context.Background(), hooks, input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteToolPreHooks_FirstFails(t *testing.T) {
	hooks := []ToolHook{
		ToolHookFunc(func(ctx context.Context, input *ToolHookInput) error { return errors.New("blocked") }),
		ToolHookFunc(func(ctx context.Context, input *ToolHookInput) error { return nil }),
	}
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)

	err := ExecuteToolPreHooks(context.Background(), hooks, input)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestExecuteToolPostHooks_AllPass(t *testing.T) {
	hooks := []ToolHook{
		ToolHookFunc(func(ctx context.Context, input *ToolHookInput) error { return nil }),
		ToolHookFunc(func(ctx context.Context, input *ToolHookInput) error { return nil }),
	}
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)
	input.WithResult("result", nil)

	err := ExecuteToolPostHooks(context.Background(), hooks, input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteToolPostHooks_FirstFails(t *testing.T) {
	hooks := []ToolHook{
		ToolHookFunc(func(ctx context.Context, input *ToolHookInput) error { return errors.New("error") }),
		ToolHookFunc(func(ctx context.Context, input *ToolHookInput) error { return nil }),
	}
	input := NewToolHookInput("agent-1", "call-1", "test_func", nil)
	input.WithResult("result", nil)

	err := ExecuteToolPostHooks(context.Background(), hooks, input)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
