package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/cache"
	"github.com/jholhewres/agent-go/pkg/agentgo/learning"
	"github.com/jholhewres/agent-go/pkg/agentgo/memory"
	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/run"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/calculator"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// MockModel is a simple mock for testing
type MockModel struct {
	models.BaseModel
	InvokeFunc       func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
	InvokeStreamFunc func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error)
}

func (m *MockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, req)
	}
	return &types.ModelResponse{
		ID:      "test-response",
		Content: "Mock response",
		Model:   "mock-model",
	}, nil
}

func (m *MockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	if m.InvokeStreamFunc != nil {
		return m.InvokeStreamFunc(ctx, req)
	}
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Name:  "TestAgent",
				Model: &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
			},
			wantErr: false,
		},
		{
			name: "missing model",
			config: Config{
				Name: "TestAgent",
			},
			wantErr: true,
			errMsg:  "model is required",
		},
		{
			name: "with default values",
			config: Config{
				Model: &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if agent == nil {
					t.Error("New() returned nil agent")
					return
				}
				// Check defaults
				if agent.MaxLoops <= 0 {
					t.Error("MaxLoops should have default value > 0")
				}
				if agent.Memory == nil {
					t.Error("Memory should be initialized")
				}
			}
		})
	}
}

func TestAgent_Run_SimpleResponse(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test-1",
				Content: "Hello, this is a test response",
				Model:   "test",
			}, nil
		},
	}

	agent, err := New(Config{
		Name:  "TestAgent",
		Model: mockModel,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if output.Content != "Hello, this is a test response" {
		t.Errorf("Run() content = %v, want %v", output.Content, "Hello, this is a test response")
	}

	if len(output.Messages) < 2 {
		t.Errorf("Run() should have at least 2 messages (user + assistant)")
	}

	if output.Status != RunStatusCompleted {
		t.Fatalf("expected run to be completed, got %s", output.Status)
	}

	if output.RunID == "" {
		t.Fatalf("expected run id to be set")
	}
}

func TestAgent_RunStream_Basic(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeStreamFunc: func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
			ch := make(chan types.ResponseChunk, 2)
			ch <- types.ResponseChunk{Content: "Hello"}
			ch <- types.ResponseChunk{Content: " world"}
			close(ch)
			return ch, nil
		},
	}

	ag, err := New(Config{
		Name:  "StreamAgent",
		Model: mockModel,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	result, err := ag.RunStream(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if result == nil {
		t.Fatal("RunStream() returned nil result")
	}

	var chunks []string
	for evt := range result.Events {
		contentEvt, ok := evt.(*run.RunContentEvent)
		if !ok {
			t.Fatalf("expected RunContentEvent, got %T", evt)
		}
		chunks = append(chunks, contentEvt.Content)
	}

	done := <-result.Done
	if done.Err != nil {
		t.Fatalf("RunStream() Done error = %v", done.Err)
	}
	if done.Output == nil {
		t.Fatal("RunStream() Done output is nil")
	}

	if want := "Hello world"; done.Output.Content != want {
		t.Errorf("RunStream() content = %q, want %q", done.Output.Content, want)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 streamed chunks, got %d", len(chunks))
	}
	if len(done.Output.Events) != 3 {
		t.Fatalf("expected 3 events (2 content + 1 completed), got %d", len(done.Output.Events))
	}
}

func TestAgent_Run_EmitsEvents(t *testing.T) {
	agent, err := New(Config{
		Name: "events-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				return &types.ModelResponse{ID: "evt", Content: "event payload", Model: "mock"}, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	output, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(output.Events) < 1 {
		t.Fatalf("expected events to be recorded")
	}
	contentEvent, ok := output.Events[0].(*run.RunContentEvent)
	if !ok {
		t.Fatalf("expected first event to be RunContentEvent, got %T", output.Events[0])
	}
	if contentEvent.Content != "event payload" {
		t.Fatalf("unexpected event content %q", contentEvent.Content)
	}
	completed := output.Events[len(output.Events)-1]
	if completed.EventType() != "run_completed" {
		t.Fatalf("expected terminal event to be run_completed, got %s", completed.EventType())
	}
}

func TestAgent_Run_EmptyInput(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
	}

	agent, err := New(Config{
		Name:  "TestAgent",
		Model: mockModel,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	_, err = agent.Run(context.Background(), "")
	if err == nil {
		t.Error("Run() should return error for empty input")
	}
}

func TestAgent_Run_WithToolCalls(t *testing.T) {
	callCount := 0
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++

			// First call: return tool call
			if callCount == 1 {
				return &types.ModelResponse{
					ID:    "test-1",
					Model: "test",
					ToolCalls: []types.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 5, "b": 3}`,
							},
						},
					},
				}, nil
			}

			// Second call: return final answer
			return &types.ModelResponse{
				ID:      "test-2",
				Content: "The result is 8",
				Model:   "test",
			}, nil
		},
	}

	agent, err := New(Config{
		Name:     "TestAgent",
		Model:    mockModel,
		Toolkits: []toolkit.Toolkit{calculator.New()},
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "What is 5 + 3?")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 model calls (tool call + final), got %d", callCount)
	}

	if output.Content != "The result is 8" {
		t.Errorf("Run() content = %v, want %v", output.Content, "The result is 8")
	}

	if output.Status != RunStatusCompleted {
		t.Fatalf("expected run status completed, got %s", output.Status)
	}

	// Check metadata
	loops, ok := output.Metadata["loops"].(int)
	if !ok || loops != 2 {
		t.Errorf("Run() loops = %v, want 2", loops)
	}
}

func TestAgent_Run_UsesCache(t *testing.T) {
	provider, err := cache.NewMemoryProvider(8, time.Minute)
	if err != nil {
		t.Fatalf("NewMemoryProvider error = %v", err)
	}

	callCount := 0
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			return &types.ModelResponse{
				ID:      "cached-response",
				Content: "Cached result",
				Model:   "test",
			}, nil
		},
	}

	agent, err := New(Config{
		Name:          "CacheAgent",
		Model:         mockModel,
		EnableCache:   true,
		CacheProvider: provider,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	first, err := agent.Run(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 provider call, got %d", callCount)
	}
	if hit, _ := first.Metadata["cache_hit"].(bool); hit {
		t.Fatalf("first run should not be cache hit")
	}

	agent.ClearMemory()
	second, err := agent.Run(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected cached run without provider call, got %d", callCount)
	}
	if hit, _ := second.Metadata["cache_hit"].(bool); !hit {
		t.Fatalf("expected cache hit on second run")
	}

	if second.Content != "Cached result" {
		t.Fatalf("unexpected content: %v", second.Content)
	}

	if second.Status != RunStatusCompleted {
		t.Fatalf("expected cached run to be completed")
	}
}

func TestAgent_Run_ContextCancelled(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return nil, context.Canceled
		},
	}

	agent, err := New(Config{
		Name:  "CancelAgent",
		Model: mockModel,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	output, runErr := agent.Run(ctx, "Hello")
	if runErr == nil {
		t.Fatalf("expected cancellation error")
	}

	agnoErr, ok := runErr.(*types.AgnoError)
	if !ok || agnoErr.Code != types.ErrCodeCancelled {
		t.Fatalf("expected cancellation error code, got %#v", runErr)
	}

	if output == nil {
		t.Fatalf("expected run output even on cancellation")
	}
	if output.Status != RunStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", output.Status)
	}
	if output.CancellationReason == "" {
		t.Fatalf("expected cancellation reason to be populated")
	}
}

func TestAgent_Run_MaxLoops(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			// Always return tool calls to trigger max loops
			return &types.ModelResponse{
				ID:    "test",
				Model: "test",
				ToolCalls: []types.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: types.ToolCallFunction{
							Name:      "add",
							Arguments: `{"a": 1, "b": 1}`,
						},
					},
				},
			}, nil
		},
	}

	agent, err := New(Config{
		Name:     "TestAgent",
		Model:    mockModel,
		Toolkits: []toolkit.Toolkit{calculator.New()},
		MaxLoops: 3,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	_, err = agent.Run(context.Background(), "Test")
	if err == nil {
		t.Error("Run() should return error when max loops reached")
	}
}

func TestAgent_ClearMemory(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
	}

	agent, err := New(Config{
		Name:         "TestAgent",
		Model:        mockModel,
		Instructions: "You are a helpful assistant",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Add some messages
	agent.Memory.Add(types.NewUserMessage("Hello"))
	agent.Memory.Add(types.NewAssistantMessage("Hi there"))

	if len(agent.Memory.GetMessages()) < 3 { // system + user + assistant
		t.Error("Should have at least 3 messages")
	}

	// Clear memory
	agent.ClearMemory()

	messages := agent.Memory.GetMessages()
	if len(messages) != 1 {
		t.Errorf("After clear, should have 1 message (system), got %d", len(messages))
	}

	if messages[0].Role != types.RoleSystem {
		t.Error("First message after clear should be system message")
	}
}

func TestAgent_WithCustomMemory(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
	}

	customMemory := memory.NewInMemory(5)
	customMemory.Add(types.NewUserMessage("Previous message"))

	agent, err := New(Config{
		Name:   "TestAgent",
		Model:  mockModel,
		Memory: customMemory,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	messages := agent.Memory.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Should preserve custom memory, got %d messages", len(messages))
	}
}

// TestAgent_MultiTenant tests multi-tenant memory isolation
// 测试多租户内存隔离
func TestAgent_MultiTenant(t *testing.T) {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			// Echo back the number of messages in memory
			return &types.ModelResponse{
				ID:      "test-response",
				Content: fmt.Sprintf("I see %d messages in history", len(req.Messages)),
				Model:   "test",
			}, nil
		},
	}

	// Create two agents with same memory but different userIDs
	sharedMemory := memory.NewInMemory(100)

	agent1, _ := New(Config{
		ID:     "agent1",
		Name:   "Agent for User 1",
		Model:  model,
		UserID: "user1",
		Memory: sharedMemory,
	})

	agent2, _ := New(Config{
		ID:     "agent2",
		Name:   "Agent for User 2",
		Model:  model,
		UserID: "user2",
		Memory: sharedMemory,
	})

	// User 1 sends first message
	output1, err := agent1.Run(context.Background(), "Hello from user1")
	if err != nil {
		t.Fatalf("User1 run failed: %v", err)
	}

	// When Run() is called, it gets messages BEFORE adding the assistant response
	// So model sees: [user message] = 1 message
	if !strings.Contains(output1.Content, "1 messages") {
		t.Errorf("User1 model should see 1 message, got: %s", output1.Content)
	}

	// User 2 sends first message (should start fresh)
	output2, err := agent2.Run(context.Background(), "Hello from user2")
	if err != nil {
		t.Fatalf("User2 run failed: %v", err)
	}

	// User 2 also sees 1 message (their user message)
	if !strings.Contains(output2.Content, "1 messages") {
		t.Errorf("User2 model should see 1 message in their own context, got: %s", output2.Content)
	}

	// User 1 sends second message
	output1b, err := agent1.Run(context.Background(), "Second message from user1")
	if err != nil {
		t.Fatalf("User1 second run failed: %v", err)
	}

	// User 1 model should see 3 messages: [user1, assistant1, user2]
	if !strings.Contains(output1b.Content, "3 messages") {
		t.Errorf("User1 model should see 3 messages after second interaction, got: %s", output1b.Content)
	}

	// Verify memory isolation: user1 has 4 messages, user2 has 2 messages
	user1Size := sharedMemory.Size("user1")
	user2Size := sharedMemory.Size("user2")

	if user1Size != 4 {
		t.Errorf("User1 should have 4 messages in memory, got %d", user1Size)
	}

	if user2Size != 2 {
		t.Errorf("User2 should have 2 messages in memory, got %d", user2Size)
	}

	// Clear user1's memory
	agent1.ClearMemory()

	// User1 should start fresh
	if sharedMemory.Size("user1") != 0 {
		t.Errorf("User1 memory should be cleared, got %d messages", sharedMemory.Size("user1"))
	}

	// User2's memory should be unaffected
	if sharedMemory.Size("user2") != 2 {
		t.Errorf("User2 memory should be unaffected, got %d messages", sharedMemory.Size("user2"))
	}
}

func TestRunTyped_ValidJSON(t *testing.T) {
	type Result struct {
		Title string  `json:"title"`
		Score float64 `json:"score"`
	}

	ag, err := New(Config{
		Name: "typed-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				return &types.ModelResponse{Content: `{"title": "Test", "score": 9.5}`}, nil
			},
		},
		ResponseFormat: &models.ResponseFormat{
			Type: "json_schema",
			JSONSchema: map[string]interface{}{
				"name": "Result",
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{"type": "string"},
					"score": map[string]interface{}{"type": "number"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, output, err := RunTyped[Result](context.Background(), ag, "test")
	if err != nil {
		t.Fatalf("RunTyped failed: %v", err)
	}
	if result.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", result.Title)
	}
	if result.Score != 9.5 {
		t.Errorf("expected score 9.5, got %f", result.Score)
	}
	if output == nil {
		t.Error("expected non-nil output")
	}
}

func TestRunTyped_InvalidJSON(t *testing.T) {
	type Result struct {
		Title string `json:"title"`
	}

	ag, _ := New(Config{
		Name: "typed-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				return &types.ModelResponse{Content: "not valid json"}, nil
			},
		},
	})

	_, output, err := RunTyped[Result](context.Background(), ag, "test")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if output == nil {
		t.Error("expected non-nil output even on parse failure")
	}
}

func TestAgent_ResponseFormat_PassedToRequest(t *testing.T) {
	var capturedReq *models.InvokeRequest

	rf := &models.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]interface{}{
			"name": "TestSchema",
			"type": "object",
		},
	}

	ag, _ := New(Config{
		Name: "rf-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				capturedReq = req
				return &types.ModelResponse{Content: `{}`}, nil
			},
		},
		ResponseFormat: rf,
	})

	_, err := ag.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if capturedReq.ResponseFormat == nil {
		t.Fatal("expected ResponseFormat to be passed to InvokeRequest")
	}
	if capturedReq.ResponseFormat.Type != "json_schema" {
		t.Errorf("expected ResponseFormat.Type 'json_schema', got %q", capturedReq.ResponseFormat.Type)
	}
}

func TestAgent_NilResponseFormat_NotPassedToRequest(t *testing.T) {
	var capturedReq *models.InvokeRequest

	ag, _ := New(Config{
		Name: "no-rf-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				capturedReq = req
				return &types.ModelResponse{Content: "ok"}, nil
			},
		},
	})

	_, err := ag.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if capturedReq.ResponseFormat != nil {
		t.Error("expected nil ResponseFormat when not configured")
	}
}

func TestAgent_RunStream_WithToolCalls(t *testing.T) {
	var streamCallCount int

	calc := calculator.New()

	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeStreamFunc: func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
			streamCallCount++
			ch := make(chan types.ResponseChunk)
			go func() {
				defer close(ch)
				if streamCallCount == 1 {
					// First pass: return tool call
					ch <- types.ResponseChunk{
						ToolCalls: []types.ToolCall{
							{
								ID:   "call-1",
								Type: "function",
								Function: types.ToolCallFunction{
									Name:      "add",
									Arguments: `{"a": 2, "b": 3}`,
								},
							},
						},
					}
				} else {
					// Second pass: return text content
					ch <- types.ResponseChunk{Content: "The result is 5"}
				}
			}()
			return ch, nil
		},
	}

	ag, err := New(Config{
		Name:     "stream-tool-agent",
		Model:    model,
		Toolkits: []toolkit.Toolkit{calc},
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, err := ag.RunStream(context.Background(), "add 2 and 3")
	if err != nil {
		t.Fatalf("RunStream failed: %v", err)
	}

	// Drain events
	var events []run.BaseRunOutputEvent
	for evt := range result.Events {
		events = append(events, evt)
	}

	// Wait for done
	done := <-result.Done
	if done.Err != nil {
		t.Fatalf("RunStream done error: %v", done.Err)
	}
	if done.Output == nil {
		t.Fatal("expected non-nil output")
	}

	if done.Output.Content != "The result is 5" {
		t.Errorf("expected content 'The result is 5', got %q", done.Output.Content)
	}

	if streamCallCount != 2 {
		t.Errorf("expected 2 stream invocations, got %d", streamCallCount)
	}

	if len(done.Output.ToolsExecuted) != 1 {
		t.Errorf("expected 1 tool execution, got %d", len(done.Output.ToolsExecuted))
	}

	loops, ok := done.Output.Metadata["loops"].(int)
	if !ok || loops != 2 {
		t.Errorf("expected loops=2, got %v", done.Output.Metadata["loops"])
	}
}

func TestAgent_RunStream_MaxLoops(t *testing.T) {
	// Model always returns tool calls, never text
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeStreamFunc: func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
			ch := make(chan types.ResponseChunk)
			go func() {
				defer close(ch)
				ch <- types.ResponseChunk{
					ToolCalls: []types.ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 1, "b": 1}`,
							},
						},
					},
				}
			}()
			return ch, nil
		},
	}

	ag, _ := New(Config{
		Name:     "stream-max-loops",
		Model:    model,
		MaxLoops: 3,
		Toolkits: []toolkit.Toolkit{calculator.New()},
	})

	result, err := ag.RunStream(context.Background(), "loop forever")
	if err != nil {
		t.Fatalf("RunStream failed: %v", err)
	}

	// Drain events
	for range result.Events {
	}

	done := <-result.Done
	if done.Err == nil {
		t.Fatal("expected error when max loops reached")
	}
	if !strings.Contains(done.Err.Error(), "max tool calling loops") {
		t.Errorf("expected max loops error, got: %v", done.Err)
	}
}

