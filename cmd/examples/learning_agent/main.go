package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/learning"
	"github.com/jholhewres/agent-go/pkg/agentgo/learning/sqlite"
	"github.com/jholhewres/agent-go/pkg/agentgo/models/openai"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/calculator"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
)

func main() {
	// Create OpenAI model
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create learning storage (SQLite)
	learningStorage, err := sqlite.New("./learning.db")
	if err != nil {
		log.Fatalf("Failed to create learning storage: %v", err)
	}
	defer learningStorage.Close()

	// Create learning machine
	learningMachine, err := learning.NewMachine(learningStorage)
	if err != nil {
		log.Fatalf("Failed to create learning machine: %v", err)
	}

	// Create agent with learning enabled
	ag, err := agent.New(agent.Config{
		Name:            "Learning Assistant",
		Model:           model,
		Toolkits:        []toolkit.Toolkit{calculator.New()},
		Instructions:    "You are a helpful assistant that remembers user preferences and learns from conversations.",
		UserID:          "user-123", // Required for learning
		Learning:        true,       // Enable learning
		LearningMachine: learningMachine,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()

	// First interaction - agent learns about user
	fmt.Println("=== First Interaction ===")
	output1, err := ag.Run(ctx, "Hi! My name is John and I prefer short, concise answers.")
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output1.Content)

	// Second interaction - agent should remember
	fmt.Println("=== Second Interaction ===")
	output2, err := ag.Run(ctx, "What's my name?")
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output2.Content)

	// Check learned memories
	fmt.Println("=== Learned Memories ===")
	profile, err := learningMachine.GetUserProfile(ctx, "user-123")
	if err != nil {
		log.Printf("No profile found: %v", err)
	} else {
		fmt.Printf("User Profile: %+v\n", profile)
	}

	memories, err := learningMachine.GetUserMemories(ctx, "user-123", 10)
	if err != nil {
		log.Printf("No memories found: %v", err)
	} else {
		fmt.Printf("\nUser Memories (%d):\n", len(memories))
		for _, mem := range memories {
			fmt.Printf("  - [%s] %s\n", mem.Type, mem.Content)
		}
	}
}
