package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	evolinkimg "github.com/jholhewres/agent-go/pkg/agentgo/models/evolink/image"
	evolinktxt "github.com/jholhewres/agent-go/pkg/agentgo/models/evolink/text"
	evolinkvid "github.com/jholhewres/agent-go/pkg/agentgo/models/evolink/video"
	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
	"github.com/jholhewres/agent-go/pkg/agentgo/workflow"
)

// imageNode wraps an evolink image model as a workflow.Node.
// It reads execCtx.Output as the prompt and stores the task ID back.
type imageNode struct {
	model *evolinkimg.Image
}

func (n *imageNode) GetID() string             { return "image-generation" }
func (n *imageNode) GetType() workflow.NodeType { return workflow.NodeTypeStep }
func (n *imageNode) Execute(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
	prompt := execCtx.Output
	if prompt == "" {
		prompt = execCtx.Input
	}
	resp, err := n.model.Invoke(ctx, &models.InvokeRequest{
		Messages: []*types.Message{types.NewUserMessage(prompt)},
	})
	if err != nil {
		return nil, fmt.Errorf("image generation failed: %w", err)
	}
	execCtx.Set("image_task_id", resp.ID)

	// Extract the generated image URL from the polling payload so the next
	// step can consume it. evolink image.Response.Data shape: {"images": [...]}.
	if data, ok := resp.Metadata.Extra["data"].(map[string]interface{}); ok {
		if images, ok := data["images"].([]interface{}); ok && len(images) > 0 {
			if url, ok := images[0].(string); ok {
				execCtx.Set("image_url", url)
			}
		}
	}

	execCtx.Output = fmt.Sprintf("image_task_id:%s", resp.ID)
	return execCtx, nil
}

// videoNode wraps an evolink video model as a workflow.Node.
// It reads the image task ID from the execution context.
type videoNode struct {
	model *evolinkvid.Video
}

func (n *videoNode) GetID() string             { return "video-generation" }
func (n *videoNode) GetType() workflow.NodeType { return workflow.NodeTypeStep }
func (n *videoNode) Execute(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
	prompt := execCtx.Input // original user prompt

	// Extract the generated image URL stored by the image step.
	imageURL, _ := execCtx.Get("image_url")
	imageURLStr, _ := imageURL.(string)
	if imageURLStr == "" {
		return nil, fmt.Errorf("video generation requires an image URL from the image step")
	}

	resp, err := n.model.Invoke(ctx, &models.InvokeRequest{
		Messages: []*types.Message{types.NewUserMessage(prompt)},
		Extra: map[string]interface{}{
			"model":      string(evolinkvid.ModelWan25ImageToVideo),
			"image_urls": []string{imageURLStr},
			"duration":   5,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("video generation failed: %w", err)
	}
	execCtx.Set("video_task_id", resp.ID)
	execCtx.Output = fmt.Sprintf("video_task_id:%s", resp.ID)
	return execCtx, nil
}

func main() {
	apiKey := os.Getenv("EVO_API_KEY")
	if apiKey == "" {
		log.Println("EVO_API_KEY is not set — skipping live API calls.")
		log.Println("Set EVO_API_KEY to run the full text→image→video pipeline.")
		log.Println("Example: EVO_API_KEY=your_key go run ./cmd/examples/evolink_media_pipeline/")
		return
	}

	ctx := context.Background()

	// Step 1: text model generates a visual prompt
	textModel, err := evolinktxt.New("evo-gpt-4o", evolinktxt.Config{APIKey: apiKey})
	if err != nil {
		log.Fatalf("failed to create text model: %v", err)
	}

	textAgent, err := agent.New(agent.Config{
		ID:           "prompt-writer",
		Name:         "Prompt Writer",
		Model:        textModel,
		Instructions: "You are a creative director. Given a concept, write a concise, vivid visual prompt (max 50 words) suitable for image generation. Return only the prompt text.",
	})
	if err != nil {
		log.Fatalf("failed to create text agent: %v", err)
	}

	textStep, err := workflow.NewStep(workflow.StepConfig{
		ID:    "text-to-prompt",
		Agent: textAgent,
	})
	if err != nil {
		log.Fatalf("failed to create text step: %v", err)
	}

	// Step 2: image model generates an image from the prompt
	imageModel, err := evolinkimg.New("evo-gpt-4o-images", evolinkimg.Config{
		APIKey: apiKey,
		Model:  evolinkimg.ModelGPT4O,
		Size:   "1:1",
		N:      1,
	})
	if err != nil {
		log.Fatalf("failed to create image model: %v", err)
	}

	// Step 3: video model generates a video from the image
	videoModel, err := evolinkvid.New("evo-wan25-i2v", evolinkvid.Config{
		APIKey: apiKey,
		Model:  evolinkvid.ModelWan25ImageToVideo,
	})
	if err != nil {
		log.Fatalf("failed to create video model: %v", err)
	}

	// Assemble the workflow: text step uses workflow.Step (agent-backed),
	// image and video steps use custom nodes that implement workflow.Node.
	wf, err := workflow.New(workflow.Config{
		Name: "Evolink Media Pipeline",
		Steps: []workflow.Node{
			textStep,
			&imageNode{model: imageModel},
			&videoNode{model: videoModel},
		},
	})
	if err != nil {
		log.Fatalf("failed to create workflow: %v", err)
	}

	result, err := wf.Run(ctx, "A futuristic city at sunset with flying vehicles", "")
	if err != nil {
		log.Fatalf("pipeline failed: %v", err)
	}

	imageTaskID, _ := result.Get("image_task_id")
	videoTaskID, _ := result.Get("video_task_id")

	fmt.Println("=== Evolink Media Pipeline completed ===")
	fmt.Printf("Image task ID : %v\n", imageTaskID)
	fmt.Printf("Video task ID : %v\n", videoTaskID)
	fmt.Printf("Final output  : %s\n", result.Output)
}