func TestAgent_RunStream_ContentFlowsDuringToolLoop(t *testing.T) {
	var callCount int

	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeStreamFunc: func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
			callCount++
			ch := make(chan types.ResponseChunk)
			go func() {
				defer close(ch)
				if callCount == 1 {
					// First pass: content + tool call
					ch <- types.ResponseChunk{Content: "Let me calculate. "}
					ch <- types.ResponseChunk{
						ToolCalls: []types.ToolCall{
							{
								ID:   "call-1",
								Type: "function",
								Function: types.ToolCallFunction{
									Name:      "add",
									Arguments: `{"a": 1, "b": 2}`,
								},
							},
						},
					}
				} else {
					// Second pass: final content
					ch <- types.ResponseChunk{Content: "The answer is 3."}
				}
			}()
			return ch, nil
		},
	}

	ag, _ := New(Config{
		Name:     "stream-content-flow",
		Model:    model,
		Toolkits: []toolkit.Toolkit{calculator.New()},
	})

	result, err := ag.RunStream(context.Background(), "add 1 and 2")
	if err != nil {
		t.Fatalf("RunStream failed: %v", err)
	}

	// Collect content events
	var contentParts []string
	for evt := range result.Events {
		if ce, ok := evt.(*run.RunContentEvent); ok {
			contentParts = append(contentParts, ce.Content)
		}
	}

	done := <-result.Done
	if done.Err != nil {
		t.Fatalf("RunStream done error: %v", done.Err)
	}

	// We should see content events from both passes
	if len(contentParts) < 2 {
		t.Errorf("expected at least 2 content events, got %d: %v", len(contentParts), contentParts)
	}
}

