# Production Deployment Guide

Take agent-go from "works on my laptop" to "running 99.9% in prod." This guide covers the patterns, architecture, and operational checklists for deploying agent-go systems safely to production.

## Overview

This guide is for **DevOps engineers, backend developers, and platform teams** deploying agent-go in production. We assume familiarity with containerization, databases, and observability tools.

**What you'll learn:**
- Reference production architecture
- PostgreSQL session storage (schema, pooling, multi-tenancy)
- Observability (distributed tracing, metrics, structured logging)
- Memory strategy (in-memory vs. hybrid summarization)
- Quality gates via eval framework
- Configuration via environment variables
- Secrets management best practices
- Docker Compose and Kubernetes deployment patterns
- Common pitfalls and pre-flight checklist

**Prerequisites:**
- Go 1.24+ installed
- Docker & Docker Compose (for local dev)
- Basic familiarity with PostgreSQL, REST APIs, and telemetry
- Understanding of agent-go core concepts (read `/guide/agent`, `/guide/team`, `/guide/workflow`)

---

## Reference Architecture

This diagram shows a typical production setup:

```
┌────────────────────────────────────────────────────────────┐
│                    Client Applications                      │
│         (Web, Mobile, CLI, Third-party Services)           │
└────────────────────────┬─────────────────────────────────────┘
                         │ REST / gRPC
                         ▼
         ┌───────────────────────────────────┐
         │   AgentOS REST API (stateless)    │
         │   (horizontal scale 1..N replicas)│
         └───────────┬───────────────────────┘
                     │
         ┌───────────┴───────────────────────┐
         │         Upstream Services         │
         ├─────────────────────────────────── │
         │ • OpenAI/Anthropic/Gemini APIs    │
         │ • Groq, Ollama, or other LLMs    │
         │ • External Tool/Data APIs         │
         └─────────────────────────────────── ┘
                     │
         ┌───────────┴───────────────────────────────────┐
         │         Persistent Backends                   │
         ├──────────────────────────────────────────────  │
         │ • PostgreSQL (session store, stateful)        │
         │ • Vector DB (ChromaDB, Pinecone, etc.)        │
         │ • Cache Layer (Redis, optional)               │
         └──────────────────────────────────────────────  ┘

         ┌──────────────────────────────────────────────┐
         │        Observability Pipeline                │
         ├──────────────────────────────────────────────┤
         │ • OpenTelemetry Collector (traces, metrics)  │
         │ • Jaeger / Tempo / Honeycomb (backend)       │
         │ • Prometheus (metrics scraping)              │
         │ • Structured logs (stdout → ELK / Splunk)    │
         └──────────────────────────────────────────────┘
```

**Key design principles:**
- **Stateless agent runtime** — Multiple replicas can scale horizontally.
- **Single stateful database** — PostgreSQL holds sessions; must be backed up.
- **Observability as a first-class concern** — Traces and metrics from agent execution.
- **Secrets externalized** — API keys and credentials via secrets manager, not env vars in code.

---

## Storage Configuration

### PostgreSQL Backend

Why PostgreSQL?
- **Transactions** — Atomic session state updates prevent race conditions.
- **Full-text search** — Query conversation history by content.
- **JSON columns** — Store flexible agent metadata and run history.
- **Built-in pub/sub** — LISTEN/NOTIFY for async workflows.
- **Proven at scale** — Battle-tested for multi-tenant SaaS.

#### Schema Overview

```
sessions table:
  ├─ id (uuid, primary key)
  ├─ user_id (uuid, tenant isolation)
  ├─ agent_id (string, which agent ran)
  ├─ conversation_history (jsonb, messages array)
  ├─ metadata (jsonb, custom agent state)
  ├─ token_usage (jsonb, {prompt, completion, reasoning})
  ├─ created_at (timestamp)
  ├─ updated_at (timestamp)
  ├─ last_accessed_at (timestamp)
  └─ indexes: (user_id, agent_id), (user_id, updated_at),
              GIN on conversation_history, GIN on metadata

evaluations table:
  ├─ id (uuid)
  ├─ session_id (uuid, fk → sessions)
  ├─ eval_type (string: accuracy, performance, reliability)
  ├─ score (float, 0.0..1.0)
  ├─ details (jsonb, evaluator output)
  ├─ created_at (timestamp)
  └─ indexes: (session_id), (eval_type, created_at)
```

