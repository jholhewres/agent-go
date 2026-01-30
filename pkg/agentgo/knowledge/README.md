# Knowledge Package - Document Loaders & Chunkers

Sistema completo de carregamento e processamento de documentos para RAG (Retrieval-Augmented Generation) no AgentGo.

## 📚 Loaders Disponíveis

### 1. **TextLoader** - Arquivos de texto
```go
loader := knowledge.NewTextLoader("./docs/readme.md")
docs, err := loader.Load()
```
**Suporta**: `.txt`, `.md`, `.log`, qualquer arquivo de texto

---

### 2. **DirectoryLoader** - Diretórios completos
```go
loader := knowledge.NewDirectoryLoader(
    "./docs",
    "*.md",    // Pattern: *.txt, *.md, etc.
    true,      // Recursive
)
docs, err := loader.Load()
```
**Features**:
- Suporte a glob patterns
- Modo recursivo
- Filtragem por extensão

---

### 3. **PDFLoader** - Documentos PDF
```go
loader := knowledge.NewPDFLoader("./paper.pdf")
docs, err := loader.Load()
```
**Features**:
- Extração de texto de todas as páginas
- Separador configurável entre páginas
- Metadata com número de páginas
- Suporta PDFs com texto (não OCR)

**Dependência**: `github.com/ledongthuc/pdf`

---

### 4. **CSVLoader** - Tabelas CSV
```go
loader := knowledge.NewCSVLoader("./data.csv")
loader.HasHeader = true
loader.ContentFormat = "text" // ou "json"
loader.RowsPerDoc = 0 // 0 = todas em um doc
docs, err := loader.Load()
```
**Features**:
- Detecção automática de headers
- Múltiplos formatos de saída (texto/JSON)
- Divisão por número de linhas
- Filtragem de colunas
- Delimiter configurável

**Nativo Go**: Usa `encoding/csv`

---

### 5. **JSONLoader** - Documentos JSON
```go
loader := knowledge.NewJSONLoader("./data.json")
loader.ContentFields = []string{"title", "content"}
loader.MetadataFields = []string{"author", "date"}
docs, err := loader.Load()
```
**Features**:
- Suporta objetos e arrays
- Extração seletiva de campos
- Metadata customizável
- Auto-detecção de estrutura

**Suporta**:
- Objetos JSON únicos
- Arrays de objetos
- JSON aninhado

**Nativo Go**: Usa `encoding/json`

---

### 6. **HTMLLoader** - Páginas HTML
```go
loader := knowledge.NewHTMLLoader("./page.html")
loader.RemoveScripts = true
loader.RemoveStyles = true
loader.ExtractMetaTags = true
loader.Selectors = []string{"article", ".content"} // Opcional
docs, err := loader.Load()
```
**Features**:
- Remoção de scripts/styles
- Extração de meta tags
- Seletores CSS customizados
- Preservação de links (opcional)
- Limpeza automática de whitespace

**Dependência**: `github.com/PuerkitoBio/goquery`

---

### 7. **URLLoader** - Conteúdo da Web
```go
loader := knowledge.NewURLLoader("https://example.com/article")
loader.Timeout = 30 * time.Second
loader.Headers = map[string]string{"Authorization": "Bearer token"}
docs, err := loader.Load()
```
**Features**:
- Auto-detecção de content-type
- Suporta HTML, JSON, PDF, texto
- Headers customizáveis
- Timeout configurável
- Follow redirects

**Roteamento automático**:
- HTML → HTMLLoader
- JSON → JSONLoader
- PDF → PDFLoader
- Texto → TextLoader

---

### 8. **MultiURLLoader** - Múltiplos URLs
```go
urls := []string{
    "https://example.com/page1",
    "https://example.com/page2",
    "https://example.com/page3",
}
loader := knowledge.NewMultiURLLoader(urls)
loader.MaxConcurrent = 5
loader.ContinueOnErr = true
docs, err := loader.Load()
```
**Features**:
- Carregamento concorrente
- Controle de taxa (rate limiting)
- Metadata compartilhada
- Tratamento de erros individual

---

### 9. **ReaderLoader** - Streams (io.Reader)
```go
loader := knowledge.NewReaderLoader(reader, "doc-id", metadata)
docs, err := loader.Load()
```
**Use cases**:
- HTTP response bodies
- Stdin
- Buffers em memória
- Pipes

---

## ✂️ Chunkers (Divisão de Documentos)

