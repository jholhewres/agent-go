package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/models/openai"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/agenttool"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey:    apiKey,
		MaxTokens: 500,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// --- Example 1: Agent as Tool ---
	fmt.Println("=== Example 1: Agent as Tool ===")

	// Create a specialist sub-agent.
	researcher, err := agent.New(agent.Config{
		Name:  "researcher",
		Model: model,
		Instructions: `You are a research expert. When asked a question, provide
a concise factual answer in 2-3 sentences.`,
	})
	if err != nil {
		log.Fatalf("Failed to create researcher: %v", err)
	}

	// Wrap the researcher as a tool for the orchestrator.
	researchTool := agenttool.New(researcher, "Ask the research expert for factual information")

	orchestrator, err := agent.New(agent.Config{
		Name:     "orchestrator",
		Model:    model,
		Toolkits: []toolkit.Toolkit{researchTool},
		Instructions: `You are an orchestrator. When the user asks a factual question,
delegate to the research expert using the ask_researcher tool, then summarize the answer.`,
		MaxLoops: 5,
	})
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	ctx := context.Background()
	output, err := orchestrator.Run(ctx, "What causes auroras (northern lights)?")
	if err != nil {
		log.Fatalf("Orchestrator run failed: %v", err)
	}
	fmt.Println(output.Content)

	// --- Example 2: SpawnAll (concurrent sub-agents) ---
	fmt.Println("\n=== Example 2: SpawnAll (concurrent) ===")

	historian, err := agent.New(agent.Config{
		Name:         "historian",
		Model:        model,
		Instructions: "You are a historian. Answer in one sentence.",
	})
	if err != nil {
		log.Fatalf("Failed to create historian: %v", err)
	}

	scientist, err := agent.New(agent.Config{
		Name:         "scientist",
		Model:        model,
		Instructions: "You are a scientist. Answer in one sentence.",
	})
	if err != nil {
		log.Fatalf("Failed to create scientist: %v", err)
	}

	results := agent.SpawnAll(ctx, []agent.SpawnConfig{
		{Agent: historian, Input: "When was the first moon landing?"},
		{Agent: scientist, Input: "What is the speed of light in km/s?"},
	})

	for _, r := range results {
		if r.Error != nil {
			fmt.Printf("[%s] Error: %v\n", r.AgentID, r.Error)
		} else {
			fmt.Printf("[%s] %s\n", r.AgentID, r.Output.Content)
		}
	}
}