> **TODO(post-S1)**: link to the final PostgreSQL schema migration once Sprint 1 lands in `pkg/agentgo/storage/postgres/schema.sql`.

#### Connection Pool Tuning

Recommended configuration for production:

```go
// Pseudocode; see pkg/agentgo/storage/postgres for the latest API
type PostgresConfig struct {
    // Max concurrent connections in pool
    MaxOpenConns int           // default: 25, set to (# replicas * 4)

    // Idle connections kept open
    MaxIdleConns int           // default: 5, set to MaxOpenConns / 5

    // How long an idle conn is kept before close
    ConnMaxIdleTime time.Duration  // default: 5m

    // Max lifetime of a conn before reuse
    ConnMaxLifetime time.Duration  // default: 15m

    // Timeout for acquiring a conn from pool
    DialTimeout time.Duration  // default: 5s
}
```

**How to size:**
- **MaxOpenConns** = (number of agent-go replicas) × 4. Example: 3 replicas × 4 = 12 connections.
- **MaxIdleConns** = MaxOpenConns / 5.
- Monitor pg_stat_activity to ensure you're not hitting `max_connections` (default 100).

#### Migration Workflow

Before first production run:

```bash
# 1. Define or update schema
# (Once Sprint 1 migration APIs land, use something like:)
export DATABASE_URL="postgres://user:pass@localhost:5432/agentgo_prod"

# 2. Run migrations (CLI or SDK)
agentgo migrate up

# 3. Verify schema
psql $DATABASE_URL -c "\dt"

# 4. Create initial indexes
# (Included in migration, but verify they exist)
psql $DATABASE_URL -c "SELECT * FROM pg_indexes WHERE schemaname='public';"
```

> **TODO(post-S1)**: Replace with actual CLI commands once `agentgo migrate` is implemented.

#### Multi-Tenant Isolation

For SaaS deployments with multiple organizations:

```sql
-- Add tenant_id column to sessions
ALTER TABLE sessions ADD COLUMN tenant_id UUID NOT NULL;

-- Create indexes on tenant_id for fast lookups
CREATE INDEX idx_sessions_tenant_user
  ON sessions(tenant_id, user_id);

-- Add tenant_id to RLS (Row-Level Security) policy
CREATE POLICY rls_sessions_tenant ON sessions
  USING (tenant_id = current_user_id);
```

Then in application code:

```go
// Always filter by tenant_id in queries
// (See pkg/agentgo/storage for the latest API)
query := `
  SELECT * FROM sessions
  WHERE tenant_id = $1 AND user_id = $2
`
```

#### Backup & Recovery

**Backup strategy:**
```bash
# Daily full backup at 02:00 UTC (cron job)
0 2 * * * pg_dump -Fc $DATABASE_URL > /backups/agentgo-$(date +\%Y\%m\%d).dump

# Backup size: ~5 GB per million sessions (estimate)
# Keep 30 days of backups
find /backups -mtime +30 -delete
```

**Point-in-Time Recovery (PITR):**
```bash
# Restore to a specific timestamp
pg_restore -d agentgo_prod /backups/agentgo-20260407.dump

# Or use PostgreSQL WAL archiving (AWS RDS does this automatically)
# https://www.postgresql.org/docs/current/continuous-archiving.html
```

**Retention policy:**
- Keep 7 days of daily dumps locally.
- Archive to S3/GCS every 7 days for 1 year.
- For compliance: check your data residency and retention requirements.

---

## Observability

### Distributed Tracing (OTLP)

