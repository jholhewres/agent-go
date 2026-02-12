// Example: Agent with Tool Execution Hooks
// This example demonstrates how to use tool hooks to monitor and control tool execution.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/hooks"
	"github.com/jholhewres/agent-go/pkg/agentgo/models/openai"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/calculator"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
)

// LoggingToolHook logs all tool executions with timing information
type LoggingToolHook struct{}

func (h *LoggingToolHook) OnToolPre(ctx context.Context, input *hooks.ToolHookInput) error {
	fmt.Printf("[PRE] Tool '%s' starting...\n", input.FunctionName)
	fmt.Printf("      Arguments: %v\n", input.Arguments)
	return nil
}

func (h *LoggingToolHook) OnToolPost(ctx context.Context, input *hooks.ToolHookInput) error {
	status := "SUCCESS"
	if input.Failed() {
		status = "FAILED"
	}
	fmt.Printf("[POST] Tool '%s' completed in %v (%s)\n",
		input.FunctionName, input.Duration, status)
	if input.ResultError != nil {
		fmt.Printf("       Error: %v\n", input.ResultError)
	} else {
		fmt.Printf("       Result: %v\n", input.Result)
	}
	return nil
}

// RateLimitToolHook blocks rapid tool calls
type RateLimitToolHook struct {
	lastCall time.Time
	minDelay time.Duration
}

func NewRateLimitToolHook(minDelay time.Duration) *RateLimitToolHook {
	return &RateLimitToolHook{
		minDelay: minDelay,
	}
}

func (h *RateLimitToolHook) OnToolPre(ctx context.Context, input *hooks.ToolHookInput) error {
	if !h.lastCall.IsZero() && time.Since(h.lastCall) < h.minDelay {
		return fmt.Errorf("rate limit: please wait %v before next tool call", h.minDelay)
	}
	h.lastCall = time.Now()
	return nil
}

func (h *RateLimitToolHook) OnToolPost(ctx context.Context, input *hooks.ToolHookInput) error {
	return nil
}

// MetricsToolHook collects metrics about tool usage
type MetricsToolHook struct {
	callCount   map[string]int
	totalTime   map[string]time.Duration
	errorCount  map[string]int
}

func NewMetricsToolHook() *MetricsToolHook {
	return &MetricsToolHook{
		callCount:  make(map[string]int),
		totalTime:  make(map[string]time.Duration),
		errorCount: make(map[string]int),
	}
}

func (h *MetricsToolHook) OnToolPre(ctx context.Context, input *hooks.ToolHookInput) error {
	return nil
}

func (h *MetricsToolHook) OnToolPost(ctx context.Context, input *hooks.ToolHookInput) error {
	h.callCount[input.FunctionName]++
	h.totalTime[input.FunctionName] += input.Duration
	if input.Failed() {
		h.errorCount[input.FunctionName]++
	}
	return nil
}

func (h *MetricsToolHook) PrintReport() {
	fmt.Println("\n=== Tool Metrics Report ===")
	for name, count := range h.callCount {
		avgTime := h.totalTime[name] / time.Duration(count)
		fmt.Printf("Tool: %s\n", name)
		fmt.Printf("  Calls: %d\n", count)
		fmt.Printf("  Avg Time: %v\n", avgTime)
		fmt.Printf("  Errors: %d\n", h.errorCount[name])
	}
	fmt.Println("===========================")
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable is required")
		os.Exit(1)
	}

	// Create model
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey: apiKey,
	})
	if err != nil {
		fmt.Printf("Error creating model: %v\n", err)
		os.Exit(1)
	}

	// Create hooks
	loggingHook := &LoggingToolHook{}
	metricsHook := NewMetricsToolHook()

	// Create agent with tool hooks
	ag, err := agent.New(agent.Config{
		Name:     "Agent with Tool Hooks",
		Model:    model,
		Toolkits: []toolkit.Toolkit{calculator.New()},
		ToolHooks: []hooks.ToolHook{
			loggingHook,  // Log all tool executions
			metricsHook,  // Collect metrics
		},
	})
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Run agent
	fmt.Println("Running agent with tool hooks...\n")
	output, err := ag.Run(context.Background(), "What is 25 * 4 + 10?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Print response
	fmt.Printf("\nResponse: %s\n", output.Content)

	// Print tool execution summary
	fmt.Printf("\n=== Tool Execution Summary ===\n")
	fmt.Printf("Total tools executed: %d\n", len(output.ToolsExecuted))
	for i, exec := range output.ToolsExecuted {
		fmt.Printf("\nTool #%d:\n", i+1)
		fmt.Printf("  Name: %s\n", exec.FunctionName)
		fmt.Printf("  Status: %s\n", exec.Status)
		fmt.Printf("  Duration: %v\n", exec.Duration)
		if exec.Arguments != nil {
			fmt.Printf("  Arguments: %v\n", exec.Arguments)
		}
		if exec.Result != nil {
			fmt.Printf("  Result: %v\n", exec.Result)
		}
		if exec.Error != "" {
			fmt.Printf("  Error: %s\n", exec.Error)
		}
	}

	// Print metrics report
	metricsHook.PrintReport()
}
