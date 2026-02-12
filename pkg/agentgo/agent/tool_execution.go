package agent

import (
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/hooks"
)

// ToolExecutionStatus represents the outcome of a tool call
// ToolExecutionStatus 表示工具调用的结果状态
type ToolExecutionStatus string

const (
	// ToolExecutionStatusSuccess indicates successful execution
	// ToolExecutionStatusSuccess 表示执行成功
	ToolExecutionStatusSuccess ToolExecutionStatus = "success"
	// ToolExecutionStatusFailed indicates failed execution
	// ToolExecutionStatusFailed 表示执行失败
	ToolExecutionStatusFailed ToolExecutionStatus = "failed"
	// ToolExecutionStatusBlocked indicates execution was blocked by a pre-hook
	// ToolExecutionStatusBlocked 表示执行被前置钩子阻止
	ToolExecutionStatusBlocked ToolExecutionStatus = "blocked"
)

// ToolExecutionSummary captures details about a single tool execution.
// ToolExecutionSummary 捕获单个工具执行的详细信息。
type ToolExecutionSummary struct {
	// ToolCallID is the unique identifier for this tool call
	// ToolCallID 是此工具调用的唯一标识符
	ToolCallID string `json:"tool_call_id"`

	// FunctionName is the name of the executed function
	// FunctionName 是执行的函数名称
	FunctionName string `json:"function_name"`

	// Arguments are the arguments passed to the tool
	// Arguments 是传递给工具的参数
	Arguments map[string]interface{} `json:"arguments,omitempty"`

	// Result is the tool's return value (may be truncated for large results)
	// Result 是工具的返回值（对于大结果可能会被截断）
	Result interface{} `json:"result,omitempty"`

	// Error contains the error message if execution failed
	// Error 包含执行失败时的错误消息
	Error string `json:"error,omitempty"`

	// Status indicates success, failure, or blocked
	// Status 指示成功、失败或阻止
	Status ToolExecutionStatus `json:"status"`

	// StartTime is when execution began
	// StartTime 是执行开始的时间
	StartTime time.Time `json:"start_time"`

	// EndTime is when execution completed
	// EndTime 是执行完成的时间
	EndTime time.Time `json:"end_time"`

	// Duration is the total execution time
	// Duration 是总执行时间
	Duration time.Duration `json:"duration"`

	// Metadata contains additional execution context
	// Metadata 包含额外的执行上下文
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewToolExecutionSummary creates a summary from a tool hook input.
// NewToolExecutionSummary 从工具钩子输入创建摘要。
func NewToolExecutionSummary(input *hooks.ToolHookInput, result interface{}, err error) *ToolExecutionSummary {
	summary := &ToolExecutionSummary{
		ToolCallID:   input.ToolCallID,
		FunctionName: input.FunctionName,
		Arguments:    input.Arguments,
		StartTime:    input.StartTime,
		EndTime:      input.EndTime,
		Duration:     input.Duration,
		Metadata:     make(map[string]interface{}),
	}

	// Copy metadata from hook input
	// 从钩子输入复制元数据
	for k, v := range input.Metadata {
		summary.Metadata[k] = v
	}

	if err != nil {
		summary.Status = ToolExecutionStatusFailed
		summary.Error = err.Error()
	} else {
		summary.Status = ToolExecutionStatusSuccess
		summary.Result = result
	}

	return summary
}

// NewBlockedToolExecutionSummary creates a summary for a blocked tool execution.
// NewBlockedToolExecutionSummary 为被阻止的工具执行创建摘要。
func NewBlockedToolExecutionSummary(input *hooks.ToolHookInput, blockErr error) *ToolExecutionSummary {
	summary := &ToolExecutionSummary{
		ToolCallID:   input.ToolCallID,
		FunctionName: input.FunctionName,
		Arguments:    input.Arguments,
		StartTime:    input.StartTime,
		EndTime:      time.Now(),
		Status:       ToolExecutionStatusBlocked,
		Error:        blockErr.Error(),
		Metadata:     make(map[string]interface{}),
	}
	summary.Duration = summary.EndTime.Sub(summary.StartTime)

	// Copy metadata from hook input
	// 从钩子输入复制元数据
	for k, v := range input.Metadata {
		summary.Metadata[k] = v
	}

	return summary
}

// IsSuccess returns true if execution was successful.
// IsSuccess 如果执行成功，则返回 true。
func (s *ToolExecutionSummary) IsSuccess() bool {
	return s.Status == ToolExecutionStatusSuccess
}

// IsFailed returns true if execution failed.
// IsFailed 如果执行失败，则返回 true。
func (s *ToolExecutionSummary) IsFailed() bool {
	return s.Status == ToolExecutionStatusFailed
}

// IsBlocked returns true if execution was blocked.
// IsBlocked 如果执行被阻止，则返回 true。
func (s *ToolExecutionSummary) IsBlocked() bool {
	return s.Status == ToolExecutionStatusBlocked
}