**Why it matters:**
Agent execution involves multiple steps (input parsing, tool calls, LLM invocation, result aggregation). Traces let you:
- Correlate requests across service boundaries.
- Identify bottlenecks (slow LLM, slow tool, network latency).
- Debug multi-agent teams and workflows.
- Sample traces for cost control (not all requests need full instrumentation).

#### Configuration

> **TODO(post-S1)**: Link to the final OTLP config struct once available in `pkg/agentgo/observability/otlp.go`.

Minimal setup:

```bash
# Environment variables for OTLP exporter
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"  # gRPC endpoint
export OTEL_SERVICE_NAME="agentgo-prod"
export OTEL_DEPLOYMENT_ENVIRONMENT="production"

# Optional: sampling (head sampling: ratio of requests to trace)
export OTEL_TRACES_SAMPLER="parentbased_traceidratio"
export OTEL_TRACES_SAMPLER_ARG="0.1"  # Trace 10% of requests
```

#### Span Attributes

The framework emits the following span attributes:

| Attribute | Example | Type | Notes |
|-----------|---------|------|-------|
| `agent.name` | "DataProcessor" | string | Agent name from config |
| `agent.model` | "gpt-4o" | string | Model ID |
| `agent.tools` | ["calculator", "http"] | array | Tool names |
| `tool.name` | "calculator" | string | Tool being executed |
| `tool.duration_ms` | 145 | number | Time in milliseconds |
| `tokens.prompt` | 500 | number | Prompt tokens consumed |
| `tokens.completion` | 150 | number | Completion tokens consumed |
| `tokens.reasoning` | 50 | number | (o1/o3 models) reasoning tokens |
| `span.kind` | "INTERNAL" / "CLIENT" | string | OpenTelemetry standard |
| `error` | "max_loops_exceeded" | string | Error code (if failed) |

#### Example: Jaeger + Collector Compose

```yaml
# docker-compose.yml excerpt
version: '3.8'
services:
  # Jaeger all-in-one for local dev/testing
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "6831:6831/udp"    # Jaeger agent (thrift compact)
      - "16686:16686"      # Web UI: http://localhost:16686
    environment:
      COLLECTOR_OTLP_ENABLED: "true"

  # OpenTelemetry Collector (production pattern)
  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports:
      - "4317:4317"        # gRPC receiver
      - "4318:4318"        # HTTP receiver
    volumes:
      - ./otel-config.yaml:/etc/otel-collector-config.yaml
    command: ["--config=/etc/otel-collector-config.yaml"]
    depends_on:
      - jaeger

  # AgentOS service
  agentos:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: "postgres://user:pass@postgres:5432/agentgo"
      OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
      OTEL_SERVICE_NAME: "agentgo-prod"
    depends_on:
      - postgres
      - otel-collector

  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: agentgo
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data

volumes:
  postgres-data:
```

**otel-config.yaml** for production:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"
      http:
        endpoint: "0.0.0.0:4318"

processors:
  batch:
    send_batch_size: 100
    timeout: 10s
  memory_limiter:
    check_interval: 5s
    limit_mib: 512

