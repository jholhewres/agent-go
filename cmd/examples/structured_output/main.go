package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/models/openai"
	"github.com/jholhewres/agent-go/pkg/agentgo/structured"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// CodeReview represents a structured code review result.
type CodeReview struct {
	Summary     string   `json:"summary" description:"A brief summary of the code review"`
	Score       int      `json:"score" description:"Quality score from 1 to 10"`
	Issues      []string `json:"issues" description:"List of issues found"`
	Suggestions []string `json:"suggestions,omitempty" description:"Optional improvement suggestions"`
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey:      apiKey,
		Temperature: 0.3,
		MaxTokens:   1000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Generate a JSON schema from the Go struct.
	schema, err := structured.SchemaFromType(CodeReview{})
	if err != nil {
		log.Fatalf("Failed to generate schema: %v", err)
	}
	schema.Description = "A structured code review result"

	// Create an agent with structured output.
	ag, err := agent.New(agent.Config{
		Name:  "Code Reviewer",
		Model: model,
		Instructions: `You are a code review expert. When given code to review,
analyze it and respond with a structured JSON review including a summary,
quality score (1-10), issues found, and improvement suggestions.`,
		ResponseFormat: schema.ToResponseFormat(),
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()

	// --- Method 1: RunTyped (generic helper) ---
	fmt.Println("=== Method 1: RunTyped ===")
	review, _, err := agent.RunTyped[CodeReview](ctx, ag, `
Review this Go function:

func divide(a, b int) int {
    return a / b
}`)
	if err != nil {
		log.Fatalf("RunTyped failed: %v", err)
	}
	fmt.Printf("Summary:  %s\n", review.Summary)
	fmt.Printf("Score:    %d/10\n", review.Score)
	fmt.Println("Issues:")
	for _, issue := range review.Issues {
		fmt.Printf("  - %s\n", issue)
	}
	if len(review.Suggestions) > 0 {
		fmt.Println("Suggestions:")
		for _, s := range review.Suggestions {
			fmt.Printf("  - %s\n", s)
		}
	}

	// --- Method 2: Manual parsing ---
	fmt.Println("\n=== Method 2: Manual Run + Parse ===")
	output, err := ag.Run(ctx, `
Review this Go function:

func readFile(path string) string {
    data, _ := os.ReadFile(path)
    return string(data)
}`)
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}

	var review2 CodeReview
	if err := structured.ParseResponse(&types.ModelResponse{Content: output.Content}, &review2); err != nil {
		log.Fatalf("ParseResponse failed: %v", err)
	}
	fmt.Printf("Summary: %s\n", review2.Summary)
	fmt.Printf("Score:   %d/10\n", review2.Score)
	fmt.Printf("Issues:  %v\n", review2.Issues)
}
