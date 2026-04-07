// summarizing_memory_agent demonstrates SummarizingMemory with a mock LLM.
// No API key is required — a local MockModel returns a fixed summary string.
// Run:
//
//	go run ./cmd/examples/summarizing_memory_agent/
package main

import (
	"context"
	"fmt"

	"github.com/jholhewres/agent-go/pkg/agentgo/memory"
	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// MockModel satisfies models.Model without any network call.
type MockModel struct {
	models.BaseModel
}

func (m *MockModel) Invoke(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
	return &types.ModelResponse{
		Content: "User discussed 50 topics including greetings, math, and general knowledge.",
	}, nil
}

func (m *MockModel) InvokeStream(_ context.Context, _ *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk, 1)
	ch <- types.ResponseChunk{Content: "summary", Done: true}
	close(ch)
	return ch, nil
}

func main() {
	inner := memory.NewInMemory(200)
	mock := &MockModel{
		BaseModel: models.BaseModel{ID: "mock", Provider: "local"},
	}

	sm, err := memory.NewSummarizingMemory(memory.SummarizingConfig{
		Inner:        inner,
		Model:        mock,
		Threshold:    50,
		PreserveLast: 10,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create SummarizingMemory: %v", err))
	}

	fmt.Println("Adding 60 messages...")
	for i := 1; i <= 60; i++ {
		sm.Add(types.NewUserMessage(fmt.Sprintf("message number %d", i)))
	}

	msgs := sm.GetMessages()
	fmt.Printf("\nMemory size after 60 messages: %d (expected ~11: 1 summary + 10 kept)\n", len(msgs))

	for i, msg := range msgs {
		role := string(msg.Role)
		content := msg.Content
		if len(content) > 80 {
			content = content[:80] + "..."
		}
		fmt.Printf("  [%d] (%s) %s\n", i, role, content)
	}
}