exporters:
  jaeger:
    endpoint: "jaeger:14250"
  # For production, add Tempo, Honeycomb, or other backends
  # otlp:
  #   endpoint: "tempo:4317"

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [jaeger]
```

#### Sampling Strategy

**Head sampling** (sample at source):
- Best for: High-volume, exploratory monitoring.
- Config: `OTEL_TRACES_SAMPLER_ARG=0.01` (1% of requests).
- Cost: Low, but you may miss rare errors.

**Tail sampling** (sample after seeing full trace):
- Best for: Error debugging, high-latency detection.
- Requires Collector's tail_sampling processor.
- Config: Keep all traces in buffer, sample by outcome (errors always, >5s latency always, 10% others).

**Adaptive sampling**:
- Adjust ratio based on error rate or load.
- Start with 10% head sampling; tune based on cost and SLO.

#### Resource Attributes

Always include:

```go
resource.Attributes{
    "service.name":           "agentgo-prod",
    "service.version":        "1.5.0",
    "deployment.environment": "production",
    "deployment.region":      "us-east-1",
    "container.id":           os.Getenv("HOSTNAME"),  // pod name in k8s
}
```

### Metrics

**What to measure:**

| Metric | Type | Purpose |
|--------|------|---------|
| `agent_run_duration_seconds` | Histogram | P50, P95, P99 latency per agent |
| `agent_run_total` | Counter | Throughput (runs/min) |
| `agent_errors_total` | Counter | Errors by type (max_loops, tool_error, etc.) |
| `tokens_used` | Counter | Prompt/completion/reasoning tokens |
| `tool_call_duration_seconds` | Histogram | Time per tool (calculator vs. http) |
| `session_active_count` | Gauge | Active sessions at any time |

> **TODO(post-S1)**: Once Prometheus instrumentation lands, see `pkg/agentgo/observability/metrics.go` for the official exporter.

**Prometheus scrape config:**

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'agentgo'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
    scrape_timeout: 5s
```

### Logging

Use structured logging via `log/slog` (Go 1.21+):

```bash
# Enable JSON output (machine-readable)
export AGENT_LOG_FORMAT="json"
export AGENT_LOG_LEVEL="info"

# Forward stdout to ELK, Splunk, or Datadog
# (Usually via container orchestrator or log agent)
```

**Example log line:**

```json
{
  "time": "2026-04-07T12:34:56Z",
  "level": "INFO",
  "msg": "agent_run_complete",
  "agent_name": "DataProcessor",
  "session_id": "uuid-...",
  "duration_ms": 1234,
  "tokens": {"prompt": 500, "completion": 150}
}
```

**PII scrubbing:**
- Never log raw API keys, user emails, or credit cards.
- Use `log.Redacted()` for sensitive fields.
- Example: `"user_input": "[REDACTED]"` if input contains PII patterns.

---

## Memory Strategy

Agents need to remember conversation history. Choose the right backend:

### Memory Backend Comparison

| Backend | Use Case | Pros | Cons |
|---------|----------|------|------|
| **InMemory** | Single-process, dev/test | Fast, no setup | Lost on restart, no persistence |
| **HybridMemory** | Production with summarization | Balances cost & context | Summary quality varies |
| **SummarizingMemory** | Long conversations (>10k tokens) | Cheap, scales to 100k+ msgs | Requires summarization latency |

> **TODO(post-S1)**: Link to memory implementations in `pkg/agentgo/memory/`.

### Choosing the Right Backend

**For chat bots (short, interactive):**
```go
// Use hybrid: keep recent messages in full, old ones summarized
memory.NewHybridMemory(&memory.HybridConfig{
    MaxRecentMessages: 20,        // last 20 messages in full
    SummarizationThreshold: 100,  // summarize if >100 messages
    Model: openaiModel,           // cheap model for summarization
})
```

**For long-running agents (research, data processing):**
```go
// Use summarizing: everything gets compressed
memory.NewSummarizingMemory(&memory.SummarizingConfig{
    ContextWindowSize: 4000,  // keep ~4k tokens of context
    SummaryModel: groqModel,  // fast & cheap (Groq, not GPT-4)
    CacheSummaries: true,     // don't re-summarize same batch
})
```

**For internal systems (logging, guardrails):**
```go
// Use in-memory: no DB latency, clear truncation policy
memory.NewMemory(&memory.Config{
    MessageLimit: 100,  // drop oldest when >100
})
```

### Summarization Tuning

If using `HybridMemory` or `SummarizingMemory`:

```bash
# Environment variables
export AGENT_MEMORY_SUMMARIZE_AT=150  # trigger at 150 messages
export AGENT_MEMORY_PRESERVE_LAST=30  # always keep last 30 in full
export AGENT_MEMORY_SUMMARY_MODEL="gpt-4o-mini"  # cheap + fast
```

