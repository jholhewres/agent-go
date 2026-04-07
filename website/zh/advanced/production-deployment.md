# 生产部署指南

让 agent-go 从"在我的笔记本电脑上运行"升级到"在生产环境中运行 99.9% 的时间"。本指南涵盖了安全部署 agent-go 系统到生产环境的模式、架构和运维清单。

## 概览

本指南面向在生产环境中部署 agent-go 的 **DevOps 工程师、后端开发者和平台团队**。我们假设您熟悉容器化、数据库和可观测性工具。

**您将学到：**
- 参考生产架构
- PostgreSQL 会话存储（模式、连接池、多租户）
- 可观测性（分布式追踪、指标、结构化日志）
- 内存策略（内存 vs. 混合总结）
- 通过评估框架进行质量控制
- 通过环境变量进行配置
- 密钥管理最佳实践
- Docker Compose 和 Kubernetes 部署模式
- 常见陷阱和飞行前清单

**前置要求：**
- Go 1.24+ 已安装
- Docker & Docker Compose（用于本地开发）
- 熟悉 PostgreSQL、REST API 和遥测工具
- 了解 agent-go 核心概念（阅读 `/guide/agent`、`/guide/team`、`/guide/workflow`）

---

## 参考架构

此图显示典型的生产设置：

```
┌────────────────────────────────────────────────────────────┐
│                    客户端应用程序                          │
│      （网页、移动应用、CLI、第三方服务）                   │
└────────────────────────┬─────────────────────────────────────┘
                         │ REST / gRPC
                         ▼
         ┌───────────────────────────────────┐
         │   AgentOS REST API（无状态）      │
         │  （可水平扩展至 1..N 个副本）    │
         └───────────┬───────────────────────┘
                     │
         ┌───────────┴───────────────────────┐
         │         上游服务                  │
         ├──────────────────────────────────  │
         │ • OpenAI/Anthropic/Gemini APIs    │
         │ • Groq、Ollama 或其他 LLM         │
         │ • 外部工具/数据 API               │
         └──────────────────────────────────  ┘
                     │
         ┌───────────┴───────────────────────────────────┐
         │         持久化后端                          │
         ├──────────────────────────────────────────────  │
         │ • PostgreSQL（会话存储，有状态）            │
         │ • 向量数据库（ChromaDB、Pinecone 等）       │
         │ • 缓存层（Redis，可选）                    │
         └──────────────────────────────────────────────  ┘

         ┌──────────────────────────────────────────────┐
         │        可观测性管道                         │
         ├──────────────────────────────────────────────┤
         │ • OpenTelemetry 收集器（追踪、指标）        │
         │ • Jaeger / Tempo / Honeycomb（后端）        │
         │ • Prometheus（指标爬取）                   │
         │ • 结构化日志（stdout → ELK / Splunk）       │
         └──────────────────────────────────────────────┘
```

**关键设计原则：**
- **无状态代理运行时** — 多个副本可以水平扩展。
- **单一有状态数据库** — PostgreSQL 保存会话；必须备份。
- **可观测性是一流的关注点** — 代理执行中的追踪和指标。
- **密钥外部化** — 通过密钥管理器而非代码中的环境变量。

---

## 存储配置

### PostgreSQL 后端

为什么选择 PostgreSQL？
- **事务** — 原子会话状态更新防止竞争条件。
- **全文搜索** — 按内容查询会话历史。
- **JSON 列** — 存储灵活的代理元数据和运行历史。
- **内置 pub/sub** — LISTEN/NOTIFY 用于异步工作流。
- **大规模验证** — 在多租户 SaaS 中久经考验。

#### 模式概览

