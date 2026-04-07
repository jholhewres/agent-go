# Roadmap de Migração: Python Agno → Go

> **Projeto Base**: [agno-agi/agno](https://github.com/agno-agi/agno) (Python)  
> **Projeto Atual**: Agno-Go (Baseado no trabalho de rexleimo)  
> **Objetivo**: Migrar recursos faltantes do Agno Python para criar um framework completo em Go

---

## 📊 Status Geral

| Categoria | Status | Cobertura Estimada |
|-----------|--------|-------------------|
| **Core Agent** | ✅ Completo | 95% |
| **Models/Providers** | 🟡 Parcial | 60% |
| **Tools** | 🟡 Parcial | 40% |
| **Knowledge/RAG** | 🟡 Básico | 30% |
| **Learning** | ❌ Ausente | 0% |
| **Storage** | ✅ Completo | 90% |
| **Workflows** | ✅ Completo | 95% |
| **Teams** | ✅ Completo | 90% |

---

## 🚨 RECURSOS CRÍTICOS AUSENTES

### 1. **Learning System** (PRIORIDADE MÁXIMA)
O diferencial principal do Agno Python: agentes que aprendem e melhoram com o tempo.

**Componentes Ausentes:**
- [ ] `pkg/agentgo/learning/` - Sistema de aprendizado principal
- [ ] User Profiles (perfis persistentes entre sessões)
- [ ] User Memories (acumulação de memórias ao longo do tempo)
- [ ] Knowledge Learning (transferência de conhecimento entre usuários)
- [ ] Learning Modes (`always` ou `agentic`)
- [ ] Integração com `agent.Config` (flag `Learning bool`)

**Dependências:**
- Storage já existe (PostgreSQL, MongoDB, SQLite)
- Precisa de tabelas/collections adicionais para:
  - `user_profiles`
  - `user_memories`
  - `learned_knowledge`
  - `learning_events`

**Referência Python:**
```python
agent = Agent(
    model=OpenAIResponses(id="gpt-4"),
    db=SqliteDb(db_file="agents.db"),
    learning=True,  # ← Este recurso não existe no Go
)
```

---

## 🔧 TOOLS FALTANTES

### Tools Essenciais do Python Agno Ausentes:

#### **Data & Analytics**
- [ ] `duckdb` - Queries SQL em DataFrames e arquivos
- [ ] `polars` - Manipulação de dados (alternativa ao Pandas)
- [ ] `yfinance` - Dados financeiros do Yahoo Finance
- [ ] `shelltools` - Execução segura de comandos shell
- [ ] `sql` - Conexão e queries em bancos SQL
- [ ] `csv_tools` (avançado) - Já existe básico, falta análise completa

#### **APIs & Integrações**
- [ ] `discord` - Integração com Discord
- [ ] `discord_webhook` - Webhooks do Discord
- [ ] `github` - API do GitHub (issues, PRs, repos)
- [ ] `gitlab` - API do GitLab
- [ ] `linear` - Integração com Linear
- [ ] `notion` - API do Notion
- [ ] `slack` - Integração com Slack (webhooks, messages)
- [ ] `trello` - API do Trello
- [ ] `asana` - Integração com Asana
- [ ] `monday` - API do Monday.com
- [ ] `airtable` - Integração com Airtable
- [ ] `zendesk` - API do Zendesk
- [ ] `intercom` - Integração com Intercom
- [ ] `stripe` - API do Stripe
- [ ] `paypal` - Integração com PayPal
- [ ] `twilio` - SMS e chamadas via Twilio
- [ ] `sendgrid` - Envio de emails
- [ ] `mailchimp` - API do Mailchimp

#### **Pesquisa & Web**
- [ ] `exa` - Motor de busca avançado
- [ ] `serper` - Google Search API
- [ ] `serpapi` - SerpApi integration
- [ ] `duckduckgo` - Busca via DuckDuckGo
- [ ] `wikipedia` - API da Wikipedia
- [ ] `crawl4ai` - Web crawler avançado
- [ ] `firecrawl` - Extração de dados web
- [ ] `spider` - Web scraping
- [ ] `newspaper4k` - Extração de artigos de notícias
- [ ] `requests` (avançado) - HTTP client mais completo

#### **AI & ML**
- [ ] `huggingface` - Integração com Hugging Face
- [ ] `replicate` - API do Replicate
- [ ] `stability` - Stability AI (imagens)
- [ ] `resend` - Envio de emails AI-powered
- [ ] `cerebras` - Inference de modelos Cerebras
- [ ] `groq_tools` - Tools específicas do Groq (além do modelo)

#### **Áudio & Vídeo**
- [ ] `whisper` - Transcrição de áudio (OpenAI Whisper)
- [ ] `assemblyai` - Transcrição e análise de áudio
- [ ] `deepgram` - Speech-to-text
- [ ] `audio` - Manipulação de áudio genérica
- [ ] `video` - Manipulação de vídeo

#### **Documentos & Arquivos**
- [ ] `pdf` - Leitura e manipulação de PDFs (básico)
- [ ] `docx` - Documentos Word
- [ ] `excel` - Planilhas Excel
- [ ] `markdown` - Parser e gerador de Markdown
- [ ] `json_tools` - Manipulação avançada de JSON
- [ ] `xml` - Parser XML
- [ ] `yaml` - Parser YAML

#### **Cloud & Infraestrutura**
- [ ] `aws` - Integração com AWS (S3, Lambda, etc)
- [ ] `gcp` - Google Cloud Platform
- [ ] `azure` - Microsoft Azure
- [ ] `digitalocean` - API do DigitalOcean
- [ ] `heroku` - Integração com Heroku
- [ ] `vercel` - API do Vercel
- [ ] `netlify` - Integração com Netlify
- [ ] `cloudflare` - API do Cloudflare
- [ ] `terraform` - Gerenciamento de infraestrutura

#### **Desenvolvedor**
- [ ] `code_interpreter` - Execução de código Python/JS
- [ ] `git` - Operações Git (além do básico)
- [ ] `docker` - Controle de containers
- [ ] `kubernetes` - Integração com K8s
- [ ] `redis_tools` - Tools Redis (além do vector DB)
- [ ] `postgres_tools` - Tools PostgreSQL avançadas
- [ ] `mongodb_tools` - Tools MongoDB avançadas

#### **Outros**
- [ ] `browser` - Controle de navegador (Playwright/Selenium)
- [ ] `screenshot` - Captura de screenshots
- [ ] `ocr` - Reconhecimento de texto em imagens
- [ ] `barcode` - Leitura de códigos de barras/QR
- [ ] `weather` - Dados meteorológicos (além do OpenWeather)
- [ ] `maps` - Integração com Google Maps/Mapbox
- [ ] `calendar` - Google Calendar, Outlook, etc
- [ ] `email` - Cliente de email genérico
- [ ] `sms` - Envio de SMS genérico
- [ ] `crypto` - APIs de criptomoedas
- [ ] `blockchain` - Integração com blockchains

### Tools Já Implementados em Go:
- [x] `calculator`
- [x] `http` (básico)
- [x] `file`
- [x] `search` (básico)
- [x] `tavily`
- [x] `claude` (Claude Agent Skills)
- [x] `jira`
- [x] `gmail` (mark-as-read)
- [x] `googlesheets`
- [x] `elevenlabs` (speech)
- [x] `confluence`
- [x] `hackernews`
- [x] `pubmed`
- [x] `arxiv`
- [x] `bitbucket`
- [x] `youtube`
- [x] `csv` (básico)
- [x] `airflow`
- [x] `openweather`
- [x] `websearch`
- [x] `pandas` (parcial)

---

## 📚 KNOWLEDGE & RAG

### Recursos Ausentes:

#### **Vector Databases**
Agno Python suporta 20+ vector stores. Go tem apenas:
- [x] ChromaDB
- [x] RedisDB

**Faltam:**
- [ ] `pinecone` - Pinecone vector DB
- [ ] `weaviate` - Weaviate vector DB
- [ ] `qdrant` - Qdrant vector DB
- [ ] `milvus` - Milvus vector DB
- [ ] `pgvector` - PostgreSQL com pgvector
- [ ] `elasticsearch` - Elasticsearch
- [ ] `opensearch` - OpenSearch
- [ ] `faiss` - Facebook AI Similarity Search
- [ ] `lance` - LanceDB
- [ ] `vespa` - Vespa.ai
- [ ] `supabase` - Supabase Vector
- [ ] `azure_cognitive_search` - Azure Cognitive Search
- [ ] `mongodb_atlas` - MongoDB Atlas Vector Search
- [ ] `astra` - DataStax Astra
- [ ] `couchbase` - Couchbase Vector Search
- [ ] `neo4j` - Neo4j Vector Index
- [ ] `rockset` - Rockset
- [ ] `singlestore` - SingleStore

#### **Embedding Models**
Agno Python suporta múltiplos providers. Go tem:
- [x] OpenAI Embeddings
- [x] Ollama Embeddings (parcial)

**Faltam:**
- [ ] `cohere` - Cohere Embeddings
- [ ] `huggingface` - HuggingFace Embeddings
- [ ] `sentence_transformers` - Sentence Transformers locais
- [ ] `google` - Google/VertexAI Embeddings
- [ ] `azure` - Azure OpenAI Embeddings
- [ ] `mistral` - Mistral Embeddings
- [ ] `voyage` - Voyage AI Embeddings
- [ ] `jina` - Jina Embeddings

#### **Document Loaders**
- [ ] `pdf_loader` - Carregar PDFs para RAG
- [ ] `docx_loader` - Documentos Word
- [ ] `excel_loader` - Planilhas Excel
- [ ] `csv_loader` - Arquivos CSV
- [ ] `json_loader` - Arquivos JSON
- [ ] `xml_loader` - Arquivos XML
- [ ] `markdown_loader` - Arquivos Markdown
- [ ] `html_loader` - Páginas HTML
- [ ] `text_loader` - Arquivos de texto
- [ ] `url_loader` - Carregar de URLs
- [ ] `github_loader` - Repos GitHub
- [ ] `notion_loader` - Páginas Notion
- [ ] `confluence_loader` - Páginas Confluence
- [ ] `google_drive_loader` - Google Drive
- [ ] `dropbox_loader` - Dropbox
- [ ] `s3_loader` - AWS S3
- [ ] `youtube_loader` - Transcrições YouTube

#### **Chunking Strategies**
Go tem apenas chunking básico:
- [x] Character-based chunking
- [x] Token-based chunking

**Faltam:**
- [ ] `semantic_chunking` - Chunking semântico
- [ ] `recursive_chunking` - Chunking recursivo
- [ ] `paragraph_chunking` - Por parágrafos
- [ ] `sentence_chunking` - Por sentenças
- [ ] `markdown_header_chunking` - Por headers Markdown
- [ ] `code_chunking` - Específico para código

#### **Reranking**
- [ ] `cohere_rerank` - Cohere Reranker
- [ ] `cross_encoder` - Cross-encoder models
- [ ] `bm25` - BM25 reranking

#### **Hybrid Search**
- [ ] Combinação de semantic + keyword search
- [ ] Fusion de resultados (RRF - Reciprocal Rank Fusion)

---

## 🤖 MODELS & PROVIDERS

### Providers Ausentes:

#### **Major Providers**
- [ ] `azure_openai` - Azure OpenAI Service
- [ ] `aws_bedrock` - AWS Bedrock
- [ ] `vertex_ai` - Google Vertex AI
- [ ] `mistral` - Mistral AI
- [ ] `perplexity` - Perplexity AI
- [ ] `replicate` - Replicate
- [ ] `together` - Together AI (existe mas incompleto)
- [ ] `anyscale` - Anyscale Endpoints
- [ ] `fireworks` - Fireworks AI
- [ ] `modal` - Modal
- [ ] `runpod` - RunPod
- [ ] `cerebras` - Cerebras

#### **Specialized Providers**
- [ ] `stability` - Stability AI (imagens)
- [ ] `midjourney` - Midjourney (via API)
- [ ] `dalle` - DALL-E (OpenAI Images)
- [ ] `leonardo` - Leonardo.ai
- [ ] `ideogram` - Ideogram
- [ ] `flux` - Flux (Black Forest Labs)

#### **Local/Open Source**
- [ ] `llamacpp` - llama.cpp integration
- [ ] `vllm` - vLLM
- [ ] `textgen_webui` - oobabooga text-generation-webui
- [ ] `koboldai` - KoboldAI
- [ ] `localai` - LocalAI

### Recursos de Models Ausentes:
- [ ] **Function Calling Unificado** - Abstração comum para todos os providers
- [ ] **Prompt Caching** - Cache de prefixos de prompts (Anthropic, OpenAI)
- [ ] **Structured Outputs** - JSON Schema enforcement nativo
- [ ] **Vision** - Suporte consistente para imagens em todos os modelos
- [ ] **Audio Input** - Modelos que aceitam áudio
- [ ] **Video Input** - Modelos que aceitam vídeo
- [ ] **Tool Choice Control** - Forçar uso de ferramentas específicas
- [ ] **Response Format** - Controle fino do formato (JSON, YAML, etc)
- [ ] **Seed** - Reprodutibilidade com seeds
- [ ] **Top K/Top P** - Controle de sampling
- [ ] **Frequency/Presence Penalty** - Controle de repetição
- [ ] **Logit Bias** - Bias de tokens específicos
- [ ] **Logprobs** - Retornar log probabilities

---

## 🎨 PROMPT ENGINEERING

### Recursos Ausentes:
- [ ] **Prompt Templates** - Sistema de templates
- [ ] **Few-Shot Examples** - Gerenciamento de exemplos
- [ ] **Prompt Versioning** - Versionamento de prompts
- [ ] **Prompt Optimization** - Auto-otimização de prompts
- [ ] **Prompt Evaluation** - Avaliação de qualidade
- [ ] **Dynamic Prompts** - Prompts que se adaptam ao contexto

---

## 🔐 SECURITY & SAFETY

### Guardrails Existentes em Go:
- [x] Prompt Injection Guard (básico)
- [x] Custom Pre/Post Hooks
- [x] Media Validation

### Faltam:
- [ ] **PII Detection** - Detecção de informações pessoais
- [ ] **Toxic Content Filter** - Filtro de conteúdo tóxico
- [ ] **Bias Detection** - Detecção de viés
- [ ] **Output Validation** - Validação avançada de saídas
- [ ] **Rate Limiting** - Controle de taxa por usuário/sessão
- [ ] **Cost Controls** - Limites de custo por execução
- [ ] **Consent Management** - Gerenciamento de consentimentos
- [ ] **Audit Logging** - Logs detalhados para auditoria
- [ ] **Content Moderation** - Moderação de conteúdo
- [ ] **Sandboxing** - Execução isolada de código

---

## 📊 OBSERVABILITY & MONITORING

### Existente em Go:
- [x] SSE Event Stream
- [x] Logfire Integration (OpenTelemetry)
- [x] Reasoning Events

### Faltam:
- [ ] **LangSmith** - Integração com LangSmith
- [ ] **Weights & Biases** - W&B integration
- [ ] **Datadog** - Datadog APM
- [ ] **New Relic** - New Relic integration
- [ ] **Honeycomb** - Honeycomb observability
- [ ] **Grafana** - Dashboards Grafana
- [ ] **Prometheus** - Métricas Prometheus
- [ ] **Sentry** - Error tracking
- [ ] **Custom Metrics** - Sistema de métricas customizadas
- [ ] **Cost Tracking** - Rastreamento de custos por run/sessão/usuário
- [ ] **Performance Profiling** - Profiling detalhado
- [ ] **Token Usage Analytics** - Análise de uso de tokens

---

## 🧪 EVALUATION & TESTING

### Existente em Go:
- [x] `eval` package (básico)

### Faltam:
- [ ] **LLM-as-Judge** - Avaliação usando LLMs
- [ ] **Human Evaluation** - Interface para avaliação humana
- [ ] **Dataset Management** - Gerenciamento de datasets de teste
- [ ] **Regression Testing** - Testes de regressão automáticos
- [ ] **A/B Testing** - Framework de A/B testing
- [ ] **Benchmarking** - Benchmarks padronizados
- [ ] **Quality Metrics** - Métricas de qualidade (relevance, coherence, etc)
- [ ] **Cost Analysis** - Análise de custo/benefício
- [ ] **Latency Profiling** - Análise detalhada de latência
- [ ] **Prompt Testing** - Testing framework para prompts

---

## 🔄 ORCHESTRATION & WORKFLOWS

### Existente em Go:
- [x] Workflows (5 primitives: Step, Condition, Loop, Parallel, Router)
- [x] Teams (4 modes: Sequential, Parallel, Leader-Follower, Consensus)

### Faltam:
- [ ] **Conditional Routing** - Roteamento baseado em condições complexas
- [ ] **Error Handling Strategies** - Estratégias de retry, fallback, circuit breaker
- [ ] **Human-in-the-Loop** - Aprovações humanas (existe básico, falta avançado)
- [ ] **Timeout Management** - Gestão granular de timeouts
- [ ] **Resource Pooling** - Pool de recursos (agents, conexões)
- [ ] **Queue Management** - Filas de tarefas
- [ ] **Priority Scheduling** - Agendamento por prioridade
- [ ] **Background Tasks** - Tarefas em background
- [ ] **Cron Jobs** - Execução agendada
- [ ] **Event-Driven Workflows** - Workflows acionados por eventos

---

## 💾 STORAGE & STATE

### Existente em Go:
- [x] PostgreSQL
- [x] MongoDB
- [x] SQLite
- [x] SurrealDB (parcial)
- [x] Session Storage
- [x] Response Caching

### Faltam:
- [ ] **MySQL** - Suporte MySQL
- [ ] **MariaDB** - Suporte MariaDB
- [ ] **CockroachDB** - CockroachDB
- [ ] **TimescaleDB** - TimescaleDB (time-series)
- [ ] **DynamoDB** - AWS DynamoDB
- [ ] **Firestore** - Google Firestore
- [ ] **Cassandra** - Apache Cassandra
- [ ] **CosmosDB** - Azure CosmosDB
- [ ] **State Snapshots** - Snapshots completos de estado
- [ ] **State Migration** - Migração de estado entre versões
- [ ] **Distributed State** - Estado distribuído (Redis Cluster, etc)
- [ ] **State Compression** - Compressão de estado histórico

---

## 🌐 API & INTEGRATIONS

### Existente em Go:
- [x] AgentOS REST API (OpenAPI 3.0)
- [x] MCP (Model Context Protocol)
- [x] A2A (Agent-to-Agent)

### Faltam:
- [ ] **GraphQL API** - Alternativa GraphQL
- [ ] **gRPC API** - API gRPC
- [ ] **WebSocket API** - Real-time WebSocket
- [ ] **Webhook Management** - Sistema de webhooks
- [ ] **OAuth Integration** - Autenticação OAuth
- [ ] **API Key Management** - Gerenciamento de API keys
- [ ] **Rate Limiting** - Rate limiting por API
- [ ] **API Versioning** - Versionamento de API
- [ ] **SDK Generation** - Geração automática de SDKs
- [ ] **OpenAPI Extensions** - Extensões customizadas

---

## 🎯 AGENT PATTERNS

### Existente em Go:
- [x] Simple Agent
- [x] Agent with Tools
- [x] Multi-Agent Teams
- [x] Workflows

### Faltam:
- [ ] **ReAct Agent** - Padrão ReAct explícito
- [ ] **Plan-and-Execute** - Planejamento antes da execução
- [ ] **Self-Ask** - Self-asking pattern
- [ ] **Tree of Thoughts** - ToT pattern
- [ ] **Chain of Thought** - CoT pattern explícito
- [ ] **Reflexion** - Self-reflection pattern
- [ ] **Debate** - Multiple agents debating
- [ ] **Voting** - Consensus through voting
- [ ] **Hierarchical** - Hierarchical agent structures
- [ ] **Swarm** - Swarm intelligence patterns

---

## 📖 DOCUMENTATION & EXAMPLES

### Existente em Go:
- [x] VitePress documentation site
- [x] 15+ examples em `cmd/examples/`
- [x] API Reference

### Faltam:
- [ ] **Interactive Tutorials** - Tutoriais interativos
- [ ] **Video Tutorials** - Tutoriais em vídeo
- [ ] **Cookbook** - Cookbook completo (150+ recipes do Python)
- [ ] **Best Practices Guide** - Guia de melhores práticas
- [ ] **Architecture Patterns** - Padrões de arquitetura
- [ ] **Migration Guides** - Guias de migração (Python → Go)
- [ ] **Troubleshooting Guide** - Guia de troubleshooting
- [ ] **FAQ** - FAQ completo
- [ ] **Community Examples** - Repositório de exemplos da comunidade

---

## 🚀 DEPLOYMENT & OPERATIONS

### Existente em Go:
- [x] Docker support
- [x] Docker Compose
- [x] Health checks

### Faltam:
- [ ] **Kubernetes Manifests** - Manifests K8s completos
- [ ] **Helm Charts** - Helm charts
- [ ] **Terraform Modules** - Módulos Terraform
- [ ] **AWS Deployment** - Templates AWS (CDK, CloudFormation)
- [ ] **GCP Deployment** - Templates GCP
- [ ] **Azure Deployment** - Templates Azure
- [ ] **Serverless** - Deployment serverless (Lambda, Cloud Functions)
- [ ] **Edge Deployment** - Deployment em edge (Cloudflare Workers, etc)
- [ ] **Auto-Scaling** - Configuração de auto-scaling
- [ ] **Load Balancing** - Configuração de load balancing
- [ ] **Multi-Region** - Deployment multi-região
- [ ] **Disaster Recovery** - Planos de DR
- [ ] **Backup/Restore** - Estratégias de backup

---

## 📋 PRIORIZAÇÃO SUGERIDA

### 🔴 Fase 1: FUNDAÇÃO (8-12 semanas)
1. **Learning System** ⭐⭐⭐⭐⭐
   - User Profiles
   - User Memories
   - Knowledge Learning
   - Learning Modes
   
2. **Tools Essenciais** ⭐⭐⭐⭐
   - duckdb
   - github
   - slack
   - discord
   - notion
   - pdf loader
   - code_interpreter

3. **Vector Databases Core** ⭐⭐⭐⭐
   - Pinecone
   - Qdrant
   - Weaviate
   - pgvector

### 🟡 Fase 2: EXPANSÃO (12-16 semanas)
4. **RAG Avançado** ⭐⭐⭐
   - Document loaders (10+)
   - Hybrid search
   - Reranking (Cohere)
   - Semantic chunking

5. **Models & Providers** ⭐⭐⭐
   - Azure OpenAI
   - AWS Bedrock
   - Mistral
   - Function calling unificado
   - Structured outputs

6. **Security & Safety** ⭐⭐⭐
   - PII detection
   - Content moderation
   - Cost controls
   - Audit logging

### 🟢 Fase 3: MATURIDADE (16-24 semanas)
7. **Tools Avançadas** ⭐⭐
   - 50+ tools adicionais
   - Browser automation
   - Cloud integrations (AWS, GCP, Azure)
   - Developer tools avançadas

8. **Observability** ⭐⭐
   - LangSmith
   - Cost tracking
   - Performance profiling
   - Custom metrics

9. **Evaluation & Testing** ⭐⭐
   - LLM-as-Judge
   - A/B testing
   - Benchmarking suite

### 🔵 Fase 4: POLIMENTO (24+ semanas)
10. **Documentation** ⭐
    - Cookbook completo
    - Interactive tutorials
    - Video content
    - Migration guides

11. **Deployment** ⭐
    - K8s completo
    - Helm charts
    - Cloud templates
    - Serverless support

12. **Agent Patterns** ⭐
    - ReAct, ToT, CoT explícitos
    - Reflexion
    - Hierarchical structures

---

## 📦 DEPENDÊNCIAS EXTERNAS

### Bibliotecas Go Necessárias:
```go
// Learning & ML
- github.com/chewxy/math32
- github.com/gonum/gonum
- github.com/sjwhitworth/golearn

// Data Processing
- github.com/apache/arrow/go
- github.com/marcboeker/go-duckdb
- github.com/xitongsys/parquet-go

// Vector DBs
- github.com/pinecone-io/go-pinecone
- github.com/qdrant/go-client
- github.com/weaviate/weaviate-go-client

// Security
- github.com/google/go-github
- github.com/slack-go/slack
- github.com/bwmarrin/discordgo

// Observability
- go.opentelemetry.io/otel
- github.com/prometheus/client_golang
- github.com/DataDog/datadog-go
```

---

## 🎓 REFERÊNCIAS

- **Agno Python**: https://github.com/agno-agi/agno
- **Documentação Agno**: https://docs.agno.com
- **Agno-Go Original**: https://github.com/jholhewres/agent-go
- **Go Best Practices**: https://go.dev/doc/effective_go

---

## 📝 NOTAS

1. **Compatibilidade**: Manter compatibilidade de conceitos com o Python, mas adaptar para idiomática Go
2. **Performance**: Aproveitar goroutines e canais para operações paralelas
3. **Type Safety**: Usar o sistema de tipos do Go para prevenir erros
4. **Testing**: Manter >70% de cobertura em todos os pacotes novos
5. **Documentation**: Documentar tudo com GoDoc
6. **Examples**: Criar exemplo prático para cada recurso novo

---

## 🤝 CONTRIBUIÇÃO

Para contribuir com a migração:
1. Escolha um item da lista acima
2. Crie uma issue no repositório
3. Desenvolva seguindo os padrões do projeto
4. Adicione testes (>70% coverage)
5. Atualize documentação
6. Submeta PR com referência à issue

---

**Última atualização**: Janeiro 2026  
**Versão**: 1.0  
**Mantido por**: [Seu Nome]