**Summary prompt design:**
```
You are a conversation summarizer. Produce a concise 1-2 paragraph summary
of the conversation so far, preserving key facts, decisions, and context
for the agent to continue effectively.

Conversation:
{history}

Summary:
```

**Evaluating quality:**
- Run eval: does the agent still answer accurately after summarization?
- Compare token cost: summarization latency + cost vs. full-context cost.
- If summary loses >10% accuracy, use shorter summarization threshold or longer `PreserveLast`.

---

## Quality Gates (Eval Framework)

Before deploying new agent versions, run evaluations to catch regressions.

### Eval in CI

**Workflow:** Write eval suite → Run in CI → Gate deployment on pass rate.

Example GitHub Actions workflow (25 lines):

```yaml
# .github/workflows/eval.yml
name: Eval Quality Gates
on:
  pull_request:
    paths:
      - 'pkg/agentgo/**'
      - 'cmd/agentos/**'

jobs:
  eval:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run Eval Suite
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          go test -v -run TestEval ./test/evals/... -junit-out=results.xml

      - name: Check Pass Rate
        run: |
          # Parse JUnit XML; fail if pass_rate < 0.9
          python3 scripts/check_eval_threshold.py results.xml 0.9

      - name: Upload Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: eval-results
          path: results.xml
```

### Evaluator Selection

Which evaluator to use?

| Evaluator | Checks | Cost | Speed | Example |
|-----------|--------|------|-------|---------|
| **Accuracy** | Answer correctness (exact match, semantic similarity) | Medium | Fast | "Is the final answer correct?" |
| **Performance** | Latency, token usage | Low | Instant | "Did it finish <2s? Used <500 tokens?" |
| **Reliability** | Error handling, edge cases | High | Slow | "Does it fail gracefully on bad input?" |
| **Judge** (LLM-as-judge) | Holistic quality, reasoning | High | Slow | "Is the response helpful and harmless?" |

**When to use each:**
- **Accuracy**: Always (gate deployment on this).
- **Performance**: For cost-sensitive apps; alert if tokens spike.
- **Reliability**: Before major model changes.
- **Judge**: Subjective quality; useful for content generation, reasoning tasks.

---

## Configuration via Environment Variables

Complete reference for production:

| Variable | Default | Type | Required | Description |
|----------|---------|------|----------|-------------|
| `DATABASE_URL` | (none) | string | YES | PostgreSQL connection: `postgres://user:pass@host:5432/db` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4317` | string | NO | OpenTelemetry collector gRPC endpoint |
| `OTEL_SERVICE_NAME` | `agentgo` | string | NO | Service name in traces/metrics |
| `OPENAI_API_KEY` | (none) | string | if using OpenAI | OpenAI API key |
| `ANTHROPIC_API_KEY` | (none) | string | if using Claude | Anthropic API key |
| `GROQ_API_KEY` | (none) | string | if using Groq | Groq API key |
| `GEMINI_API_KEY` | (none) | string | if using Gemini | Google Gemini API key |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | string | if using Ollama | Ollama server URL |
| `AGENT_LOG_LEVEL` | `info` | string | NO | `debug`, `info`, `warn`, `error` |
| `AGENT_LOG_FORMAT` | `text` | string | NO | `text` or `json` |
| `AGENT_MAX_LOOPS` | `10` | int | NO | Max tool-call iterations per agent.Run() |
| `AGENT_MEMORY_TYPE` | `hybrid` | string | NO | `inmemory`, `hybrid`, `summarizing` |
| `AGENT_MEMORY_SUMMARIZE_AT` | `150` | int | NO | Trigger summarization at N messages |
| `VECTOR_DB_TYPE` | (none) | string | NO | `chromadb`, `pinecone`, or omit for no RAG |
| `CHROMADB_ENDPOINT` | `http://localhost:8000` | string | if using ChromaDB | ChromaDB service endpoint |
| `SERVER_PORT` | `8080` | int | NO | HTTP server port |
| `SERVER_SHUTDOWN_TIMEOUT` | `30s` | duration | NO | Graceful shutdown timeout |

