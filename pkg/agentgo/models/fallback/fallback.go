// Package fallback provides a Model implementation that chains multiple models
// together, falling back to the next model when one fails with a retryable error.
package fallback

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// Option configures a FallbackModel.
type Option func(*Options)

// Options holds configuration for fallback behavior.
type Options struct {
	MaxRetries int           // Max retries per model before falling back (default: 1)
	RetryDelay time.Duration // Delay between retries (default: 500ms)
}

func defaultOptions() Options {
	return Options{
		MaxRetries: 1,
		RetryDelay: 500 * time.Millisecond,
	}
}

// WithMaxRetries sets the maximum number of retries per model.
// Use 0 for no retries (each model is tried once before falling back).
func WithMaxRetries(n int) Option {
	return func(o *Options) {
		if n >= 0 {
			o.MaxRetries = n
		}
	}
}

// WithRetryDelay sets the delay between retries.
func WithRetryDelay(d time.Duration) Option {
	return func(o *Options) {
		if d > 0 {
			o.RetryDelay = d
		}
	}
}

// FallbackModel wraps multiple models and tries each in order until one succeeds.
type FallbackModel struct {
	models.BaseModel
	chain   []models.Model
	options Options
}

// New creates a FallbackModel from the given chain of models.
// At least two models are required.
func New(chain []models.Model, opts ...Option) (*FallbackModel, error) {
	if len(chain) < 2 {
		return nil, fmt.Errorf("fallback chain requires at least 2 models, got %d", len(chain))
	}

	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	chainCopy := make([]models.Model, len(chain))
	copy(chainCopy, chain)

	ids := make([]string, len(chainCopy))
	for i, m := range chainCopy {
		ids[i] = m.GetID()
	}

	return &FallbackModel{
		BaseModel: models.BaseModel{
			ID:       strings.Join(ids, ","),
			Name:     "fallback(" + strings.Join(ids, ",") + ")",
			Provider: "fallback",
		},
		chain:   chainCopy,
		options: options,
	}, nil
}

// Invoke tries each model in the chain, falling back on retryable errors.
func (f *FallbackModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	var lastErr error

	for i, model := range f.chain {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		resp, err := f.invokeWithRetry(ctx, model, req)
		if err == nil {
			if resp.Metadata.Extra == nil {
				resp.Metadata.Extra = make(map[string]interface{})
			}
			resp.Metadata.Extra["fallback_index"] = i
			resp.Metadata.Extra["fallback_model"] = model.GetID()
			resp.Metadata.Extra["fallback_provider"] = model.GetProvider()
			return resp, nil
		}

		if !isRetryable(err) {
			return nil, err
		}

		lastErr = err
	}

	return nil, fmt.Errorf("all models in fallback chain failed: %w", lastErr)
}

// InvokeStream tries each model in the chain for streaming.
// Only falls back if the stream fails to open (not mid-stream failures).
func (f *FallbackModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	var lastErr error

	for _, model := range f.chain {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		stream, err := f.invokeStreamWithRetry(ctx, model, req)
		if err == nil {
			return stream, nil
		}

		if !isRetryable(err) {
			return nil, err
		}

		lastErr = err
	}

	return nil, fmt.Errorf("all models in fallback chain failed (stream): %w", lastErr)
}

func (f *FallbackModel) invokeWithRetry(ctx context.Context, model models.Model, req *models.InvokeRequest) (*types.ModelResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= f.options.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if attempt > 0 {
			timer := time.NewTimer(f.options.RetryDelay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			}
		}

		resp, err := model.Invoke(ctx, req)
		if err == nil {
			return resp, nil
		}

		if !isRetryable(err) {
			return nil, err
		}

		lastErr = err
	}

	return nil, lastErr
}

func (f *FallbackModel) invokeStreamWithRetry(ctx context.Context, model models.Model, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	var lastErr error

	for attempt := 0; attempt <= f.options.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if attempt > 0 {
			timer := time.NewTimer(f.options.RetryDelay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			}
		}

		stream, err := model.InvokeStream(ctx, req)
		if err == nil {
			return stream, nil
		}

		if !isRetryable(err) {
			return nil, err
		}

		lastErr = err
	}

	return nil, lastErr
}

// isRetryable determines whether an error should trigger a fallback/retry.
// Invalid input and config errors are NOT retryable (they'd fail on any model).
// API errors, rate limits, and timeouts ARE retryable.
func isRetryable(err error) bool {
	var agnoErr *types.AgnoError
	if errors.As(err, &agnoErr) {
		switch agnoErr.Code {
		case types.ErrCodeInvalidInput, types.ErrCodeInvalidConfig,
			types.ErrCodeInputCheck, types.ErrCodeOutputCheck,
			types.ErrCodePromptInjection, types.ErrCodePIIDetected,
			types.ErrCodeContentModeration:
			return false
		default:
			return true
		}
	}
	// Non-AgnoError errors are retryable by default (network errors, etc.)
	return true
}