```
sessions 表：
  ├─ id (uuid，主键)
  ├─ user_id (uuid，租户隔离)
  ├─ agent_id (string，哪个代理运行)
  ├─ conversation_history (jsonb，消息数组)
  ├─ metadata (jsonb，自定义代理状态)
  ├─ token_usage (jsonb，{prompt, completion, reasoning})
  ├─ created_at (timestamp)
  ├─ updated_at (timestamp)
  ├─ last_accessed_at (timestamp)
  └─ 索引：(user_id, agent_id)、(user_id, updated_at)、
           conversation_history 上的 GIN、metadata 上的 GIN

evaluations 表：
  ├─ id (uuid)
  ├─ session_id (uuid，fk → sessions)
  ├─ eval_type (string：accuracy、performance、reliability)
  ├─ score (float，0.0..1.0)
  ├─ details (jsonb，评估器输出)
  ├─ created_at (timestamp)
  └─ 索引：(session_id)、(eval_type, created_at)
```

> **TODO(post-S1)**：Sprint 1 完成后，链接到 `pkg/agentgo/storage/postgres/schema.sql` 中的最终 PostgreSQL 模式迁移。

#### 连接池调优

生产推荐配置：

```go
// 伪代码；参见 pkg/agentgo/storage/postgres 获取最新 API
type PostgresConfig struct {
    // 池中最大并发连接数
    MaxOpenConns int           // 默认：25，设置为 (副本数 * 4)

    // 保持打开的空闲连接
    MaxIdleConns int           // 默认：5，设置为 MaxOpenConns / 5

    // 空闲连接关闭前的时间
    ConnMaxIdleTime time.Duration  // 默认：5m

    // 连接重用前的最大生命周期
    ConnMaxLifetime time.Duration  // 默认：15m

    // 从池中获取连接的超时时间
    DialTimeout time.Duration  // 默认：5s
}
```

**如何调整大小：**
- **MaxOpenConns** = (代理-go 副本数) × 4。例如：3 个副本 × 4 = 12 个连接。
- **MaxIdleConns** = MaxOpenConns / 5。
- 监控 pg_stat_activity 确保不会达到 `max_connections`（默认 100）。

#### 迁移工作流程

在首次生产运行之前：

```bash
# 1. 定义或更新模式
# （Sprint 1 迁移 API 完成后，使用类似：）
export DATABASE_URL="postgres://user:pass@localhost:5432/agentgo_prod"

# 2. 运行迁移（CLI 或 SDK）
agentgo migrate up

# 3. 验证模式
psql $DATABASE_URL -c "\dt"

# 4. 创建初始索引
# （包含在迁移中，但验证它们存在）
psql $DATABASE_URL -c "SELECT * FROM pg_indexes WHERE schemaname='public';"
```

> **TODO(post-S1)**：实现 `agentgo migrate` 后，替换为实际 CLI 命令。

#### 多租户隔离

对于具有多个组织的 SaaS 部署：

```sql
-- 向 sessions 添加 tenant_id 列
ALTER TABLE sessions ADD COLUMN tenant_id UUID NOT NULL;

-- 为快速查找创建索引
CREATE INDEX idx_sessions_tenant_user
  ON sessions(tenant_id, user_id);

-- 为 RLS（行级安全）策略添加 tenant_id
CREATE POLICY rls_sessions_tenant ON sessions
  USING (tenant_id = current_user_id);
```

然后在应用代码中：

```go
// 始终在查询中按 tenant_id 过滤
// （参见 pkg/agentgo/storage 获取最新 API）
query := `
  SELECT * FROM sessions
  WHERE tenant_id = $1 AND user_id = $2
`
```

#### 备份和恢复

**备份策略：**
```bash
# 每日完整备份（UTC 02:00）（cron 作业）
0 2 * * * pg_dump -Fc $DATABASE_URL > /backups/agentgo-$(date +\%Y\%m\%d).dump

# 备份大小：~5 GB 每百万会话（估计）
# 保留 30 天的备份
find /backups -mtime +30 -delete
```

