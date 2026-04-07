// Example: Agent with OpenTelemetry Tracing (stdout exporter)
// Demonstrates how to attach OTel tracing hooks to an agent without modifying agent.go.
// Spans are emitted to stdout — no collector required.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/hooks"
	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	agentootel "github.com/jholhewres/agent-go/pkg/agentgo/observability/otel"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/calculator"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"

	"github.com/jholhewres/agent-go/pkg/agentgo/models/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY is not set — set it to run this example.")
		fmt.Fprintln(os.Stderr, "The example will still compile and demonstrate hook wiring.")
		runDemo(nil)
		return
	}

	model, err := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create model: %v\n", err)
		os.Exit(1)
	}
	runDemo(model)
}

func runDemo(model models.Model) {
	ctx := context.Background()

	// 1. Create a stdout tracer provider (no collector needed).
	tp, shutdown, err := agentootel.NewStdoutTracerProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create tracer provider: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "tracer provider shutdown error: %v\n", err)
		}
	}()

	tracer := tp.Tracer("otlp_tracing_agent")

	// 2. Create tracing hooks.
	toolHook := agentootel.NewToolTracingHook(tracer)
	preHook, postHook := agentootel.NewAgentTracingHook(tracer)

	if model == nil {
		// No API key — demonstrate that hook wiring compiles and hooks are callable.
		fmt.Println("No model configured. Verifying hook wiring only...")

		// Simulate a tool hook cycle with a fake input.
		ti := hooks.NewToolHookInput("demo-agent", "call-1", "calculator.add", map[string]interface{}{
			"a": 2, "b": 3,
		})
		toolCtx := context.Background()
		_ = toolHook.OnToolPre(toolCtx, ti)
		ti.WithResult(5, nil)
		_ = toolHook.OnToolPost(toolCtx, ti)

		// Simulate an agent run cycle.
		hi := hooks.NewHookInput("2+3").WithAgentID("demo-agent")
		_ = preHook(ctx, hi)
		hi2 := hooks.NewHookInput("2+3").WithAgentID("demo-agent").WithOutput("5")
		_ = postHook(ctx, hi2)

		fmt.Println("Hook wiring OK. Spans emitted above.")
		return
	}

	// 3. Create agent with OTel hooks wired in.
	ag, err := agent.New(agent.Config{
		Name:  "tracing-demo-agent",
		Model: model,
		Toolkits: []toolkit.Toolkit{
			calculator.New(),
		},
		ToolHooks: []hooks.ToolHook{toolHook},
		PreHooks:  []hooks.Hook{preHook},
		PostHooks: []hooks.Hook{postHook},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create agent: %v\n", err)
		os.Exit(1)
	}

	// 4. Run a simple query.
	fmt.Println("Running agent query: What is 42 * 7?")
	output, err := ag.Run(ctx, "What is 42 * 7?")
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent run error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Agent response: %s\n", output.Content)
	fmt.Printf("Tools executed: %d\n", len(output.ToolsExecuted))
	fmt.Println("Spans emitted to stdout above.")
}
