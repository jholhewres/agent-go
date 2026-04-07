# pkg/agentgo/observability/otel

OpenTelemetry tracing integration for AgentGo. Provides a tracer provider factory and hook implementations that plug into the existing agent hook system **without modifying `agent.go`**.

## Why OTel

OpenTelemetry is the CNCF standard for distributed tracing. Using OTel means spans from AgentGo agents can be collected by any compatible backend: Jaeger, Zipkin, Grafana Tempo, Datadog, Honeycomb, Logfire, and many others — with zero code changes to switch collectors.

## Quick Start

```go
import (
    "context"
    "log"

    "github.com/jholhewres/agent-go/pkg/agentgo/agent"
    "github.com/jholhewres/agent-go/pkg/agentgo/hooks"
    agentootel "github.com/jholhewres/agent-go/pkg/agentgo/observability/otel"
)

func main() {
    ctx := context.Background()

    // 1. Create a tracer provider (OTLP HTTP to a local collector).
    tp, shutdown, err := agentootel.NewTracerProvider(ctx, agentootel.Config{
        ServiceName:    "my-agent-service",
        ServiceVersion: "1.0.0",
        Endpoint:       "http://localhost:4318", // default
        SamplingRate:   1.0,                     // sample everything
        ResourceAttributes: map[string]string{
            "deployment.environment": "production",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer shutdown(ctx)

    tracer := tp.Tracer("my-agent-service")

    // 2. Create hooks.
    toolHook := agentootel.NewToolTracingHook(tracer)
    preHook, postHook := agentootel.NewAgentTracingHook(tracer)

    // 3. Wire hooks into the agent — no agent.go modification needed.
    ag, _ := agent.New(agent.Config{
        Name:      "my-agent",
        Model:     myModel,
        ToolHooks: []hooks.ToolHook{toolHook},
        PreHooks:  []hooks.Hook{preHook},
        PostHooks: []hooks.Hook{postHook},
    })

    output, _ := ag.Run(ctx, "What is 2 + 2?")
    _ = output
}
```

## Local Dev / Tests (no collector)

Use the stdout exporter — spans are printed as JSON to stdout:

```go
tp, shutdown, err := agentootel.NewStdoutTracerProvider()
```

See `cmd/examples/otlp_tracing_agent/` for a runnable example.

## Exported Spans

### `agent.run` (pre/post hook pair)

| Attribute | Type | Description |
|---|---|---|
| `agent.id` | string | Agent ID |
| `input.length` | int | Character length of input |
| `output.length` | int | Character length of output |
| `agent.status` | string | `"success"` or `"error"` |
| `agent.duration_ms` | float64 | Wall-clock run duration |

### `tool.<name>` (ToolTracingHook)

| Attribute | Type | Description |
|---|---|---|
| `tool.name` | string | Function name |
| `tool.call_id` | string | Unique call ID |
| `agent.id` | string | Owning agent ID |
| `tool.status` | string | `"success"` or `"error"` |
| `tool.duration_ms` | float64 | Execution duration |

## Endpoints

| Protocol | Endpoint format | Config field |
|---|---|---|
| OTLP HTTP (default) | `http://host:4318` | `Config.Endpoint` |
| Jaeger all-in-one | `http://localhost:4318` | same |
| Grafana Agent | `http://localhost:4318` | same |

For gRPC (`otlptracehttp` → `otlptracegrpc`), import `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` and build a custom provider using `NewTracerProvider` as a reference.

## Recommended Resource Attributes

```go
ResourceAttributes: map[string]string{
    "deployment.environment": "production",
    "service.instance.id":    hostname,
    "k8s.pod.name":           podName,
}
```

## Sampling Strategy

The provider uses `ParentBased(TraceIDRatioBased(SamplingRate))`:

- `SamplingRate = 1.0` — sample every trace (default, good for dev/test)
- `SamplingRate = 0.1` — sample 10% (typical for high-traffic production)
- `SamplingRate = 0.0` — drop all traces (disable tracing without removing hooks)

Parent-based sampling ensures that if the parent span is sampled, all child spans in the same trace are also sampled, keeping traces coherent.
