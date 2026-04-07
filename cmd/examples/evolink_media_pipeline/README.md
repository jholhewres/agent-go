# Evolink Media Pipeline

Demonstrates a three-step text→image→video pipeline using the EvoLink provider.

## Prerequisites

- Go 1.21+
- An EvoLink API key: <https://evolink.ai>

```bash
export EVO_API_KEY=your_api_key_here
```

If `EVO_API_KEY` is not set, the program prints a usage hint and exits cleanly
without making any API calls.

## How to run

```bash
go run ./cmd/examples/evolink_media_pipeline/
```

Or build first:

```bash
go build -o bin/evolink_media_pipeline ./cmd/examples/evolink_media_pipeline/
./bin/evolink_media_pipeline
```

## What the pipeline does

| Step | Model | Input | Output |
|------|-------|-------|--------|
| 1 — Prompt Writer | `evo-gpt-4o` (text) | User concept | Concise visual prompt |
| 2 — Image Generation | `evo-gpt-4o-images` (image) | Visual prompt | Image task ID |
| 3 — Video Generation | `wan2.5-i2v` (video) | Image task ID + prompt | Video task ID |

The three steps are chained inside a `workflow.Workflow`:

- Step 1 uses a standard `workflow.Step` backed by an `agent.Agent`.
- Steps 2 and 3 use lightweight custom nodes that implement `workflow.Node`
  directly, since image and video models do not conform to the text-chat
  interface expected by `agent.Agent`.

## Expected output

```
=== Evolink Media Pipeline completed ===
Image task ID : timg_xxxxxxxx
Video task ID : tvid_xxxxxxxx
Final output  : video_task_id:tvid_xxxxxxxx
```

Task IDs are EvoLink async task identifiers. Poll
`GET /v1/tasks/{id}` (or use `evolink.Client.PollTask`) to retrieve the
generated asset URLs once the tasks reach `completed` status.
