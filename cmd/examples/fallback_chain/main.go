package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/models/fallback"
	"github.com/jholhewres/agent-go/pkg/agentgo/models/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create multiple models to form the fallback chain.
	// If the primary model fails (rate limit, timeout, etc.), the next one is tried.
	primary, err := openai.New("gpt-4o", openai.Config{
		APIKey:    apiKey,
		MaxTokens: 500,
	})
	if err != nil {
		log.Fatalf("Failed to create primary model: %v", err)
	}

	secondary, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey:    apiKey,
		MaxTokens: 500,
	})
	if err != nil {
		log.Fatalf("Failed to create secondary model: %v", err)
	}

	// Build a fallback chain: try gpt-4o first, fall back to gpt-4o-mini.
	model, err := fallback.New(
		[]models.Model{primary, secondary},
		fallback.WithMaxRetries(1),
		fallback.WithRetryDelay(500*time.Millisecond),
	)
	if err != nil {
		log.Fatalf("Failed to create fallback model: %v", err)
	}

	ag, err := agent.New(agent.Config{
		Name:         "Resilient Assistant",
		Model:        model,
		Instructions: "You are a helpful assistant. Respond concisely.",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()
	output, err := ag.Run(ctx, "What are the three laws of thermodynamics? Be brief.")
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}

	fmt.Println("Response:")
	fmt.Println(output.Content)

	// Show which model in the chain actually responded.
	if fm, ok := output.Metadata["fallback_model"]; ok {
		fmt.Printf("\nServed by: %s\n", fm)
	}
}
