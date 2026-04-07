package memory

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// mockModel is a test double for models.Model.
type mockModel struct {
	models.BaseModel
	invokeFn  func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
	callCount int
}

func (m *mockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	m.callCount++
	if m.invokeFn != nil {
		return m.invokeFn(ctx, req)
	}
	return &types.ModelResponse{Content: "mocked summary"}, nil
}

func (m *mockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk, 1)
	ch <- types.ResponseChunk{Content: "mocked", Done: true}
	close(ch)
	return ch, nil
}

func newTestSummarizing(t *testing.T, threshold, preserveLast int, model *mockModel) *SummarizingMemory {
	t.Helper()
	sm, err := NewSummarizingMemory(SummarizingConfig{
		Inner:        NewInMemory(200),
		Model:        model,
		Threshold:    threshold,
		PreserveLast: preserveLast,
	})
	if err != nil {
		t.Fatalf("NewSummarizingMemory: %v", err)
	}
	return sm
}

// Test 1: below threshold — compact is never called.
func TestSummarizingMemory_BelowThreshold(t *testing.T) {
	model := &mockModel{}
	sm := newTestSummarizing(t, 10, 5, model)

	for i := 0; i < 5; i++ {
		sm.Add(types.NewUserMessage("hello"))
	}

	if model.callCount != 0 {
		t.Errorf("expected 0 model calls below threshold, got %d", model.callCount)
	}
	if sm.Size() != 5 {
		t.Errorf("expected size 5, got %d", sm.Size())
	}
}

// Test 2: above threshold — compact fires and results in 1+PreserveLast messages.
func TestSummarizingMemory_AboveThreshold_CompactsToExpectedSize(t *testing.T) {
	const threshold = 5
	const preserveLast = 3
	model := &mockModel{}
	sm := newTestSummarizing(t, threshold, preserveLast, model)

	// Add threshold+1 messages to trigger compact.
	for i := 0; i < threshold+1; i++ {
		sm.Add(types.NewUserMessage("msg"))
	}

	// After compact: 1 summary + preserveLast kept messages.
	wantSize := 1 + preserveLast
	if sm.Size() != wantSize {
		t.Errorf("expected size %d after compact, got %d", wantSize, sm.Size())
	}
	if model.callCount == 0 {
		t.Error("expected model to be called during compact")
	}
}

// Test 3: first message after compact is a System message with SummaryTag prefix.
func TestSummarizingMemory_SummaryMessageRole(t *testing.T) {
	const threshold = 5
	const preserveLast = 2
	tag := "[TEST SUMMARY]"
	model := &mockModel{
		invokeFn: func(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{Content: "key facts here"}, nil
		},
	}
	sm, err := NewSummarizingMemory(SummarizingConfig{
		Inner:        NewInMemory(200),
		Model:        model,
		Threshold:    threshold,
		PreserveLast: preserveLast,
		SummaryTag:   tag,
	})
	if err != nil {
		t.Fatalf("NewSummarizingMemory: %v", err)
	}

	for i := 0; i < threshold+1; i++ {
		sm.Add(types.NewUserMessage("msg"))
	}

	msgs := sm.GetMessages()
	if len(msgs) == 0 {
		t.Fatal("expected messages after compact")
	}
	first := msgs[0]
	if first.Role != types.RoleSystem {
		t.Errorf("expected first message role System, got %s", first.Role)
	}
	if !strings.HasPrefix(first.Content, tag) {
		t.Errorf("expected content to start with %q, got %q", tag, first.Content)
	}
}

// Test 4: model failure preserves original state.
func TestSummarizingMemory_ModelFailurePreservesState(t *testing.T) {
	const threshold = 5
	const preserveLast = 2
	model := &mockModel{
		invokeFn: func(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
			return nil, errors.New("LLM unavailable")
		},
	}
	sm := newTestSummarizing(t, threshold, preserveLast, model)

	for i := 0; i < threshold+1; i++ {
		sm.Add(types.NewUserMessage("msg"))
	}

	// State should be unchanged (threshold+1 messages still present).
	if sm.Size() != threshold+1 {
		t.Errorf("expected original size %d preserved after model failure, got %d", threshold+1, sm.Size())
	}
}

// Test 5: PreserveLast >= total messages → no-op (no compact).
func TestSummarizingMemory_PreserveLastGtTotal_Noop(t *testing.T) {
	const threshold = 3
	const preserveLast = 20 // larger than any message count we add
	model := &mockModel{}
	sm := newTestSummarizing(t, threshold, preserveLast, model)

	for i := 0; i < threshold+1; i++ {
		sm.Add(types.NewUserMessage("msg"))
	}

	// compact triggers but is a no-op because preserveLast >= total.
	// Messages stay intact.
	if sm.Size() != threshold+1 {
		t.Errorf("expected size %d (no-op compact), got %d", threshold+1, sm.Size())
	}
}

// Test 6: chronological order is preserved after compact.
func TestSummarizingMemory_ChronologicalOrderPreserved(t *testing.T) {
	const threshold = 4
	const preserveLast = 2
	model := &mockModel{}
	sm := newTestSummarizing(t, threshold, preserveLast, model)

	msgs := []string{"a", "b", "c", "d", "e"}
	for _, content := range msgs {
		sm.Add(types.NewUserMessage(content))
	}

	got := sm.GetMessages()
	// Expect: [summary, "d", "e"]
	if len(got) != 1+preserveLast {
		t.Fatalf("expected %d messages, got %d", 1+preserveLast, len(got))
	}
	// Last two kept messages should be in original order.
	if got[1].Content != "d" {
		t.Errorf("expected second message content 'd', got %q", got[1].Content)
	}
	if got[2].Content != "e" {
		t.Errorf("expected third message content 'e', got %q", got[2].Content)
	}
}

// Test 7: SummarizingMemory wrapping HybridMemory compiles and delegates correctly.
func TestSummarizingMemory_WrapsHybridMemoryInterface(t *testing.T) {
	// HybridMemory requires VectorDB and Embedder — we test only that
	// SummarizingMemory accepts any Memory implementation (here InMemory as stand-in).
	// The type assertion below confirms interface compatibility.
	inner := NewInMemory(100)
	model := &mockModel{}

	sm, err := NewSummarizingMemory(SummarizingConfig{
		Inner:        inner, // any Memory satisfies the contract, including HybridMemory
		Model:        model,
		Threshold:    10,
		PreserveLast: 3,
	})
	if err != nil {
		t.Fatalf("NewSummarizingMemory: %v", err)
	}

	// Verify it implements the Memory interface.
	var _ Memory = sm

	sm.Add(types.NewUserMessage("test"))
	if sm.Size() != 1 {
		t.Errorf("expected size 1, got %d", sm.Size())
	}
	sm.Clear()
	if sm.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", sm.Size())
	}
}

// Test 8: NewSummarizingMemory returns error on missing required fields.
func TestNewSummarizingMemory_ValidationErrors(t *testing.T) {
	model := &mockModel{}
	inner := NewInMemory(10)

	_, err := NewSummarizingMemory(SummarizingConfig{Inner: nil, Model: model})
	if err == nil {
		t.Error("expected error when Inner is nil")
	}

	_, err = NewSummarizingMemory(SummarizingConfig{Inner: inner, Model: nil})
	if err == nil {
		t.Error("expected error when Model is nil")
	}
}
