# Knowledge Package - Document Loaders & Chunkers

Complete document loading and processing system for RAG (Retrieval-Augmented Generation) in AgentGo.

## Available Loaders

### 1. **TextLoader** - Text files
```go
loader := knowledge.NewTextLoader("./docs/readme.md")
docs, err := loader.Load()
```
**Supports**: `.txt`, `.md`, `.log`, any text file

---

### 2. **DirectoryLoader** - Complete directories
```go
loader := knowledge.NewDirectoryLoader(
    "./docs",
    "*.md",    // Pattern: *.txt, *.md, etc.
    true,      // Recursive
)
docs, err := loader.Load()
```
**Features**:
- Glob pattern support
- Recursive mode
- Extension filtering

---

### 3. **PDFLoader** - PDF documents
```go
loader := knowledge.NewPDFLoader("./paper.pdf")
docs, err := loader.Load()
```
**Features**:
- Text extraction from all pages
- Configurable page separator
- Metadata with page count
- Supports text-based PDFs (not OCR)

**Dependency**: `github.com/ledongthuc/pdf`

---

### 4. **CSVLoader** - CSV tables
```go
loader := knowledge.NewCSVLoader("./data.csv")
loader.HasHeader = true
loader.ContentFormat = "text" // or "json"
loader.RowsPerDoc = 0 // 0 = all rows in one doc
docs, err := loader.Load()
```
**Features**:
- Automatic header detection
- Multiple output formats (text/JSON)
- Row-based document splitting
- Column filtering
- Configurable delimiter

**Native Go**: Uses `encoding/csv`

---

### 5. **JSONLoader** - JSON documents
```go
loader := knowledge.NewJSONLoader("./data.json")
loader.ContentFields = []string{"title", "content"}
loader.MetadataFields = []string{"author", "date"}
docs, err := loader.Load()
```
**Features**:
- Supports objects and arrays
- Selective field extraction
- Customizable metadata
- Auto-detection of structure

**Supports**:
- Single JSON objects
- Arrays of objects
- Nested JSON

**Native Go**: Uses `encoding/json`

---

### 6. **HTMLLoader** - HTML pages
```go
loader := knowledge.NewHTMLLoader("./page.html")
loader.RemoveScripts = true
loader.RemoveStyles = true
loader.ExtractMetaTags = true
loader.Selectors = []string{"article", ".content"} // Optional
docs, err := loader.Load()
```
**Features**:
- Script/style removal
- Meta tag extraction
- Custom CSS selectors
- Link preservation (optional)
- Automatic whitespace cleanup

**Dependency**: `github.com/PuerkitoBio/goquery`

---

### 7. **URLLoader** - Web content
```go
loader := knowledge.NewURLLoader("https://example.com/article")
loader.Timeout = 30 * time.Second
loader.Headers = map[string]string{"Authorization": "Bearer token"}
docs, err := loader.Load()
```
**Features**:
- Auto-detection of content-type
- Supports HTML, JSON, PDF, text
- Customizable headers
- Configurable timeout
- Follow redirects

**Automatic routing**:
- HTML -> HTMLLoader
- JSON -> JSONLoader
- PDF -> PDFLoader
- Text -> TextLoader

---

### 8. **MultiURLLoader** - Multiple URLs
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
- Concurrent loading
- Rate limiting
- Shared metadata
- Individual error handling

---

### 9. **ReaderLoader** - Streams (io.Reader)
```go
loader := knowledge.NewReaderLoader(reader, "doc-id", metadata)
docs, err := loader.Load()
```
**Use cases**:
- HTTP response bodies
- Stdin
- In-memory buffers
- Pipes

---

## Chunkers (Document Splitting)

### 1. **CharacterChunker** - By characters
```go
chunker := knowledge.NewCharacterChunker(
    1000,  // ChunkSize (characters)
    100,   // ChunkOverlap
)
chunks, err := chunker.Chunk(document)
```
**Features**:
- Smart splitting on separators
- Overlap for context
- Preserves complete words
- Automatic metadata (start_char, end_char)

**Ideal for**: Text without clear structure

---

### 2. **SentenceChunker** - By sentences
```go
chunker := knowledge.NewSentenceChunker(
    1000,  // MaxChunkSize
    250,   // MinChunkSize
)
chunks, err := chunker.Chunk(document)
```
**Features**:
- Preserves complete sentences
- Automatic detection (`.`, `!`, `?`)
- Respects min/max limits
- Semantic integrity

**Ideal for**: Articles, narrative documents

---

### 3. **ParagraphChunker** - By paragraphs
```go
chunker := knowledge.NewParagraphChunker(2000) // MaxChunkSize
chunks, err := chunker.Chunk(document)
```
**Features**:
- Splits on `\n\n`
- Fallback to CharacterChunker (for large paragraphs)
- Maintains document structure

**Ideal for**: Documentation, books, structured articles

---

