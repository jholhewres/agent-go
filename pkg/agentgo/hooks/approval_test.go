package hooks

import (
	"context"
	"errors"
	"testing"
)

func TestApprovalHook_Approved(t *testing.T) {
	hook := NewApprovalHook(func(ctx context.Context, input *ToolHookInput) (bool, error) {
		return true, nil
	})

	input := NewToolHookInput("agent-1", "call-1", "delete_file", map[string]interface{}{"path": "/tmp/x"})
	err := hook.OnToolPre(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error on approval, got: %v", err)
	}
}

func TestApprovalHook_Denied(t *testing.T) {
	hook := NewApprovalHook(func(ctx context.Context, input *ToolHookInput) (bool, error) {
		return false, nil
	})

	input := NewToolHookInput("agent-1", "call-1", "delete_file", map[string]interface{}{})
	err := hook.OnToolPre(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when approval denied")
	}
}

func TestApprovalHook_ApprovalFuncError(t *testing.T) {
	hook := NewApprovalHook(func(ctx context.Context, input *ToolHookInput) (bool, error) {
		return false, errors.New("network error")
	})

	input := NewToolHookInput("agent-1", "call-1", "delete_file", map[string]interface{}{})
	err := hook.OnToolPre(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when approval func fails")
	}
}

func TestApprovalHook_ToolFilter_FilteredTool(t *testing.T) {
	called := false
	hook := NewApprovalHook(func(ctx context.Context, input *ToolHookInput) (bool, error) {
		called = true
		return true, nil
	}, "delete_file", "send_email")

	input := NewToolHookInput("agent-1", "call-1", "delete_file", map[string]interface{}{})
	err := hook.OnToolPre(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("approval func should have been called for filtered tool")
	}
}

func TestApprovalHook_ToolFilter_UnfilteredTool(t *testing.T) {
	called := false
	hook := NewApprovalHook(func(ctx context.Context, input *ToolHookInput) (bool, error) {
		called = true
		return false, nil
	}, "delete_file", "send_email")

	input := NewToolHookInput("agent-1", "call-1", "calculator_add", map[string]interface{}{})
	err := hook.OnToolPre(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error for unfiltered tool: %v", err)
	}
	if called {
		t.Error("approval func should NOT have been called for unfiltered tool")
	}
}

func TestApprovalHook_NoFilter_AllToolsRequireApproval(t *testing.T) {
	callCount := 0
	hook := NewApprovalHook(func(ctx context.Context, input *ToolHookInput) (bool, error) {
		callCount++
		return true, nil
	})

	tools := []string{"calculator_add", "delete_file", "send_email", "http_get"}
	for _, tool := range tools {
		input := NewToolHookInput("agent-1", "call-1", tool, map[string]interface{}{})
		err := hook.OnToolPre(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error for tool %s: %v", tool, err)
		}
	}

	if callCount != len(tools) {
		t.Errorf("expected %d calls, got %d", len(tools), callCount)
	}
}

func TestApprovalHook_OnToolPost_NoOp(t *testing.T) {
	hook := NewApprovalHook(func(ctx context.Context, input *ToolHookInput) (bool, error) {
		return false, nil // would deny in pre
	})

	input := NewToolHookInput("agent-1", "call-1", "delete_file", map[string]interface{}{})
	input.WithResult("ok", nil)
	err := hook.OnToolPost(context.Background(), input)
	if err != nil {
		t.Fatalf("OnToolPost should be no-op, got: %v", err)
	}
}

func TestApprovalHook_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	hook := NewApprovalHook(func(ctx context.Context, input *ToolHookInput) (bool, error) {
		return false, ctx.Err()
	})

	input := NewToolHookInput("agent-1", "call-1", "delete_file", map[string]interface{}{})
	err := hook.OnToolPre(ctx, input)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestApprovalHook_ReceivesCorrectInput(t *testing.T) {
	var capturedInput *ToolHookInput
	hook := NewApprovalHook(func(ctx context.Context, input *ToolHookInput) (bool, error) {
		capturedInput = input
		return true, nil
	})

	args := map[string]interface{}{"path": "/tmp/test", "force": true}
	input := NewToolHookInput("agent-42", "call-99", "delete_file", args)
	_ = hook.OnToolPre(context.Background(), input)

	if capturedInput.AgentID != "agent-42" {
		t.Errorf("expected agent-42, got %s", capturedInput.AgentID)
	}
	if capturedInput.FunctionName != "delete_file" {
		t.Errorf("expected delete_file, got %s", capturedInput.FunctionName)
	}
	if capturedInput.Arguments["path"] != "/tmp/test" {
		t.Errorf("expected path /tmp/test, got %v", capturedInput.Arguments["path"])
	}
}