### 1. **CharacterChunker** - Por caracteres
```go
chunker := knowledge.NewCharacterChunker(
    1000,  // ChunkSize (caracteres)
    100,   // ChunkOverlap
)
chunks, err := chunker.Chunk(document)
```
**Features**:
- Quebra inteligente em separadores
- Overlap para contexto
- Preserva palavras completas
- Metadata automática (start_char, end_char)

**Ideal para**: Textos sem estrutura clara

---

### 2. **SentenceChunker** - Por sentenças
```go
chunker := knowledge.NewSentenceChunker(
    1000,  // MaxChunkSize
    250,   // MinChunkSize
)
chunks, err := chunker.Chunk(document)
```
**Features**:
- Preserva sentenças completas
- Detecção automática (`.`, `!`, `?`)
- Respeita limites min/max
- Integridade semântica

**Ideal para**: Artigos, documentos narrativos

---

### 3. **ParagraphChunker** - Por parágrafos
```go
chunker := knowledge.NewParagraphChunker(2000) // MaxChunkSize
chunks, err := chunker.Chunk(document)
```
**Features**:
- Quebra em `\n\n`
- Fallback para CharacterChunker (parágrafos grandes)
- Mantém estrutura do documento

**Ideal para**: Documentação, livros, artigos estruturados

---

## 🔄 Pipeline Completo RAG

```go
package main

import (
    "context"
    "github.com/jholhewres/agent-go/pkg/agentgo/knowledge"
    "github.com/jholhewres/agent-go/pkg/agentgo/embeddings/openai"
    "github.com/jholhewres/agent-go/pkg/agentgo/vectordb/pgvector"
)

func main() {
    ctx := context.Background()

    // 1. Carregar documentos (múltiplos tipos)
    var allDocs []knowledge.Document

    // PDFs
    pdfLoader := knowledge.NewPDFDirectoryLoader("./pdfs", true)
    pdfDocs, _ := pdfLoader.Load()
    allDocs = append(allDocs, pdfDocs...)

    // Markdown
    mdLoader := knowledge.NewDirectoryLoader("./docs", "*.md", true)
    mdDocs, _ := mdLoader.Load()
    allDocs = append(allDocs, mdDocs...)

    // URLs
    urlLoader := knowledge.NewMultiURLLoader([]string{
        "https://example.com/article1",
        "https://example.com/article2",
    })
    urlDocs, _ := urlLoader.Load()
    allDocs = append(allDocs, urlDocs...)

    // 2. Chunkar documentos
    chunker := knowledge.NewCharacterChunker(1000, 100)
    var allChunks []knowledge.Chunk

    for _, doc := range allDocs {
        chunks, _ := chunker.Chunk(doc)
        allChunks = append(allChunks, chunks...)
    }

    // 3. Criar embeddings
    embedder := openai.NewEmbedding("text-embedding-3-small", apiKey)

    // 4. Armazenar no vector database
    vectorDB := pgvector.New(connString, "knowledge_base")

    for _, chunk := range allChunks {
        embedding, _ := embedder.Embed(ctx, chunk.Content)
        
        vectorDB.Add(ctx, []vectordb.Document{{
            ID:        chunk.ID,
            Content:   chunk.Content,
            Embedding: embedding,
            Metadata:  chunk.Metadata,
        }})
    }

    // 5. Query
    queryEmbedding, _ := embedder.Embed(ctx, "Como funciona o sistema?")
    results, _ := vectorDB.Query(ctx, queryEmbedding, 5, nil)

    for _, result := range results {
        println(result.Content)
    }
}
```

---

## 📊 Estruturas de Dados

### Document
```go
type Document struct {
    ID       string                 // Identificador único
    Content  string                 // Conteúdo textual
    Metadata map[string]interface{} // Metadata (filename, path, etc.)
    Source   string                 // Origem (file path, URL)
}
```

### Chunk
```go
type Chunk struct {
    ID       string                 // Identificador único
    Content  string                 // Conteúdo do chunk
    Metadata map[string]interface{} // Metadata herdada + chunk info
    Index    int                    // Posição no documento original
}
```

---

## 🚀 Performance