---

## Secrets Management

**DO NOT:**
```bash
# Bad: hardcoded keys
export OPENAI_API_KEY="sk-..."
git commit .env
```

**DO:**
```bash
# Good: externalized secrets
# AWS Secrets Manager
export OPENAI_API_KEY=$(aws secretsmanager get-secret-value --secret-id agentgo/prod/openai | jq -r .SecretString)

# Or: inject at runtime
# Kubernetes secret mounted as env var:
# spec.containers[0].env:
#   - name: OPENAI_API_KEY
#     valueFrom:
#       secretKeyRef:
#         name: agentgo-secrets
#         key: openai-api-key
```

**Rotation:**
- Rotate secrets every 90 days.
- Plan rotation window (low-traffic time); brief API errors expected.
- Test rotation in staging first.

**Scope:**
- API keys: Read-only when possible (e.g., GPT-4 vision-only, not fine-tuning).
- Database: Separate user for agents (SELECT, INSERT, UPDATE on sessions only; no DROP).
- Service accounts: One per deployment environment (dev, staging, prod).

---

## Docker Compose Example

Complete, production-ready `docker-compose.yml`:

```yaml
version: '3.8'

services:
  # PostgreSQL (stateful backend)
  postgres:
    image: postgres:16-alpine
    container_name: agentgo-postgres
    environment:
      POSTGRES_USER: agentgo
      POSTGRES_PASSWORD: changeme123  # Use secrets in prod
      POSTGRES_DB: agentgo_prod
      POSTGRES_INITDB_ARGS: "-c shared_preload_libraries=pg_stat_statements"
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
      # Mount migration scripts if needed
      # - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U agentgo"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - agentgo-net

  # OpenTelemetry Collector
  otel-collector:
    image: otel/opentelemetry-collector:latest
    container_name: agentgo-otel
    command: ["--config=/etc/otel-collector-config.yaml"]
    ports:
      - "4317:4317"  # gRPC
      - "4318:4318"  # HTTP
    volumes:
      - ./otel-config.yaml:/etc/otel-collector-config.yaml
    depends_on:
      - jaeger
    networks:
      - agentgo-net

  # Jaeger for local tracing (replace with Tempo/Honeycomb in prod)
  jaeger:
    image: jaegertracing/all-in-one:latest
    container_name: agentgo-jaeger
    ports:
      - "6831:6831/udp"
      - "16686:16686"  # Web UI
    environment:
      COLLECTOR_OTLP_ENABLED: "true"
    networks:
      - agentgo-net

  # AgentOS REST API (stateless, horizontally scalable)
  agentos:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: agentgo-api
    ports:
      - "8080:8080"
    environment:
      # Database
      DATABASE_URL: "postgres://agentgo:changeme123@postgres:5432/agentgo_prod"

      # Observability
      OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
      OTEL_SERVICE_NAME: "agentgo-prod"
      AGENT_LOG_LEVEL: "info"
      AGENT_LOG_FORMAT: "json"

      # Agent config
      AGENT_MAX_LOOPS: "10"
      AGENT_MEMORY_TYPE: "hybrid"

      # LLM API keys (load from secrets manager in prod)
      OPENAI_API_KEY: "${OPENAI_API_KEY}"
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"

      # Server
      SERVER_PORT: "8080"
      SERVER_SHUTDOWN_TIMEOUT: "30s"
    depends_on:
      postgres:
        condition: service_healthy
      otel-collector:
        condition: service_started
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - agentgo-net
    restart: unless-stopped

  # (Optional) ChromaDB for RAG
  chromadb:
    image: chromadb/chroma:latest
    container_name: agentgo-chromadb
    ports:
      - "8000:8000"
    networks:
      - agentgo-net
    # Uncomment to enable:
    # profiles:
    #   - with-vectordb

networks:
  agentgo-net:
    driver: bridge

volumes:
  postgres-data:
    driver: local
```

**Usage:**

