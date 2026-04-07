# otlp_tracing_agent

Demonstrates how to attach OpenTelemetry tracing to an AgentGo agent using the existing hook system. Spans are written to **stdout** — no collector or Docker required.

## Run

```bash
# Without an API key: hook wiring demo only (compiles + runs cleanly)
go run ./cmd/examples/otlp_tracing_agent/

# With an API key: full agent run with tool span + agent span emitted to stdout
OPENAI_API_KEY=sk-... go run ./cmd/examples/otlp_tracing_agent/
```

## What it shows

1. `NewStdoutTracerProvider()` — zero-config tracer for local development
2. `NewToolTracingHook(tracer)` — one span per tool call with status + duration
3. `NewAgentTracingHook(tracer)` — one span per agent Run with input/output length
4. Both hooks wired into `agent.Config` without touching `agent.go`

## Expected output (no API key)

```
No model configured. Verifying hook wiring only...
{
  "Name": "tool.calculator.add",
  "SpanContext": { ... },
  "Attributes": [
    { "Key": "tool.name", "Value": "calculator.add" },
    { "Key": "tool.status", "Value": "success" },
    ...
  ]
}
{
  "Name": "agent.run",
  "Attributes": [
    { "Key": "agent.status", "Value": "success" },
    ...
  ]
}
Hook wiring OK. Spans emitted above.
```