**时间点恢复（PITR）：**
```bash
# 恢复到特定时间戳
pg_restore -d agentgo_prod /backups/agentgo-20260407.dump

# 或使用 PostgreSQL WAL 归档（AWS RDS 自动执行）
# https://www.postgresql.org/docs/current/continuous-archiving.html
```

**保留策略：**
- 本地保留 7 天的每日转储。
- 每 7 天存档到 S3/GCS 保留 1 年。
- 合规性：检查您的数据驻留和保留要求。

---

## 可观测性

### 分布式追踪（OTLP）

**为什么重要：**
代理执行涉及多个步骤（输入解析、工具调用、LLM 调用、结果聚合）。追踪让您能够：
- 关联跨服务边界的请求。
- 识别瓶颈（LLM 慢、工具慢、网络延迟）。
- 调试多代理团队和工作流。
- 为成本控制采样追踪（并非所有请求都需要完整检测）。

#### 配置

> **TODO(post-S1)**：Sprint 1 完成后，链接到 `pkg/agentgo/observability/otlp.go` 中的最终 OTLP 配置结构。

最小设置：

```bash
# OTLP 导出器的环境变量
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"  # gRPC 端点
export OTEL_SERVICE_NAME="agentgo-prod"
export OTEL_DEPLOYMENT_ENVIRONMENT="production"

# 可选：采样（头部采样：要追踪的请求比率）
export OTEL_TRACES_SAMPLER="parentbased_traceidratio"
export OTEL_TRACES_SAMPLER_ARG="0.1"  # 追踪 10% 的请求
```

#### 跨度属性

框架发出以下跨度属性：

| 属性 | 例子 | 类型 | 备注 |
|-----------|---------|------|-------|
| `agent.name` | "DataProcessor" | string | 来自配置的代理名称 |
| `agent.model` | "gpt-4o" | string | 模型 ID |
| `agent.tools` | ["calculator", "http"] | array | 工具名称 |
| `tool.name` | "calculator" | string | 正在执行的工具 |
| `tool.duration_ms` | 145 | number | 时间（毫秒） |
| `tokens.prompt` | 500 | number | 消耗的提示令牌 |
| `tokens.completion` | 150 | number | 完成令牌 |
| `tokens.reasoning` | 50 | number | （o1/o3 模型）推理令牌 |
| `span.kind` | "INTERNAL" / "CLIENT" | string | OpenTelemetry 标准 |
| `error` | "max_loops_exceeded" | string | 错误代码（如果失败） |

#### 示例：Jaeger + 收集器 Compose

```yaml
# docker-compose.yml 摘录
version: '3.8'
services:
  # Jaeger all-in-one 用于本地开发/测试
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "6831:6831/udp"    # Jaeger agent (thrift compact)
      - "16686:16686"      # Web UI: http://localhost:16686
    environment:
      COLLECTOR_OTLP_ENABLED: "true"

  # OpenTelemetry 收集器（生产模式）
  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports:
      - "4317:4317"        # gRPC 接收器
      - "4318:4318"        # HTTP 接收器
    volumes:
      - ./otel-config.yaml:/etc/otel-collector-config.yaml
    command: ["--config=/etc/otel-collector-config.yaml"]
    depends_on:
      - jaeger

  # AgentOS 服务
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

**生产 otel-config.yaml：**

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
  # 对于生产，添加 Tempo、Honeycomb 或其他后端
  # otlp:
  #   endpoint: "tempo:4317"

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [jaeger]
```

#### 采样策略

**头部采样**（在源处采样）：
- 最适合：高容量、探索性监控。
- 配置：`OTEL_TRACES_SAMPLER_ARG=0.01`（1% 的请求）。
- 成本：低，但您可能会遗漏罕见错误。

**尾部采样**（看到完整追踪后采样）：
- 最适合：错误调试、高延迟检测。
- 需要收集器的尾部采样处理器。
- 配置：将所有追踪保留在缓冲区中，按结果采样（错误始终，>5s 延迟始终，其他 10%）。

