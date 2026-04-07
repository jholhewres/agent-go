# media

Multimodal attachment normalisation for agent messages.

## Purpose

The `media` package defines the `Attachment` type and a `Normalize` helper that accepts many input shapes (slices, maps, pointers) and returns a canonical `[]Attachment` list. It also provides typed constructors for image, audio, and video payloads used by agents and workflow steps.

## Main Types

- `Attachment` — describes an external media resource (type, URL/path, content-type, name, metadata)
- `Normalize(input interface{}) ([]Attachment, error)` — coerces heterogeneous inputs into `[]Attachment`
- `NewImageAttachment`, `NewAudioAttachment`, `NewVideoAttachment` — typed constructors

## Minimal Example

```go
import "github.com/jholhewres/agent-go/pkg/agentgo/media"

attachments, err := media.Normalize(map[string]interface{}{
    "type": "image",
    "url":  "https://example.com/photo.jpg",
})
if err != nil {
    log.Fatal(err)
}
// attachments[0].Type == "image"
```

## Status

**stable** — used by `pkg/agentos`, `pkg/agentgo/workflow`.
