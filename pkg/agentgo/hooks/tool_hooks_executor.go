package hooks

import (
	"context"
	"fmt"
)

// ExecuteToolPreHook executes a single tool pre-hook.
// Returns error if hook blocks execution.
// ExecuteToolPreHook 执行单个工具前置钩子。
// 如果钩子阻止执行，则返回错误。
func ExecuteToolPreHook(ctx context.Context, hook ToolHook, input *ToolHookInput) error {
	// Check if it's a ToolHooker interface
	// 检查是否是 ToolHooker 接口
	if hooker, ok := hook.(ToolHooker); ok {
		return hooker.OnToolPre(ctx, input)
	}

	// Check if it's a ToolHookFunc
	// 检查是否是 ToolHookFunc
	if hookFunc, ok := hook.(ToolHookFunc); ok {
		return hookFunc(ctx, input)
	}

	// Check if it's a function with the right signature
	// 检查是否是具有正确签名的函数
	if fn, ok := hook.(func(context.Context, *ToolHookInput) error); ok {
		return fn(ctx, input)
	}

	return nil
}

// ExecuteToolPostHook executes a single tool post-hook.
// ExecuteToolPostHook 执行单个工具后置钩子。
func ExecuteToolPostHook(ctx context.Context, hook ToolHook, input *ToolHookInput) error {
	// Check if it's a ToolHooker interface
	// 检查是否是 ToolHooker 接口
	if hooker, ok := hook.(ToolHooker); ok {
		return hooker.OnToolPost(ctx, input)
	}

	// Check if it's a ToolHookFunc
	// 检查是否是 ToolHookFunc
	if hookFunc, ok := hook.(ToolHookFunc); ok {
		return hookFunc(ctx, input)
	}

	// Check if it's a function with the right signature
	// 检查是否是具有正确签名的函数
	if fn, ok := hook.(func(context.Context, *ToolHookInput) error); ok {
		return fn(ctx, input)
	}

	return nil
}

// ExecuteToolPreHooks executes all pre-tool hooks in order.
// Stops and returns error if any hook blocks execution.
// ExecuteToolPreHooks 按顺序执行所有前置工具钩子。
// 如果任何钩子阻止执行，则停止并返回错误。
func ExecuteToolPreHooks(ctx context.Context, hooks []ToolHook, input *ToolHookInput) error {
	for i, hook := range hooks {
		if err := ExecuteToolPreHook(ctx, hook, input); err != nil {
			return fmt.Errorf("tool pre-hook %d blocked execution: %w", i, err)
		}
	}
	return nil
}

// ExecuteToolPostHooks executes all post-tool hooks in order.
// Returns first error encountered.
// ExecuteToolPostHooks 按顺序执行所有后置工具钩子。
// 返回遇到的第一个错误。
func ExecuteToolPostHooks(ctx context.Context, hooks []ToolHook, input *ToolHookInput) error {
	for i, hook := range hooks {
		if err := ExecuteToolPostHook(ctx, hook, input); err != nil {
			return fmt.Errorf("tool post-hook %d error: %w", i, err)
		}
	}
	return nil
}
