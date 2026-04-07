# experimental/integrations

> **Warning: Experimental — API may change without notice.**

Thread-safe registry for third-party service integrations with optional health checks.

## Purpose

Provides a central `Registry` where integrations (external APIs, databases, services) are registered by name with an optional `Health` function. Supports listing all integrations and bulk health-checking via `CheckHealth`.

## Main Types

- `Integration` — `{Name, Description, Health func(ctx) (latency, error)}`
- `Registry` — `Register`, `List`, `CheckHealth`

## Minimal Example

```go
import "github.com/jholhewres/agent-go/pkg/agentgo/experimental/integrations"

reg := integrations.NewRegistry()
reg.Register(integrations.Integration{
    Name:        "slack",
    Description: "Slack webhook",
    Health: func(ctx context.Context) (time.Duration, error) {
        // ping endpoint
        return 0, nil
    },
})
results := reg.CheckHealth(ctx) // map[string]error
```

## Status

**experimental** — no consumers outside the package itself.