```bash
# Start all services
docker-compose up -d

# Verify
curl http://localhost:8080/health

# View logs
docker-compose logs -f agentos

# Scale API replicas (for load testing)
docker-compose up -d --scale agentos=3

# Shutdown
docker-compose down
```

---

## Scaling Considerations

### Horizontal Scaling

**Agent runtime is stateless** — Deploy N replicas behind a load balancer:

```yaml
# Kubernetes example
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentos
spec:
  replicas: 3  # Start with 3; autoscale based on CPU/memory
  selector:
    matchLabels:
      app: agentos
  template:
    metadata:
      labels:
        app: agentos
    spec:
      containers:
      - name: agentos
        image: agentgo:1.5.0
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "1"
            memory: "1Gi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
```

**PostgreSQL is the scaling bottleneck:**
- Connection pool: max 25 connections per replica (adjust for your load).
- Max connections in PostgreSQL: `max_connections` (default 100).
- For 4 replicas × 25 conns = 100, you're at the limit.
- Solution: Use PgBouncer (connection pooler) in front of PostgreSQL.

### PgBouncer for Connection Pooling

```ini
# pgbouncer.ini
[databases]
agentgo_prod = host=postgres port=5432 dbname=agentgo_prod

[pgbouncer]
pool_mode = transaction         # Cheapest; one conn per transaction
max_client_conn = 1000         # Clients from app layer
default_pool_size = 25         # Conn pool per database
reserve_pool_size = 5
reserve_pool_timeout = 3

[users]
agentgo = "password"
```

Then:
```bash
# Point DATABASE_URL to PgBouncer (port 6432)
export DATABASE_URL="postgres://agentgo:password@pgbouncer:6432/agentgo_prod"
```

### Rate Limiting

Upstream LLM APIs have rate limits. Implement backpressure:

```go
// Pseudocode
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(
    rate.Limit(100),  // 100 requests/sec to LLM
    1,                // burst of 1
)

if !limiter.Allow() {
    return fmt.Errorf("rate limit exceeded; retry after 1s")
}
```

**Cost controls:**
- Track token usage per session/user.
- Set session budget: `MAX_TOKENS_PER_SESSION=100000`.
- Reject if exceeded; charge user and alert ops.

---

## Common Pitfalls

1. **Forgetting to call shutdown on TracerProvider**
   - Buffered spans are lost on process exit. Always call `tp.Shutdown(ctx)` in defer.

2. **Connection pool size > PostgreSQL max_connections**
   - Result: "too many connections" errors. Check `SHOW max_connections;` in postgres.
   - Fix: Either increase `max_connections`, or reduce pool size per replica.

3. **Running migrations while app is live**
   - Risk: Schema mismatch between code and DB. Schedule during low-traffic window, take backup first.

4. **Mocking LLMs in tests but not asserting prompt content**
   - Risk: Prompt injection, PII leakage in production. Test prompt content, not just responses.

5. **No backup before major config change**
   - Risk: Data loss if rollback needed. Always backup sessions before ALTER TABLE.

6. **Storing API keys in environment variables without rotation**
   - Risk: Key compromise is undetected. Use secrets manager with rotation.

7. **Summarization model different from production agent model**
   - Risk: Summarization fails or quality degrades if model is unavailable. Test failover.

8. **Not monitoring token usage**
   - Risk: Surprise bills. Set up metrics and alerts for tokens/hour.

9. **Sampling traces at 100% in production**
   - Risk: Massive cost and storage. Start at 10%, adjust based on volume.

10. **Ignoring slow database queries**
    - Risk: Latency spikes on peak load. Use `EXPLAIN ANALYZE` on slow queries; add indexes.

---

## Pre-Flight Checklist

Before promoting to production:

- [ ] **Database**
  - [ ] PostgreSQL instance provisioned with 30-day backup retention
  - [ ] Schema migrations tested and passing locally
  - [ ] Connection pool tuned (MaxOpenConns = replicas × 4)
  - [ ] PgBouncer (or equivalent) in front of DB if >4 replicas