// mockSessionPersister records PersistRun calls for testing.
type mockSessionPersister struct {
	calls []mockPersistCall
}

type mockPersistCall struct {
	SessionID string
	AgentID   string
	UserID    string
	Output    *RunOutput
}

func (m *mockSessionPersister) PersistRun(ctx context.Context, sessionID, agentID, userID string, output *RunOutput) error {
	m.calls = append(m.calls, mockPersistCall{
		SessionID: sessionID,
		AgentID:   agentID,
		UserID:    userID,
		Output:    output,
	})
	return nil
}

func TestAgent_SessionPersistence_Run(t *testing.T) {
	persister := &mockSessionPersister{}

	ag, _ := New(Config{
		Name:             "session-agent",
		Model:            &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
		UserID:           "user-1",
		SessionPersister: persister,
		SessionID:        "sess-123",
	})

	_, err := ag.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(persister.calls) != 1 {
		t.Fatalf("expected 1 PersistRun call, got %d", len(persister.calls))
	}
	if persister.calls[0].SessionID != "sess-123" {
		t.Errorf("expected session ID 'sess-123', got %q", persister.calls[0].SessionID)
	}
	if persister.calls[0].UserID != "user-1" {
		t.Errorf("expected user ID 'user-1', got %q", persister.calls[0].UserID)
	}
	if persister.calls[0].Output == nil {
		t.Error("expected non-nil output")
	}
}

