package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	evolinkimg "github.com/jholhewres/agent-go/pkg/agentgo/models/evolink/image"
	evolinktxt "github.com/jholhewres/agent-go/pkg/agentgo/models/evolink/text"
	evolinkvid "github.com/jholhewres/agent-go/pkg/agentgo/models/evolink/video"
	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/workflow"
)

// TestMediaPipelineEndToEnd mocks all three EvoLink endpoints and runs the
// text→image→video workflow to confirm the models compose correctly (task 2.2).
func TestMediaPipelineEndToEnd(t *testing.T) {
	var imgCalls, vidCalls int32

	mux := http.NewServeMux()

	// Text: chat completions
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "ctxt1",
			"model": "evo-gpt-4o",
			"choices": []any{map[string]any{
				"message": map[string]any{"content": "A vivid sunset over a futuristic skyline"},
			}},
			"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 12, "total_tokens": 22},
		})
	})

	// Image: generation endpoint
	mux.HandleFunc("/v1/images/generations", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&imgCalls, 1)
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"task_id": "timg_pipeline"})
	})

	// Image: task polling
	mux.HandleFunc("/v1/tasks/timg_pipeline", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "timg_pipeline", "status": "completed",
			"data": map[string]any{"images": []any{"https://example.com/image.png"}},
		})
	})

	// Video: generation endpoint
	mux.HandleFunc("/v1/videos/generations", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&vidCalls, 1)
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"task_id": "tvid_pipeline"})
	})

	// Video: task polling
	mux.HandleFunc("/v1/tasks/tvid_pipeline", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "tvid_pipeline", "status": "completed",
			"data": map[string]any{"video_url": "https://example.com/video.mp4"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx := context.Background()
	timeout := 10 * time.Second

	// Build text model + agent (step 1)
	textModel, err := evolinktxt.New("evo-gpt-4o", evolinktxt.Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Timeout: timeout,
	})
	if err != nil {
		t.Fatalf("text model: %v", err)
	}
	textAgent, err := agent.New(agent.Config{
		ID:           "prompt-writer",
		Name:         "Prompt Writer",
		Model:        textModel,
		Instructions: "Write a vivid visual prompt.",
	})
	if err != nil {
		t.Fatalf("text agent: %v", err)
	}
	textStep, err := workflow.NewStep(workflow.StepConfig{ID: "text-to-prompt", Agent: textAgent})
	if err != nil {
		t.Fatalf("text step: %v", err)
	}

	// Build image model (step 2)
	imageModel, err := evolinkimg.New("evo-gpt-4o-images", evolinkimg.Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Timeout: timeout,
		Model:   evolinkimg.ModelGPT4O,
		Size:    "1:1",
		N:       1,
	})
	if err != nil {
		t.Fatalf("image model: %v", err)
	}

	// Build video model (step 3)
	videoModel, err := evolinkvid.New("evo-wan25-i2v", evolinkvid.Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Timeout: timeout,
		Model:   evolinkvid.ModelWan25ImageToVideo,
	})
	if err != nil {
		t.Fatalf("video model: %v", err)
	}

	// Assemble workflow
	wf, err := workflow.New(workflow.Config{
		Name: "test-media-pipeline",
		Steps: []workflow.Node{
			textStep,
			&imageNode{model: imageModel},
			&videoNode{model: videoModel},
		},
	})
	if err != nil {
		t.Fatalf("workflow: %v", err)
	}

	result, err := wf.Run(ctx, "A futuristic city at sunset", "test-session")
	if err != nil {
		t.Fatalf("pipeline run: %v", err)
	}

	// Verify image step was called and task ID stored
	if atomic.LoadInt32(&imgCalls) == 0 {
		t.Error("image generation endpoint was never called")
	}
	imgTaskID, ok := result.Get("image_task_id")
	if !ok || imgTaskID != "timg_pipeline" {
		t.Errorf("expected image_task_id=timg_pipeline, got %v (ok=%v)", imgTaskID, ok)
	}

	// Verify video step was called and task ID stored
	if atomic.LoadInt32(&vidCalls) == 0 {
		t.Error("video generation endpoint was never called")
	}
	vidTaskID, ok := result.Get("video_task_id")
	if !ok || vidTaskID != "tvid_pipeline" {
		t.Errorf("expected video_task_id=tvid_pipeline, got %v (ok=%v)", vidTaskID, ok)
	}
}
