package otel_test

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/jholhewres/agent-go/pkg/agentgo/hooks"
	agentootel "github.com/jholhewres/agent-go/pkg/agentgo/observability/otel"
)

// TestNewStdoutTracerProvider verifies the stdout provider initialises and shuts down cleanly.
func TestNewStdoutTracerProvider(t *testing.T) {
	tp, shutdown, err := agentootel.NewStdoutTracerProvider()
	if err != nil {
		t.Fatalf("NewStdoutTracerProvider returned error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil TracerProvider")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}
}

// newMemoryProvider builds a TracerProvider backed by an in-memory exporter
// and returns both the provider and the exporter so tests can inspect recorded spans.
func newMemoryProvider() (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sr),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	return tp, sr
}

// TestToolTracingHook_EmitsSpanWithAttributes verifies that ToolTracingHook
// creates a span with the expected attributes on a successful tool call.
func TestToolTracingHook_EmitsSpanWithAttributes(t *testing.T) {
	tp, sr := newMemoryProvider()
	tracer := tp.Tracer("test")

	hooker := agentootel.NewToolTracingHook(tracer)

	input := hooks.NewToolHookInput("agent-1", "call-42", "calculator.add", map[string]interface{}{
		"a": 1, "b": 2,
	})

	ctx := context.Background()
	if err := hooker.OnToolPre(ctx, input); err != nil {
		t.Fatalf("OnToolPre error: %v", err)
	}

	// Simulate the tool result being set.
	input.WithResult(3, nil)

	if err := hooker.OnToolPost(ctx, input); err != nil {
		t.Fatalf("OnToolPost error: %v", err)
	}

	// Flush and inspect.
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	spans := sr.Ended()
	if len(spans) == 0 {
		t.Fatal("expected at least one span, got none")
	}

	span := spans[0]
	if span.Name() != "tool.calculator.add" {
		t.Errorf("span name = %q, want %q", span.Name(), "tool.calculator.add")
	}
	if span.Status().Code != codes.Ok {
		t.Errorf("span status = %v, want Ok", span.Status().Code)
	}

	attrMap := attrsByKey(span)
	if attrMap["tool.name"] != "calculator.add" {
		t.Errorf("tool.name = %q, want %q", attrMap["tool.name"], "calculator.add")
	}
	if attrMap["tool.status"] != "success" {
		t.Errorf("tool.status = %q, want %q", attrMap["tool.status"], "success")
	}
}

// TestToolTracingHook_EmitsSpanWithErrorStatus verifies that a failing tool call
// produces a span with status=Error and tool.status="error".
func TestToolTracingHook_EmitsSpanWithErrorStatus(t *testing.T) {
	tp, sr := newMemoryProvider()
	tracer := tp.Tracer("test")
	hooker := agentootel.NewToolTracingHook(tracer)

	input := hooks.NewToolHookInput("agent-2", "call-99", "http.get", map[string]interface{}{
		"url": "http://example.com",
	})
	ctx := context.Background()
	if err := hooker.OnToolPre(ctx, input); err != nil {
		t.Fatalf("OnToolPre error: %v", err)
	}

	toolErr := context.DeadlineExceeded
	input.WithResult(nil, toolErr)

	if err := hooker.OnToolPost(ctx, input); err != nil {
		t.Fatalf("OnToolPost error: %v", err)
	}

	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	spans := sr.Ended()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	span := spans[0]
	if span.Status().Code != codes.Error {
		t.Errorf("expected Error status, got %v", span.Status().Code)
	}

	attrMap := attrsByKey(span)
	if attrMap["tool.status"] != "error" {
		t.Errorf("tool.status = %q, want %q", attrMap["tool.status"], "error")
	}
}

// TestAgentTracingHook_SuccessSpan verifies that a successful agent run emits
// a span with status=Ok and agent.status="success".
func TestAgentTracingHook_SuccessSpan(t *testing.T) {
	tp, sr := newMemoryProvider()
	tracer := tp.Tracer("test")
	preHook, postHook := agentootel.NewAgentTracingHook(tracer)

	ctx := context.Background()

	preInput := hooks.NewHookInput("what is 2+2?").WithAgentID("agent-abc")
	if err := preHook(ctx, preInput); err != nil {
		t.Fatalf("preHook error: %v", err)
	}

	// Small delay so duration_ms > 0.
	time.Sleep(2 * time.Millisecond)

	postInput := hooks.NewHookInput("what is 2+2?").
		WithAgentID("agent-abc").
		WithOutput("The answer is 4.")
	if err := postHook(ctx, postInput); err != nil {
		t.Fatalf("postHook error: %v", err)
	}

	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	spans := sr.Ended()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	span := spans[0]
	if span.Name() != "agent.run" {
		t.Errorf("span name = %q, want %q", span.Name(), "agent.run")
	}
	if span.Status().Code != codes.Ok {
		t.Errorf("status = %v, want Ok", span.Status().Code)
	}

	attrMap := attrsByKey(span)
	if attrMap["agent.status"] != "success" {
		t.Errorf("agent.status = %q, want %q", attrMap["agent.status"], "success")
	}
}

// TestAgentTracingHook_ErrorSpan verifies that when the post-hook receives
// Metadata["error"], the span is recorded with status=Error.
func TestAgentTracingHook_ErrorSpan(t *testing.T) {
	tp, sr := newMemoryProvider()
	tracer := tp.Tracer("test")
	preHook, postHook := agentootel.NewAgentTracingHook(tracer)

	ctx := context.Background()
	preInput := hooks.NewHookInput("bad input").WithAgentID("agent-err")
	if err := preHook(ctx, preInput); err != nil {
		t.Fatalf("preHook error: %v", err)
	}

	postInput := hooks.NewHookInput("bad input").
		WithAgentID("agent-err").
		WithOutput("").
		WithMetadata(map[string]interface{}{
			"error": "model invocation failed",
		})
	if err := postHook(ctx, postInput); err != nil {
		t.Fatalf("postHook error: %v", err)
	}

	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	spans := sr.Ended()
	if len(spans) == 0 {
		t.Fatal("expected at least one span")
	}

	span := spans[0]
	if span.Status().Code != codes.Error {
		t.Errorf("expected Error status, got %v", span.Status().Code)
	}

	attrMap := attrsByKey(span)
	if attrMap["agent.status"] != "error" {
		t.Errorf("agent.status = %q, want %q", attrMap["agent.status"], "error")
	}
}

// attrsByKey converts a span's attributes to a string→string map for easy assertions.
func attrsByKey(span sdktrace.ReadOnlySpan) map[string]string {
	m := make(map[string]string)
	for _, a := range span.Attributes() {
		m[string(a.Key)] = a.Value.AsString()
	}
	return m
}