func TestAgent_SessionPersistence_NotCalledWithout(t *testing.T) {
	// No SessionPersister configured — should not call anything.
	ag, _ := New(Config{
		Name:  "no-session-agent",
		Model: &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
	})

	_, err := ag.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	// No panic, no error — session persistence is silently skipped.
}

func TestAgent_SessionPersistence_NoSessionID(t *testing.T) {
	persister := &mockSessionPersister{}

	ag, _ := New(Config{
		Name:             "no-sid-agent",
		Model:            &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
		SessionPersister: persister,
		// SessionID intentionally omitted
	})

	_, err := ag.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(persister.calls) != 0 {
		t.Errorf("expected 0 PersistRun calls without SessionID, got %d", len(persister.calls))
	}
}

// mockLearningMachine records Learn calls and returns configured data.
type mockLearningMachine struct {
	mu          sync.Mutex
	learnCalls  int
	learnMsgs   []types.Message
	profile     *learning.UserProfile
	memories    []learning.UserMemory
	profileErr  error
	memoriesErr error
}

func (m *mockLearningMachine) Learn(ctx context.Context, userID string, messages []types.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.learnCalls++
	m.learnMsgs = messages
	return nil
}

func (m *mockLearningMachine) GetUserProfile(ctx context.Context, userID string) (*learning.UserProfile, error) {
	if m.profileErr != nil {
		return nil, m.profileErr
	}
	return m.profile, nil
}