## Complete RAG Pipeline

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

    // 1. Load documents (multiple types)
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

    // 2. Chunk documents
    chunker := knowledge.NewCharacterChunker(1000, 100)
    var allChunks []knowledge.Chunk

    for _, doc := range allDocs {
        chunks, _ := chunker.Chunk(doc)
        allChunks = append(allChunks, chunks...)
    }

    // 3. Create embeddings
    embedder := openai.NewEmbedding("text-embedding-3-small", apiKey)

    // 4. Store in vector database
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
    queryEmbedding, _ := embedder.Embed(ctx, "How does the system work?")
    results, _ := vectorDB.Query(ctx, queryEmbedding, 5, nil)

    for _, result := range results {
        println(result.Content)
    }
}
```

---

## Data Structures

### Document
```go
type Document struct {
    ID       string                 // Unique identifier
    Content  string                 // Text content
    Metadata map[string]interface{} // Metadata (filename, path, etc.)
    Source   string                 // Origin (file path, URL)
}
```

### Chunk
```go
type Chunk struct {
    ID       string                 // Unique identifier
    Content  string                 // Chunk content
    Metadata map[string]interface{} // Inherited metadata + chunk info
    Index    int                    // Position in the original document
}
```

---

## Performance

| Loader | Speed | Memory Usage | Notes |
|--------|-------|--------------|-------|
| TextLoader | Very fast | Low | Direct file read |
| PDFLoader | Fast | Medium | Depends on PDF size |
| CSVLoader | Very fast | Low | Native Go parser |
| JSONLoader | Very fast | Low | Native Go parser |
| HTMLLoader | Fast | Medium | Parsing + cleanup |
| URLLoader | Medium | Medium | Depends on network |
| MultiURLLoader | Fast | Medium-High | Parallelization |

---

## Security

- **Path Traversal**: `filepath.Walk` does not follow symlinks
- **URL Validation**: Configurable timeout and headers
- **Memory Limits**: Chunkers prevent giant documents from filling memory
- **Error Handling**: All loaders return descriptive errors

---

## Dependencies

| Loader | Dependency | License |
|--------|------------|---------|
| PDFLoader | `github.com/ledongthuc/pdf` | Apache 2.0 |
| HTMLLoader | `github.com/PuerkitoBio/goquery` | BSD 3-Clause |
| CSV, JSON, Text | Native Go | BSD 3-Clause |

---

## Advanced Examples

### CSV with Column Filtering
```go
loader := knowledge.NewCSVLoader("./users.csv")
loader.TextColumns = []int{0, 2, 4} // Only columns 0, 2, 4
loader.ContentFormat = "json"
docs, _ := loader.Load()
```

### HTML with Specific Selectors
```go
loader := knowledge.NewHTMLLoader("./article.html")
loader.Selectors = []string{"article", ".post-content", "#main"}
loader.PreserveLinks = true
docs, _ := loader.Load()
```

### JSON with Custom Template
```go
loader := knowledge.NewJSONLoader("./posts.json")
loader.ContentFields = []string{"title", "body", "tags"}
loader.MetadataFields = []string{"author", "date", "category"}
docs, _ := loader.Load()
```

### URL with Custom Headers
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

## Best Practices

1. **Choose the Right Chunker**:
   - Technical documents -> `ParagraphChunker`
   - Articles/narratives -> `SentenceChunker`
   - Unstructured text -> `CharacterChunker`

2. **Adjust Chunk Size**:
   - Embeddings: 500-1000 characters
   - LLM Context: 1000-2000 characters
   - Overlap: 10-20% of chunk size

3. **Use Metadata**:
   - Filter by document type
   - Sort by date/relevance
   - Track source for citations

4. **Error Handling**:
   - Use `ContinueOnErr` for batch processing
   - Log failures for analysis
   - Validate documents before chunking

---

## TODO / Roadmap

- [ ] **OCR Support** - Text extraction from scanned PDFs
- [ ] **DocxLoader** - Microsoft Word documents
- [ ] **PPTXLoader** - PowerPoint presentations
- [ ] **ExcelLoader** - Excel spreadsheets
- [ ] **XMLLoader** - XML documents
- [ ] **EpubLoader** - E-books
- [ ] **AudioLoader** - Audio transcription
- [ ] **VideoLoader** - Video transcription
- [ ] **DatabaseLoader** - SQL/NoSQL queries
- [ ] **S3Loader** - AWS S3 objects
- [ ] **GCSLoader** - Google Cloud Storage
- [ ] **GitLoader** - Git repositories
- [ ] **SlackLoader** - Slack messages
- [ ] **NotionLoader** - Notion pages
- [ ] **ConfluenceLoader** - Wiki pages
- [ ] **JiraLoader** - Issues and documentation

---

## Contributing

To add a new loader:

1. Implement the `Loader` interface:
```go
type Loader interface {
    Load() ([]Document, error)
}
```

2. Follow the naming pattern: `*Loader`, `*ReaderLoader`
3. Add relevant metadata
4. Create unit tests
5. Update this README

---

**Author**: Jhol Hewres (@jholhewres)
**License**: Apache 2.0
**Version**: 1.0.0