**自适应采样**：
- 根据错误率或负载调整比率。
- 从 10% 头部采样开始；根据成本和 SLO 调整。

#### 资源属性

始终包括：

```go
resource.Attributes{
    "service.name":           "agentgo-prod",
    "service.version":        "1.5.0",
    "deployment.environment": "production",
    "deployment.region":      "us-east-1",
    "container.id":           os.Getenv("HOSTNAME"),  // k8s 中的 pod 名称
}
```

### 指标

**测量内容：**

| 指标 | 类型 | 目的 |
|--------|------|---------|
| `agent_run_duration_seconds` | Histogram | 每个代理的 P50、P95、P99 延迟 |
| `agent_run_total` | Counter | 吞吐量（运行数/分钟） |
| `agent_errors_total` | Counter | 按类型的错误（max_loops、tool_error 等） |
| `tokens_used` | Counter | 提示/完成/推理令牌 |
| `tool_call_duration_seconds` | Histogram | 每个工具的时间（计算器 vs. http） |
| `session_active_count` | Gauge | 任何时间的活跃会话 |

> **TODO(post-S1)**：Prometheus 检测完成后，参见 `pkg/agentgo/observability/metrics.go` 获取官方导出器。

**Prometheus 爬取配置：**

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

### 日志

通过 `log/slog`（Go 1.21+）使用结构化日志：

```bash
# 启用 JSON 输出（机器可读）
export AGENT_LOG_FORMAT="json"
export AGENT_LOG_LEVEL="info"

# 将 stdout 转发到 ELK、Splunk 或 Datadog
# （通常通过容器编排器或日志代理）
```

**日志示例：**

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

**PII 抹除：**
- 不要记录原始 API 密钥、用户电子邮件或信用卡。
- 对敏感字段使用 `log.Redacted()`。
- 示例：如果输入包含 PII 模式，则 `"user_input": "[REDACTED]"`。

---

## 内存策略

代理需要记住会话历史。选择正确的后端：

### 内存后端对比

| 后端 | 用例 | 优点 | 缺点 |
|---------|----------|------|------|
| **InMemory** | 单进程、开发/测试 | 快速、无设置 | 重启时丢失、无持久化 |
| **HybridMemory** | 生产（带总结） | 平衡成本和上下文 | 总结质量变化 |
| **SummarizingMemory** | 长会话（>10k 令牌） | 便宜、扩展至 100k+ 消息 | 需要总结延迟 |

> **TODO(post-S1)**：Sprint 1 完成后，链接到 `pkg/agentgo/memory/` 中的内存实现。

### 选择正确的后端

**对于聊天机器人（短、交互式）：**
```go
// 使用混合：在完整中保留最近消息，旧消息总结
memory.NewHybridMemory(&memory.HybridConfig{
    MaxRecentMessages: 20,        // 完整的最后 20 条消息
    SummarizationThreshold: 100,  // 如果 >100 条消息，则总结
    Model: openaiModel,           // 总结便宜模型
})
```

**对于长期运行的代理（研究、数据处理）：**
```go
// 使用总结：所有内容都被压缩
memory.NewSummarizingMemory(&memory.SummarizingConfig{
    ContextWindowSize: 4000,  // 保留 ~4k 令牌的上下文
    SummaryModel: groqModel,  // 快速和便宜（Groq，不是 GPT-4）
    CacheSummaries: true,     // 不要重新总结相同批次
})
```

**对于内部系统（日志、防护栏）：**
```go
// 使用内存：没有 DB 延迟，清晰的截断策略
memory.NewMemory(&memory.Config{
    MessageLimit: 100,  // 当 >100 时删除最旧的
})
```

### 总结调优

如果使用 `HybridMemory` 或 `SummarizingMemory`：

