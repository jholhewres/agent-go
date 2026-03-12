package fallback

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// mockModel is a test double for models.Model.
type mockModel struct {
	models.BaseModel
	invokeFunc       func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
	invokeStreamFunc func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error)
	invokeCalls      atomic.Int32
	streamCalls      atomic.Int32
}

func newMockModel(id string) *mockModel {
	return &mockModel{
		BaseModel: models.BaseModel{ID: id, Name: id, Provider: "mock"},
	}
}

func (m *mockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	m.invokeCalls.Add(1)
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, req)
	}
	return &types.ModelResponse{Content: "response from " + m.ID}, nil
}

func (m *mockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	m.streamCalls.Add(1)
	if m.invokeStreamFunc != nil {
		return m.invokeStreamFunc(ctx, req)
	}
	ch := make(chan types.ResponseChunk, 1)
	ch <- types.ResponseChunk{Content: "streamed from " + m.ID, Done: true}
	close(ch)
	return ch, nil
}

func TestNew_RequiresAtLeastTwoModels(t *testing.T) {
	m := newMockModel("m1")

	_, err := New([]models.Model{m})
	if err == nil {
		t.Fatal("expected error for single model chain")
	}

	_, err = New([]models.Model{m, newMockModel("m2")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_SetsMetadata(t *testing.T) {
	m1 := newMockModel("gpt-4")
	m2 := newMockModel("claude-3")

	fb, err := New([]models.Model{m1, m2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fb.GetProvider() != "fallback" {
		t.Errorf("expected provider 'fallback', got %q", fb.GetProvider())
	}
	if fb.GetID() != "gpt-4,claude-3" {
		t.Errorf("expected ID 'gpt-4,claude-3', got %q", fb.GetID())
	}
}

func TestInvoke_FirstModelSucceeds(t *testing.T) {
	m1 := newMockModel("m1")
	m2 := newMockModel("m2")

	fb, _ := New([]models.Model{m1, m2})
	resp, err := fb.Invoke(context.Background(), &models.InvokeRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "response from m1" {
		t.Errorf("expected content from m1, got %q", resp.Content)
	}
	if m1.invokeCalls.Load() != 1 {
		t.Errorf("expected m1 called once, got %d", m1.invokeCalls.Load())
	}
	if m2.invokeCalls.Load() != 0 {
		t.Errorf("expected m2 not called, got %d", m2.invokeCalls.Load())
	}
	if resp.Metadata.Extra["fallback_index"] != 0 {
		t.Errorf("expected fallback_index 0, got %v", resp.Metadata.Extra["fallback_index"])
	}
}

func TestInvoke_FallsBackOnError(t *testing.T) {
	m1 := newMockModel("m1")
	m1.invokeFunc = func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
		return nil, types.NewAPIError("m1 failed", nil)
	}
	m2 := newMockModel("m2")
	m3 := newMockModel("m3")

	fb, _ := New([]models.Model{m1, m2, m3})
	resp, err := fb.Invoke(context.Background(), &models.InvokeRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "response from m2" {
		t.Errorf("expected content from m2, got %q", resp.Content)
	}
	if resp.Metadata.Extra["fallback_index"] != 1 {
		t.Errorf("expected fallback_index 1, got %v", resp.Metadata.Extra["fallback_index"])
	}
}

func TestInvoke_AllModelsFail(t *testing.T) {
	apiErr := types.NewAPIError("fail", nil)
	m1 := newMockModel("m1")
	m1.invokeFunc = func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
		return nil, apiErr
	}
	m2 := newMockModel("m2")
	m2.invokeFunc = func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
		return nil, apiErr
	}

	fb, _ := New([]models.Model{m1, m2})
	_, err := fb.Invoke(context.Background(), &models.InvokeRequest{})
	if err == nil {
		t.Fatal("expected error when all models fail")
	}
}

func TestInvoke_NoFallbackOnInvalidInput(t *testing.T) {
	m1 := newMockModel("m1")
	m1.invokeFunc = func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
		return nil, types.NewInvalidInputError("bad input", nil)
	}
	m2 := newMockModel("m2")

	fb, _ := New([]models.Model{m1, m2})
	_, err := fb.Invoke(context.Background(), &models.InvokeRequest{})
	if err == nil {
		t.Fatal("expected error on invalid input")
	}
	if m2.invokeCalls.Load() != 0 {
		t.Errorf("m2 should not have been called on non-retryable error, got %d calls", m2.invokeCalls.Load())
	}
}

func TestInvoke_RetriesWithinModel(t *testing.T) {
	callCount := 0
	m1 := newMockModel("m1")
	m1.invokeFunc = func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
		callCount++
		if callCount <= 2 {
			return nil, types.NewRateLimitError("rate limited", nil)
		}
		return &types.ModelResponse{Content: "success after retry"}, nil
	}
	m2 := newMockModel("m2")

	fb, _ := New([]models.Model{m1, m2}, WithMaxRetries(3), WithRetryDelay(time.Millisecond))
	resp, err := fb.Invoke(context.Background(), &models.InvokeRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "success after retry" {
		t.Errorf("expected retry success, got %q", resp.Content)
	}
	if m2.invokeCalls.Load() != 0 {
		t.Errorf("m2 should not have been called after m1 retry succeeded")
	}
}

func TestInvoke_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	m1 := newMockModel("m1")
	m2 := newMockModel("m2")

	fb, _ := New([]models.Model{m1, m2})
	_, err := fb.Invoke(ctx, &models.InvokeRequest{})
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
	if m1.invokeCalls.Load() != 0 {
		t.Errorf("no model should have been called on cancelled context")
	}
}

func TestInvokeStream_FirstModelSucceeds(t *testing.T) {
	m1 := newMockModel("m1")
	m2 := newMockModel("m2")

	fb, _ := New([]models.Model{m1, m2})
	stream, err := fb.InvokeStream(context.Background(), &models.InvokeRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chunk := <-stream
	if chunk.Content != "streamed from m1" {
		t.Errorf("expected streamed content from m1, got %q", chunk.Content)
	}
}

func TestInvokeStream_FallsBackOnOpenError(t *testing.T) {
	m1 := newMockModel("m1")
	m1.invokeStreamFunc = func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
		return nil, types.NewAPIError("stream open failed", nil)
	}
	m2 := newMockModel("m2")

	fb, _ := New([]models.Model{m1, m2})
	stream, err := fb.InvokeStream(context.Background(), &models.InvokeRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chunk := <-stream
	if chunk.Content != "streamed from m2" {
		t.Errorf("expected streamed content from m2, got %q", chunk.Content)
	}
}

func TestInvoke_FallbackMetadata(t *testing.T) {
	m1 := newMockModel("m1")
	m1.invokeFunc = func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
		return nil, types.NewAPIError("fail", nil)
	}
	m2 := newMockModel("m2")
	m2.invokeFunc = func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
		return nil, types.NewAPIError("fail", nil)
	}
	m3 := newMockModel("m3")

	fb, _ := New([]models.Model{m1, m2, m3})
	resp, err := fb.Invoke(context.Background(), &models.InvokeRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Metadata.Extra["fallback_model"] != "m3" {
		t.Errorf("expected fallback_model 'm3', got %v", resp.Metadata.Extra["fallback_model"])
	}
	if resp.Metadata.Extra["fallback_provider"] != "mock" {
		t.Errorf("expected fallback_provider 'mock', got %v", resp.Metadata.Extra["fallback_provider"])
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"API error", types.NewAPIError("fail", nil), true},
		{"rate limit", types.NewRateLimitError("limited", nil), true},
		{"timeout", types.NewModelTimeoutError("timeout", nil), true},
		{"invalid input", types.NewInvalidInputError("bad", nil), false},
		{"invalid config", types.NewInvalidConfigError("bad", nil), false},
		{"prompt injection", types.NewPromptInjectionError("detected", nil), false},
		{"PII detected", types.NewPIIDetectedError("found", nil), false},
		{"generic error", fmt.Errorf("network error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryable(tt.err); got != tt.retryable {
				t.Errorf("isRetryable(%v) = %v, want %v", tt.err, got, tt.retryable)
			}
		})
	}
}

func TestWithOptions(t *testing.T) {
	m1 := newMockModel("m1")
	m2 := newMockModel("m2")

	fb, _ := New([]models.Model{m1, m2},
		WithMaxRetries(5),
		WithRetryDelay(2*time.Second),
	)

	if fb.options.MaxRetries != 5 {
		t.Errorf("expected MaxRetries 5, got %d", fb.options.MaxRetries)
	}
	if fb.options.RetryDelay != 2*time.Second {
		t.Errorf("expected RetryDelay 2s, got %v", fb.options.RetryDelay)
	}
}
