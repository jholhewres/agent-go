package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/eval"
	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// demoModel is a simple mock that echoes back the last word of the input.
type demoModel struct {
	models.BaseModel
}

func (m *demoModel) Invoke(_ context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	// Return a canned response that satisfies most test cases.
	last := req.Messages[len(req.Messages)-1].Content
	_ = last
	return &types.ModelResponse{
		Content: "Paris",
		Usage:   types.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		Model:   m.ID,
	}, nil
}

func (m *demoModel) InvokeStream(_ context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk, 2)
	go func() {
		defer close(ch)
		out, _ := m.Invoke(context.Background(), req)
		ch <- types.ResponseChunk{Content: out.Content}
		ch <- types.ResponseChunk{Done: true}
	}()
	return ch, nil
}

func main() {
	ctx := context.Background()

	// Build the agent under test.
	mainModel := &demoModel{BaseModel: models.BaseModel{ID: "demo-model", Provider: "mock"}}
	testAgent, err := agent.New(agent.Config{
		Name:  "demo-agent",
		Model: mainModel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create agent: %v\n", err)
		os.Exit(1)
	}

	// Build the judge agent (also uses the mock model).
	judgeModel := &demoModel{BaseModel: models.BaseModel{ID: "judge-model", Provider: "mock"}}
	judgeAgent, err := agent.New(agent.Config{
		Name:  "judge-agent",
		Model: judgeModel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create judge agent: %v\n", err)
		os.Exit(1)
	}

	// Define a Suite with 5 test cases.
	suite := &eval.Suite{
		Name: "geography-quiz",
		Cases: []*eval.TestCase{
			{Name: "france-capital", Input: "What is the capital of France?", Expected: "Paris"},
			{Name: "france-capital-2", Input: "Capital of France?", Expected: "Paris"},
			{Name: "france-capital-3", Input: "Name the capital city of France.", Expected: "Paris"},
			{Name: "wrong-expected", Input: "What is the capital of Germany?", Expected: "Berlin"},
			{Name: "contains-check", Input: "Tell me about Paris.", Expected: "Paris"},
		},
		Evaluators: []eval.Evaluator{
			eval.NewAccuracyEvaluator(eval.MatchContains),
			eval.NewPerformanceEvaluator(eval.PerformanceConfig{MaxLatencyMs: 5000}),
			eval.NewReliabilityEvaluator(),
			eval.NewJudgeEvaluator(judgeAgent, `You are an evaluator. Given an input, expected output, and actual output, decide if the actual output is acceptable. Return ONLY JSON: {"pass": true|false, "reason": "brief explanation"}`),
		},
	}

	fmt.Println("Running evaluation suite:", suite.Name)
	reports, err := eval.RunSuite(ctx, testAgent, suite)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RunSuite error: %v\n", err)
		os.Exit(1)
	}

	// Print JSON report.
	fmt.Println("\n=== JSON Report ===")
	var jsonBuf bytes.Buffer
	if err := eval.WriteJSON(&jsonBuf, reports); err != nil {
		fmt.Fprintf(os.Stderr, "WriteJSON error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(jsonBuf.String())

	// Print JUnit XML report.
	fmt.Println("\n=== JUnit XML Report ===")
	var xmlBuf bytes.Buffer
	if err := eval.WriteJUnit(&xmlBuf, reports, suite.Name); err != nil {
		fmt.Fprintf(os.Stderr, "WriteJUnit error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(xmlBuf.String())

	fmt.Println("\nDone.")
}