```bash
# 环境变量
export AGENT_MEMORY_SUMMARIZE_AT=150  # 在 150 条消息时触发
export AGENT_MEMORY_PRESERVE_LAST=30  # 始终在完整中保留最后 30 条
export AGENT_MEMORY_SUMMARY_MODEL="gpt-4o-mini"  # 便宜 + 快速
```

**总结提示设计：**
```
您是一个会话总结器。对迄今为止的会话进行简洁的 1-2 段落总结，
保留关键事实、决定和上下文，以便代理能够有效地继续。

会话：
{history}

总结：
```

**评估质量：**
- 运行评估：总结后代理是否仍能准确回答？
- 比较令牌成本：总结延迟 + 成本 vs. 完整上下文成本。
- 如果总结损失 >10% 的准确性，使用更短的总结阈值或更长的 `PreserveLast`。

---

## 质量控制（评估框架）

在部署新代理版本之前，运行评估以捕捉回归。

### CI 中的评估

**工作流：**编写评估套件 → 在 CI 中运行 → 通过通过率控制部署。

示例 GitHub Actions 工作流（25 行）：

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
          # 解析 JUnit XML；如果 pass_rate < 0.9 则失败
          python3 scripts/check_eval_threshold.py results.xml 0.9

      - name: Upload Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: eval-results
          path: results.xml
```

### 评估器选择

使用哪个评估器？

| 评估器 | 检查 | 成本 | 速度 | 例子 |
|-----------|--------|------|------|---------|
| **Accuracy** | 答案正确性（精确匹配、语义相似性） | 中 | 快 | "最终答案是否正确？" |
| **Performance** | 延迟、令牌使用 | 低 | 即时 | "是否在 <2s 内完成？使用 <500 令牌？" |
| **Reliability** | 错误处理、边界情况 | 高 | 慢 | "是否在坏输入时优雅失败？" |
| **Judge**（LLM-as-judge） | 整体质量、推理 | 高 | 慢 | "响应是否有帮助且无害？" |

**何时使用每个：**
- **Accuracy**：始终（在此基础上进行部署控制）。
- **Performance**：对于成本敏感的应用；令牌尖峰时警报。
- **Reliability**：主要模型更改之前。
- **Judge**：主观质量；对于内容生成、推理任务有用。

---

## 通过环境变量进行配置

生产的完整参考：

| 变量 | 默认 | 类型 | 必需 | 描述 |
|----------|---------|------|----------|-------------|
| `DATABASE_URL` | (无) | string | 是 | PostgreSQL 连接：`postgres://user:pass@host:5432/db` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4317` | string | 否 | OpenTelemetry 收集器 gRPC 端点 |
| `OTEL_SERVICE_NAME` | `agentgo` | string | 否 | 追踪/指标中的服务名称 |
| `OPENAI_API_KEY` | (无) | string | 如使用 OpenAI | OpenAI API 密钥 |
| `ANTHROPIC_API_KEY` | (无) | string | 如使用 Claude | Anthropic API 密钥 |
| `GROQ_API_KEY` | (无) | string | 如使用 Groq | Groq API 密钥 |
| `GEMINI_API_KEY` | (无) | string | 如使用 Gemini | Google Gemini API 密钥 |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | string | 如使用 Ollama | Ollama 服务器 URL |
| `AGENT_LOG_LEVEL` | `info` | string | 否 | `debug`、`info`、`warn`、`error` |
| `AGENT_LOG_FORMAT` | `text` | string | 否 | `text` 或 `json` |
| `AGENT_MAX_LOOPS` | `10` | int | 否 | 每个 agent.Run() 的最大工具调用迭代数 |
| `AGENT_MEMORY_TYPE` | `hybrid` | string | 否 | `inmemory`、`hybrid`、`summarizing` |
| `AGENT_MEMORY_SUMMARIZE_AT` | `150` | int | 否 | 在 N 条消息时触发总结 |
| `VECTOR_DB_TYPE` | (无) | string | 否 | `chromadb`、`pinecone`，或忽略无 RAG |
| `CHROMADB_ENDPOINT` | `http://localhost:8000` | string | 如使用 ChromaDB | ChromaDB 服务端点 |
| `SERVER_PORT` | `8080` | int | 否 | HTTP 服务器端口 |
| `SERVER_SHUTDOWN_TIMEOUT` | `30s` | duration | 否 | 优雅关闭超时 |

