package hooks

import (
	"context"
	"fmt"
)

// ApprovalFunc is called to determine if a tool execution should proceed.
// It receives the tool call details and must return true to approve or false to block.
// The function can be blocking (e.g., wait for user input via channel, HTTP callback, etc.)
type ApprovalFunc func(ctx context.Context, input *ToolHookInput) (approved bool, err error)

// ApprovalHook implements ToolHooker to gate tool execution behind an approval function.
// Tools not in ToolFilter (when set) execute without approval.
type ApprovalHook struct {
	approvalFn ApprovalFunc
	toolFilter map[string]struct{} // nil means all tools require approval
}

// NewApprovalHook creates an ApprovalHook.
// If tools are specified, only those tools require approval. If none are specified,
// all tools require approval. Panics if fn is nil.
func NewApprovalHook(fn ApprovalFunc, tools ...string) *ApprovalHook {
	if fn == nil {
		panic("NewApprovalHook: approval function must not be nil")
	}
	h := &ApprovalHook{
		approvalFn: fn,
	}
	if len(tools) > 0 {
		h.toolFilter = make(map[string]struct{}, len(tools))
		for _, t := range tools {
			h.toolFilter[t] = struct{}{}
		}
	}
	return h
}

// OnToolPre checks if approval is needed and blocks execution if denied.
// Returns an error to signal the agent to block the tool call.
func (h *ApprovalHook) OnToolPre(ctx context.Context, input *ToolHookInput) error {
	if !h.requiresApproval(input.FunctionName) {
		return nil
	}

	approved, err := h.approvalFn(ctx, input)
	if err != nil {
		return fmt.Errorf("approval check failed for %s: %w", input.FunctionName, err)
	}
	if !approved {
		return fmt.Errorf("tool execution denied: %s requires approval", input.FunctionName)
	}
	return nil
}

// OnToolPost is a no-op for approval hooks.
func (h *ApprovalHook) OnToolPost(_ context.Context, _ *ToolHookInput) error {
	return nil
}

func (h *ApprovalHook) requiresApproval(functionName string) bool {
	if h.toolFilter == nil {
		return true // all tools require approval
	}
	_, ok := h.toolFilter[functionName]
	return ok
}