| Loader | Velocidade | Uso de Memória | Notas |
|--------|------------|-----------------|-------|
| TextLoader | ⚡⚡⚡ Muito rápido | Baixo | Leitura direta de arquivo |
| PDFLoader | ⚡⚡ Rápido | Médio | Depende do tamanho do PDF |
| CSVLoader | ⚡⚡⚡ Muito rápido | Baixo | Parser nativo Go |
| JSONLoader | ⚡⚡⚡ Muito rápido | Baixo | Parser nativo Go |
| HTMLLoader | ⚡⚡ Rápido | Médio | Parsing + limpeza |
| URLLoader | ⚡ Médio | Médio | Depende da rede |
| MultiURLLoader | ⚡⚡ Rápido | Médio-Alto | Paralelização |

---

## 🔒 Segurança

- **Path Traversal**: `filepath.Walk` não segue symlinks
- **URL Validation**: Timeout e headers configuráveis
- **Memory Limits**: Chunkers previnem documentos gigantes em memória
- **Error Handling**: Todos os loaders retornam erros descritivos

---

## 📦 Dependências

| Loader | Dependência | Licença |
|--------|-------------|---------|
| PDFLoader | `github.com/ledongthuc/pdf` | Apache 2.0 |
| HTMLLoader | `github.com/PuerkitoBio/goquery` | BSD 3-Clause |
| CSV, JSON, Text | Nativo Go | BSD 3-Clause |

---

## 🛠️ Exemplos Avançados

### CSV com Filtragem de Colunas
```go
loader := knowledge.NewCSVLoader("./users.csv")
loader.TextColumns = []int{0, 2, 4} // Apenas colunas 0, 2, 4
loader.ContentFormat = "json"
docs, _ := loader.Load()
```

### HTML com Seletores Específicos
```go
loader := knowledge.NewHTMLLoader("./article.html")
loader.Selectors = []string{"article", ".post-content", "#main"}
loader.PreserveLinks = true
docs, _ := loader.Load()
```

### JSON com Template Customizado
```go
loader := knowledge.NewJSONLoader("./posts.json")
loader.ContentFields = []string{"title", "body", "tags"}
loader.MetadataFields = []string{"author", "date", "category"}
docs, _ := loader.Load()
```

### URL com Headers Customizados
```go
loader := knowledge.NewURLLoader("https://api.example.com/data")
loader.Headers = map[string]string{
    "Authorization": "Bearer " + token,
    "Accept": "application/json",
}
loader.Timeout = 60 * time.Second
docs, _ := loader.Load()
```

---

## 🎯 Best Practices

1. **Escolha o Chunker Certo**:
   - Documentos técnicos → `ParagraphChunker`
   - Artigos/narrativas → `SentenceChunker`
   - Texto sem estrutura → `CharacterChunker`

2. **Ajuste Chunk Size**:
   - Embeddings: 500-1000 caracteres
   - LLM Context: 1000-2000 caracteres
   - Overlap: 10-20% do chunk size

3. **Use Metadata**:
   - Filtre por tipo de documento
   - Ordene por data/relevância
   - Track source para citações

4. **Tratamento de Erros**:
   - Use `ContinueOnErr` para processamento em batch
   - Log failures para análise
   - Valide documentos antes de chunkar

---

## 📝 TODO / Roadmap

- [ ] **OCR Support** - Extração de texto de PDFs escaneados
- [ ] **DocxLoader** - Microsoft Word documents
- [ ] **PPTXLoader** - PowerPoint presentations
- [ ] **ExcelLoader** - Planilhas Excel
- [ ] **XMLLoader** - Documentos XML
- [ ] **EpubLoader** - E-books
- [ ] **AudioLoader** - Transcrição de áudio
- [ ] **VideoLoader** - Transcrição de vídeo
- [ ] **DatabaseLoader** - SQL/NoSQL queries
- [ ] **S3Loader** - AWS S3 objects
- [ ] **GCSLoader** - Google Cloud Storage
- [ ] **GitLoader** - Repositórios Git
- [ ] **SlackLoader** - Mensagens do Slack
- [ ] **NotionLoader** - Páginas do Notion
- [ ] **ConfluenceLoader** - Wiki pages
- [ ] **JiraLoader** - Issues e documentação

---

## 🤝 Contribuindo

Para adicionar um novo loader:

1. Implemente a interface `Loader`:
```go
type Loader interface {
    Load() ([]Document, error)
}
```

2. Siga o padrão de naming: `*Loader`, `*ReaderLoader`
3. Adicione metadata relevante
4. Crie testes unitários
5. Atualize este README

---

**Autor**: Jhol Hewres (@jholhewres)  
**Licença**: Apache 2.0  
**Versão**: 1.0.0