---

## 密钥管理

**不要做：**
```bash
# 坏：硬编码密钥
export OPENAI_API_KEY="sk-..."
git commit .env
```

**要做：**
```bash
# 好：外部化密钥
# AWS Secrets Manager
export OPENAI_API_KEY=$(aws secretsmanager get-secret-value --secret-id agentgo/prod/openai | jq -r .SecretString)

# 或：在运行时注入
# Kubernetes secret 挂载为环境变量：
# spec.containers[0].env:
#   - name: OPENAI_API_KEY
#     valueFrom:
#       secretKeyRef:
#         name: agentgo-secrets
#         key: openai-api-key
```

**轮换：**
- 每 90 天轮换密钥。
- 规划轮换窗口（低流量时间）；预期短暂 API 错误。
- 在预演环境中首先测试轮换。

**作用域：**
- API 密钥：尽可能只读（例如，仅 GPT-4 视觉，不微调）。
- 数据库：为代理提供单独用户（sessions 上的 SELECT、INSERT、UPDATE；无 DROP）。
- 服务账户：每个部署环境（dev、staging、prod）一个。

---

## Docker Compose 示例

完整、生产就绪的 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  # PostgreSQL（有状态后端）
  postgres:
    image: postgres:16-alpine
    container_name: agentgo-postgres
    environment:
      POSTGRES_USER: agentgo
      POSTGRES_PASSWORD: changeme123  # 在 prod 中使用密钥
      POSTGRES_DB: agentgo_prod
      POSTGRES_INITDB_ARGS: "-c shared_preload_libraries=pg_stat_statements"
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
      # 如需要，挂载迁移脚本
      # - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U agentgo"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - agentgo-net

  # OpenTelemetry 收集器
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

  # Jaeger 用于本地追踪（在 prod 中替换为 Tempo/Honeycomb）
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

  # AgentOS REST API（无状态，可水平扩展）
  agentos:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: agentgo-api
    ports:
      - "8080:8080"
    environment:
      # 数据库
      DATABASE_URL: "postgres://agentgo:changeme123@postgres:5432/agentgo_prod"

      # 可观测性
      OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
      OTEL_SERVICE_NAME: "agentgo-prod"
      AGENT_LOG_LEVEL: "info"
      AGENT_LOG_FORMAT: "json"

      # 代理配置
      AGENT_MAX_LOOPS: "10"
      AGENT_MEMORY_TYPE: "hybrid"

      # LLM API 密钥（在 prod 中从密钥管理器加载）
      OPENAI_API_KEY: "${OPENAI_API_KEY}"
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"

      # 服务器
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

  # （可选）ChromaDB 用于 RAG
  chromadb:
    image: chromadb/chroma:latest
    container_name: agentgo-chromadb
    ports:
      - "8000:8000"
    networks:
      - agentgo-net
    # 取消注释以启用：
    # profiles:
    #   - with-vectordb

networks:
  agentgo-net:
    driver: bridge

volumes:
  postgres-data:
    driver: local
```

**使用方法：**

```bash
# 启动所有服务
docker-compose up -d

# 验证
curl http://localhost:8080/health

# 查看日志
docker-compose logs -f agentos

# 扩展 API 副本（用于负载测试）
docker-compose up -d --scale agentos=3

# 关闭
docker-compose down
```

---

## 扩展考虑

### 水平扩展

**代理运行时无状态** — 在负载均衡器后面部署 N 个副本：

```yaml
# Kubernetes 示例
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentos
spec:
  replicas: 3  # 从 3 开始；根据 CPU/内存 自动扩展
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

