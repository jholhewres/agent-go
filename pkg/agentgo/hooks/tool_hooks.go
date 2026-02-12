package hooks

import (
	"context"
	"time"
)

// ToolHookPhase indicates the phase of tool execution
// ToolHookPhase 表示工具执行的阶段
type ToolHookPhase string

const (
	// ToolHookPhasePre is the phase before tool execution
	// ToolHookPhasePre 是工具执行前的阶段
	ToolHookPhasePre ToolHookPhase = "pre"
	// ToolHookPhasePost is the phase after tool execution
	// ToolHookPhasePost 是工具执行后的阶段
	ToolHookPhasePost ToolHookPhase = "post"
)

// ToolHookInput contains data passed to tool hooks.
// ToolHookInput 包含传递给工具钩子的数据。
type ToolHookInput struct {
	// Phase indicates whether this is a pre or post hook
	// Phase 表示这是一个前置还是后置钩子
	Phase ToolHookPhase

	// AgentID is the ID of the agent executing the tool
	// AgentID 是执行工具的代理 ID
	AgentID string

	// ToolCallID is the unique ID of this tool call
	// ToolCallID 是此工具调用的唯一 ID
	ToolCallID string

	// FunctionName is the name of the tool function
	// FunctionName 是工具函数的名称
	FunctionName string

	// Arguments are the parsed arguments passed to the tool
	// Arguments 是传递给工具的解析后的参数
	Arguments map[string]interface{}

	// Result is the tool execution result (only available in post hooks)
	// Result 是工具执行结果（仅在 post 钩子中可用）
	Result interface{}

	// ResultError is any error from tool execution (only in post hooks)
	// ResultError 是工具执行的任何错误（仅在 post 钩子中可用）
	ResultError error

	// StartTime is when the tool execution began
	// StartTime 是工具执行开始的时间
	StartTime time.Time

	// EndTime is when the tool execution completed (only in post hooks)
	// EndTime 是工具执行完成的时间（仅在 post 钩子中可用）
	EndTime time.Time

	// Duration is the total execution time (only in post hooks)
	// Duration 是总执行时间（仅在 post 钩子中可用）
	Duration time.Duration

	// Metadata allows hooks to pass data between pre and post phases
	// Metadata 允许钩子在 pre 和 post 阶段之间传递数据
	Metadata map[string]interface{}
}

// ToolHookFunc is the function signature for tool hooks.
// ToolHookFunc 是工具钩子的函数签名。
type ToolHookFunc func(ctx context.Context, input *ToolHookInput) error

// ToolHook can be either a ToolHookFunc or implement ToolHooker interface.
// ToolHook 可以是 ToolHookFunc 或实现 ToolHooker 接口。
type ToolHook interface{}

// ToolHooker is an interface for structured tool hooks.
// ToolHooker 是结构化工具钩子的接口。
type ToolHooker interface {
	// OnToolPre is called before tool execution.
	// Return error to block execution.
	// OnToolPre 在工具执行前调用。
	// 返回错误以阻止执行。
	OnToolPre(ctx context.Context, input *ToolHookInput) error

	// OnToolPost is called after tool execution.
	// Can transform result or log outcomes.
	// OnToolPost 在工具执行后调用。
	// 可以转换结果或记录输出。
	OnToolPost(ctx context.Context, input *ToolHookInput) error
}

// NewToolHookInput creates a new ToolHookInput for pre-execution.
// NewToolHookInput 为 pre-execution 创建一个新的 ToolHookInput。
func NewToolHookInput(agentID, toolCallID, functionName string, args map[string]interface{}) *ToolHookInput {
	return &ToolHookInput{
		Phase:        ToolHookPhasePre,
		AgentID:      agentID,
		ToolCallID:   toolCallID,
		FunctionName: functionName,
		Arguments:    args,
		StartTime:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}
}

// WithResult adds result information for post hooks.
// WithResult 为 post 钩子添加结果信息。
func (thi *ToolHookInput) WithResult(result interface{}, err error) *ToolHookInput {
	thi.Phase = ToolHookPhasePost
	thi.Result = result
	thi.ResultError = err
	thi.EndTime = time.Now()
	thi.Duration = thi.EndTime.Sub(thi.StartTime)
	return thi
}

// WithMetadata adds metadata to the hook input.
// WithMetadata 向钩子输入添加元数据。
func (thi *ToolHookInput) WithMetadata(metadata map[string]interface{}) *ToolHookInput {
	if thi.Metadata == nil {
		thi.Metadata = make(map[string]interface{})
	}
	for k, v := range metadata {
		thi.Metadata[k] = v
	}
	return thi
}

// IsPre returns true if this is a pre-execution hook.
// IsPre 如果这是一个 pre-execution 钩子，则返回 true。
func (thi *ToolHookInput) IsPre() bool {
	return thi.Phase == ToolHookPhasePre
}

// IsPost returns true if this is a post-execution hook.
// IsPost 如果这是一个 post-execution 钩子，则返回 true。
func (thi *ToolHookInput) IsPost() bool {
	return thi.Phase == ToolHookPhasePost
}

// Success returns true if the tool execution was successful.
// Success 如果工具执行成功，则返回 true。
func (thi *ToolHookInput) Success() bool {
	return thi.IsPost() && thi.ResultError == nil
}

// Failed returns true if the tool execution failed.
// Failed 如果工具执行失败，则返回 true。
func (thi *ToolHookInput) Failed() bool {
	return thi.IsPost() && thi.ResultError != nil
}