func (m *mockLearningMachine) GetUserMemories(ctx context.Context, userID string, limit int) ([]learning.UserMemory, error) {
	if m.memoriesErr != nil {
		return nil, m.memoriesErr
	}
	return m.memories, nil
}

func (m *mockLearningMachine) GetLearnedKnowledge(ctx context.Context, topic string, limit int) ([]learning.Knowledge, error) {
	return nil, nil
}

func (m *mockLearningMachine) DeleteUserData(ctx context.Context, userID string) error {
	return nil
}

func TestAgent_Learning_LearnCalledAfterRun(t *testing.T) {
	lm := &mockLearningMachine{}

	ag, _ := New(Config{
		Name:            "learning-agent",
		Model:           &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
		UserID:          "user-1",
		Learning:        true,
		LearningMachine: lm,
	})

	_, err := ag.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Give the goroutine time to execute.
	time.Sleep(50 * time.Millisecond)

	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.learnCalls != 1 {
		t.Errorf("expected 1 Learn call, got %d", lm.learnCalls)
	}
	if len(lm.learnMsgs) == 0 {
		t.Error("expected non-empty messages in Learn call")
	}
}

func TestAgent_Learning_ContextInjected(t *testing.T) {
	var capturedReq *models.InvokeRequest

	lm := &mockLearningMachine{
		profile: &learning.UserProfile{
			UserID: "user-1",
			Name:   "Alice",
		},
		memories: []learning.UserMemory{
			{Content: "Prefers dark mode"},
			{Content: "Uses Go language"},
		},
	}

	ag, _ := New(Config{
		Name: "learning-ctx-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				capturedReq = req
				return &types.ModelResponse{Content: "ok"}, nil
			},
		},
		UserID:          "user-1",
		Learning:        true,
		LearningMachine: lm,
		Instructions:    "You are helpful.",
	})

	_, err := ag.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check that the system message includes learned context.
	if capturedReq == nil {
		t.Fatal("no request captured")
	}
	foundLearned := false
	for _, msg := range capturedReq.Messages {
		if msg.Role == types.RoleSystem && strings.Contains(msg.Content, "Learned Context") {
			foundLearned = true
			if !strings.Contains(msg.Content, "Alice") {
				t.Error("expected user name in learned context")
			}
			if !strings.Contains(msg.Content, "dark mode") {
				t.Error("expected memory content in learned context")
			}
			break
		}
	}
	if !foundLearned {
		t.Error("expected learned context to be injected into system message")
	}
}

