package otel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/jholhewres/agent-go/pkg/agentgo/hooks"
)

// ToolTracingHook implements hooks.ToolHooker and creates an OTel span
// per tool execution. Spans carry the following attributes:
//
//   - tool.name      — function name
//   - tool.args      — JSON-serialised arguments (best-effort)
//   - tool.status    — "success" | "error"
//   - tool.duration_ms — wall-clock duration in milliseconds
//   - agent.id       — owning agent ID
//
// The hook stores the open span in ToolHookInput.Metadata so that OnToolPost
// can find and finish it.
type ToolTracingHook struct {
	tracer trace.Tracer
}

// NewToolTracingHook returns a ToolTracingHook that satisfies hooks.ToolHooker.
func NewToolTracingHook(tracer trace.Tracer) hooks.ToolHooker {
	return &ToolTracingHook{tracer: tracer}
}

// OnToolPre starts a span named "tool.<FunctionName>" and stores it in Metadata.
func (h *ToolTracingHook) OnToolPre(ctx context.Context, input *hooks.ToolHookInput) error {
	_, span := h.tracer.Start(ctx, "tool."+input.FunctionName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("tool.name", input.FunctionName),
			attribute.String("agent.id", input.AgentID),
			attribute.String("tool.call_id", input.ToolCallID),
		),
	)
	input.Metadata["otel.span"] = span
	return nil
}

// OnToolPost ends the span stored by OnToolPre, recording status and duration.
func (h *ToolTracingHook) OnToolPost(ctx context.Context, input *hooks.ToolHookInput) error {
	raw, ok := input.Metadata["otel.span"]
	if !ok {
		return nil
	}
	span, ok := raw.(trace.Span)
	if !ok {
		return nil
	}
	defer span.End()

	durationMs := float64(input.Duration) / float64(time.Millisecond)
	span.SetAttributes(
		attribute.Float64("tool.duration_ms", durationMs),
	)

	if input.Failed() {
		span.SetStatus(codes.Error, input.ResultError.Error())
		span.SetAttributes(attribute.String("tool.status", "error"))
		span.RecordError(input.ResultError)
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.String("tool.status", "success"))
	}

	return nil
}

// agentSpanEntry holds an open span and its start time for an agent run.
type agentSpanEntry struct {
	span      trace.Span
	startTime time.Time
}

// AgentTracingHook holds state for tracing an agent Run.
// Because the PreHook and PostHook are separate hooks.HookFunc values that
// both reference a shared span store (sync.Map keyed by AgentID), parent spans
// propagate correctly across the pre→run→post lifecycle.
type AgentTracingHook struct {
	tracer trace.Tracer
	spans  sync.Map // map[agentID]*agentSpanEntry
}

// NewAgentTracingHook creates a new AgentTracingHook backed by tracer.
// Returns a pre-hook and a post-hook, both of type hooks.HookFunc, ready to be
// appended to agent.Config.PreHooks / PostHooks respectively.
//
//	preHook, postHook := otel.NewAgentTracingHook(tracer)
//	agentCfg := agent.Config{
//	    PreHooks:  []hooks.Hook{preHook},
//	    PostHooks: []hooks.Hook{postHook},
//	}
func NewAgentTracingHook(tracer trace.Tracer) (hooks.HookFunc, hooks.HookFunc) {
	h := &AgentTracingHook{tracer: tracer}
	return h.preHook, h.postHook
}

// preHook starts an "agent.run" span and stores it keyed by AgentID.
func (h *AgentTracingHook) preHook(ctx context.Context, input *hooks.HookInput) error {
	if input.AgentID == "" {
		return nil
	}

	_, span := h.tracer.Start(ctx, "agent.run",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("agent.id", input.AgentID),
			attribute.Int("input.length", len(input.Input)),
		),
	)

	h.spans.Store(input.AgentID, &agentSpanEntry{
		span:      span,
		startTime: time.Now(),
	})
	return nil
}

// postHook ends the span opened by preHook, recording output length and status.
func (h *AgentTracingHook) postHook(ctx context.Context, input *hooks.HookInput) error {
	if input.AgentID == "" {
		return nil
	}

	raw, ok := h.spans.LoadAndDelete(input.AgentID)
	if !ok {
		return nil
	}
	entry, ok := raw.(*agentSpanEntry)
	if !ok {
		return nil
	}
	defer entry.span.End()

	durationMs := float64(time.Since(entry.startTime)) / float64(time.Millisecond)

	// Determine whether this is an error post-hook (output empty + error signalled
	// via input.Metadata["error"]) or a successful completion.
	var runErr error
	if errVal, exists := input.Metadata["error"]; exists {
		if e, ok := errVal.(error); ok {
			runErr = e
		} else if s, ok := errVal.(string); ok && s != "" {
			runErr = fmt.Errorf("%s", s)
		}
	}

	entry.span.SetAttributes(
		attribute.Int("output.length", len(input.Output)),
		attribute.Float64("agent.duration_ms", durationMs),
	)

	if runErr != nil {
		entry.span.SetStatus(codes.Error, runErr.Error())
		entry.span.SetAttributes(attribute.String("agent.status", "error"))
		entry.span.RecordError(runErr)
	} else {
		entry.span.SetStatus(codes.Ok, "")
		entry.span.SetAttributes(attribute.String("agent.status", "success"))
	}

	return nil
}