**PostgreSQL 是扩展瓶颈：**
- 连接池：每个副本最多 25 个连接（根据负载调整）。
- PostgreSQL 中的最大连接：`max_connections`（默认 100）。
- 对于 4 个副本 × 25 conns = 100，您处于极限。
- 解决方案：在 PostgreSQL 前使用 PgBouncer（连接池程序）。

### PgBouncer 用于连接池

```ini
# pgbouncer.ini
[databases]
agentgo_prod = host=postgres port=5432 dbname=agentgo_prod

[pgbouncer]
pool_mode = transaction         # 最便宜；每个事务一个连接
max_client_conn = 1000         # 来自应用层的客户端
default_pool_size = 25         # 每个数据库的连接池
reserve_pool_size = 5
reserve_pool_timeout = 3

[users]
agentgo = "password"
```

然后：
```bash
# 将 DATABASE_URL 指向 PgBouncer（端口 6432）
export DATABASE_URL="postgres://agentgo:password@pgbouncer:6432/agentgo_prod"
```

### 速率限制

上游 LLM API 有速率限制。实施反压：

```go
// 伪代码
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(
    rate.Limit(100),  // 100 请求/秒 到 LLM
    1,                // 突发 1
)

if !limiter.Allow() {
    return fmt.Errorf("rate limit exceeded; retry after 1s")
}
```

**成本控制：**
- 按会话/用户跟踪令牌使用。
- 设置会话预算：`MAX_TOKENS_PER_SESSION=100000`。
- 如果超出，拒绝；向用户收费并警报操作人员。

---

## 常见陷阱

1. **忘记在 TracerProvider 上调用 shutdown**
   - 缓冲跨度在进程退出时丢失。始终在 defer 中调用 `tp.Shutdown(ctx)`。

2. **连接池大小 > PostgreSQL max_connections**
   - 结果："太多连接"错误。检查 postgres 中的 `SHOW max_connections;`。
   - 修复：要么增加 `max_connections`，要么减少每个副本的池大小。

3. **应用程序处于活动状态时运行迁移**
   - 风险：代码和 DB 之间的模式不匹配。在低流量窗口安排，先备份。

4. **在测试中模拟 LLM 但不断言提示内容**
   - 风险：生产中的提示注入、PII 泄露。测试提示内容，不仅仅是响应。

5. **进行重大配置更改前不备份**
   - 风险：如果需要回滚，数据丢失。总是在 ALTER TABLE 前备份。

6. **存储 API 密钥在环境变量中而不轮换**
   - 风险：密钥泄露无法检测。使用密钥管理器并轮换。

7. **总结模型与生产代理模型不同**
   - 风险：总结失败或质量下降（如果模型不可用）。测试故障转移。

8. **不监控令牌使用**
   - 风险：意外账单。设置指标和每小时令牌警报。

9. **在生产中以 100% 进行跟踪采样**
   - 风险：巨大的成本和存储。从 10% 开始，根据容量调整。

10. **忽略缓慢的数据库查询**
    - 风险：在峰值负载下延迟尖峰。在缓慢查询上使用 `EXPLAIN ANALYZE`；添加索引。

---

## 飞行前清单

在推向生产之前：

- [ ] **数据库**
  - [ ] 提供 PostgreSQL 实例，保留 30 天的备份
  - [ ] 模式迁移在本地测试并通过
  - [ ] 连接池已调优（MaxOpenConns = 副本 × 4）
  - [ ] 如果 >4 个副本，PgBouncer（或等效项）在 DB 前面

- [ ] **可观测性**
  - [ ] OpenTelemetry 收集器运行并接收追踪
  - [ ] Jaeger/Tempo/Honeycomb 后端可访问且健康
  - [ ] 采样比率设置（生产 10%，根据需要调整）
  - [ ] Prometheus 爬取端点已测试（`/metrics`）
  - [ ] 日志聚合管道运行（ELK、Splunk 等）
  - [ ] PII 抹除规则已配置并测试