- [ ] **Observability**
  - [ ] OpenTelemetry collector running and receiving traces
  - [ ] Jaeger/Tempo/Honeycomb backend accessible and healthy
  - [ ] Sampling ratio set (10% for production, adjust as needed)
  - [ ] Prometheus scrape endpoint tested (`/metrics`)
  - [ ] Log aggregation pipeline running (ELK, Splunk, etc.)
  - [ ] PII scrubbing rules configured and tested

- [ ] **Configuration**
  - [ ] All environment variables set (DATABASE_URL, OTEL_*, LLM_*_API_KEY)
  - [ ] Secrets manager (AWS/GCP/Vault) provisioned; keys rotated
  - [ ] No secrets hardcoded in code or Docker image
  - [ ] SERVER_SHUTDOWN_TIMEOUT set to graceful window (30s)

- [ ] **Memory & Context**
  - [ ] Memory backend tested (InMemory/Hybrid/Summarizing) under load
  - [ ] Summarization model (if used) is fast and cheap (Groq, GPT-4o-mini)
  - [ ] Summary quality validated via eval suite
  - [ ] Max loop count appropriate for workload (AGENT_MAX_LOOPS)

- [ ] **Eval Framework**
  - [ ] Eval suite written and passing locally (>90% pass rate)
  - [ ] CI workflow configured to run evals on PRs
  - [ ] Quality gate threshold defined and enforced (pass_rate >= 0.9)
  - [ ] Fallback plan if eval fails (manual review, canary rollout)

- [ ] **Load & Security**
  - [ ] Load test: 100 RPS for 5 min; P99 latency <2s
  - [ ] LLM rate limits tested and failover works
  - [ ] Cost limits enforced (token budget per session)
  - [ ] SQL injection prevention (use parameterized queries)
  - [ ] API auth configured (OAuth, JWT, or internal service account)

- [ ] **Deployment**
  - [ ] Docker image built and scanned for vulnerabilities
  - [ ] Kubernetes manifests (or Docker Compose) reviewed
  - [ ] Health checks respond correctly (HTTP 200 on /health)
  - [ ] Graceful shutdown tested (SIGTERM → close DB → exit)
  - [ ] Rolling update strategy defined (max surge, max unavailable)

- [ ] **Monitoring & Alerting**
  - [ ] Alerting rules configured (5xx error rate, latency, DB connections)
  - [ ] On-call rotation and runbook in place
  - [ ] Incident response process documented
  - [ ] Post-mortem template available for incidents

- [ ] **Documentation**
  - [ ] Runbook for common issues (DB full, OOM, rate limit)
  - [ ] Disaster recovery plan (restore from backup, promote replica)
  - [ ] Team trained on deployment and troubleshooting

---

## Next Steps

1. **Read related docs:**
   - [`/advanced/architecture`](./architecture.md) — System design
   - [`/advanced/observability`](./observability.md) — SSE streams and Logfire
   - [`/advanced/testing`](./testing.md) — Quality gates and eval framework
   - [`/guide/memory`](../guide/memory.md) — Memory backends deep dive

2. **Set up locally:**
   ```bash
   docker-compose up -d
   curl http://localhost:8080/health
   curl -X POST http://localhost:8080/api/v1/agents/test/run \
     -H "Content-Type: application/json" \
     -d '{"input": "What is 2+2?"}'
   ```

3. **Deploy to staging first:**
   - Test full deployment flow (migrations, secrets, TLS).
   - Run evals; verify alerting works.
   - Load test; measure baseline latency and cost.

4. **Plan production rollout:**
   - Canary: 10% traffic to new version, monitor for 1 hour.
   - Blue-green: Run v1 and v2 in parallel, switch traffic at cutoff.
   - Rollback plan: DB migration rollback, previous image version.

---

**Questions? Issues?** Open a GitHub issue or join the discussions at [github.com/jholhewres/agent-go](https://github.com/jholhewres/agent-go).