func TestAgent_Learning_NotCalledWhenDisabled(t *testing.T) {
	lm := &mockLearningMachine{}

	ag, _ := New(Config{
		Name:            "no-learning-agent",
		Model:           &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
		UserID:          "user-1",
		Learning:        false, // Disabled
		LearningMachine: lm,
	})

	_, err := ag.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.learnCalls != 0 {
		t.Errorf("expected 0 Learn calls when disabled, got %d", lm.learnCalls)
	}
}

func TestAgent_Learning_NoUserID(t *testing.T) {
	lm := &mockLearningMachine{}

	ag, _ := New(Config{
		Name:            "no-uid-agent",
		Model:           &MockModel{BaseModel: models.BaseModel{ID: "mock", Provider: "mock"}},
		Learning:        true,
		LearningMachine: lm,
		// No UserID
	})

	_, err := ag.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.learnCalls != 0 {
		t.Errorf("expected 0 Learn calls without UserID, got %d", lm.learnCalls)
	}
}

// mockHistoryProvider returns configured history strings.
type mockHistoryProvider struct {
	history string
	err     error
}

func (m *mockHistoryProvider) GetHistory(ctx context.Context, sessionID string, maxRuns int) (string, error) {
	return m.history, m.err
}

func TestAgent_HistoryInjection(t *testing.T) {
	var capturedReq *models.InvokeRequest

	hp := &mockHistoryProvider{
		history: "[Previous Conversation History]\nRun 1: Hello world",
	}

	ag, _ := New(Config{
		Name: "history-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				capturedReq = req
				return &types.ModelResponse{Content: "ok"}, nil
			},
		},
		Instructions:    "You are helpful.",
		SessionID:       "sess-1",
		HistoryProvider: hp,
	})

	_, err := ag.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if capturedReq == nil {
		t.Fatal("no request captured")
	}
	foundHistory := false
	for _, msg := range capturedReq.Messages {
		if msg.Role == types.RoleSystem && strings.Contains(msg.Content, "Previous Conversation History") {
			foundHistory = true
			if !strings.Contains(msg.Content, "Hello world") {
				t.Error("expected history content")
			}
			break
		}
	}
	if !foundHistory {
		t.Error("expected history to be injected into system message")
	}
}