- [ ] **配置**
  - [ ] 所有环境变量已设置（DATABASE_URL、OTEL_*、LLM_*_API_KEY）
  - [ ] 密钥管理器（AWS/GCP/Vault）已提供；密钥已轮换
  - [ ] 代码或 Docker 镜像中没有硬编码密钥
  - [ ] SERVER_SHUTDOWN_TIMEOUT 设置为优雅窗口（30s）

- [ ] **内存和上下文**
  - [ ] 内存后端已在负载下测试（InMemory/Hybrid/Summarizing）
  - [ ] 总结模型（如使用）快速且便宜（Groq、GPT-4o-mini）
  - [ ] 摘要质量通过评估套件验证
  - [ ] 最大循环计数适合工作负载（AGENT_MAX_LOOPS）

- [ ] **评估框架**
  - [ ] 评估套件已编写并在本地通过（>90% 通过率）
  - [ ] CI 工作流配置为在 PR 上运行评估
  - [ ] 质量门槛值已定义并强制执行（pass_rate >= 0.9）
  - [ ] 评估失败时的故障转移计划（手动审查、金丝雀推出）

- [ ] **负载和安全**
  - [ ] 负载测试：100 RPS 5 分钟；P99 延迟 <2s
  - [ ] LLM 速率限制已测试，故障转移有效
  - [ ] 成本限制已强制执行（每个会话的令牌预算）
  - [ ] SQL 注入防护（使用参数化查询）
  - [ ] API 身份验证已配置（OAuth、JWT 或内部服务账户）

- [ ] **部署**
  - [ ] Docker 镜像已构建并扫描漏洞
  - [ ] Kubernetes 清单（或 Docker Compose）已审查
  - [ ] 健康检查正确响应（HTTP 200 on /health）
  - [ ] 已测试优雅关闭（SIGTERM → 关闭 DB → 退出）
  - [ ] 滚动更新策略已定义（最大波动、最大不可用）

- [ ] **监控和警报**
  - [ ] 警报规则已配置（5xx 错误率、延迟、DB 连接）
  - [ ] 待命轮换和运行手册就位
  - [ ] 事件响应过程已记录
  - [ ] 事后反思模板可用

- [ ] **文档**
  - [ ] 常见问题运行手册（DB 已满、OOM、速率限制）
  - [ ] 灾难恢复计划（从备份恢复、提升副本）
  - [ ] 团队已培训以部署和故障排除

---

## 后续步骤

1. **阅读相关文档：**
   - [`/advanced/architecture`](./architecture.md) — 系统设计
   - [`/advanced/observability`](./observability.md) — SSE 流和 Logfire
   - [`/advanced/testing`](./testing.md) — 质量控制和评估框架
   - [`/guide/memory`](../guide/memory.md) — 内存后端深入讨论

2. **在本地设置：**
   ```bash
   docker-compose up -d
   curl http://localhost:8080/health
   curl -X POST http://localhost:8080/api/v1/agents/test/run \
     -H "Content-Type: application/json" \
     -d '{"input": "What is 2+2?"}'
   ```

3. **首先部署到预演环境：**
   - 测试完整部署流程（迁移、密钥、TLS）。
   - 运行评估；验证警报工作。
   - 负载测试；测量基线延迟和成本。

4. **规划生产推出：**
   - 金丝雀：10% 流量到新版本，监控 1 小时。
   - 蓝绿：并行运行 v1 和 v2，在截止时切换流量。
   - 回滚计划：DB 迁移回滚、前一个镜像版本。

---

**有问题？问题？**在 [github.com/jholhewres/agent-go](https://github.com/jholhewres/agent-go) 打开 GitHub issue 或加入讨论。