func TestAgent_HistoryInjection_NoSessionID(t *testing.T) {
	var capturedReq *models.InvokeRequest

	hp := &mockHistoryProvider{
		history: "should not appear",
	}

	ag, _ := New(Config{
		Name: "no-session-history-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				capturedReq = req
				return &types.ModelResponse{Content: "ok"}, nil
			},
		},
		Instructions:    "You are helpful.",
		HistoryProvider: hp,
		// No SessionID
	})

	_, err := ag.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	for _, msg := range capturedReq.Messages {
		if msg.Role == types.RoleSystem && strings.Contains(msg.Content, "should not appear") {
			t.Error("history should not be injected without SessionID")
		}
	}
}

func TestAgent_HistoryInjection_EmptyHistory(t *testing.T) {
	var capturedReq *models.InvokeRequest

	hp := &mockHistoryProvider{
		history: "",
	}

	ag, _ := New(Config{
		Name: "empty-history-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "mock", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				capturedReq = req
				return &types.ModelResponse{Content: "ok"}, nil
			},
		},
		Instructions:    "You are helpful.",
		SessionID:       "sess-1",
		HistoryProvider: hp,
	})

	_, err := ag.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// System message should just be instructions without history
	for _, msg := range capturedReq.Messages {
		if msg.Role == types.RoleSystem {
			if strings.Contains(msg.Content, "Previous") {
				t.Error("should not inject empty history")
			}
		}
	}
}
