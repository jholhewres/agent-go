package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jholhewres/agent-go/pkg/agentgo/knowledge"
)

func main() {
	ctx := context.Background()

	fmt.Println("🚀 AgentGo Knowledge Loaders Demo")
	fmt.Println("===================================")

	// Demonstração de cada loader
	demoTextLoader()
	demoDirectoryLoader()
	demoPDFLoader()
	demoCSVLoader()
	demoJSONLoader()
	demoHTMLLoader()
	demoURLLoader()
	demoMultiURLLoader()
	demoReaderLoader()
	demoChunkers(ctx)

	fmt.Println("\n✅ Demo completo!")
	fmt.Println("📚 Consulte README.md para mais detalhes")
}

func demoTextLoader() {
	fmt.Println("📄 1. TextLoader - Arquivos de texto")
	fmt.Println("-------------------------------------")

	// Exemplo: carregar um arquivo de texto
	// loader := knowledge.NewTextLoader("./docs/readme.md")
	// docs, err := loader.Load()
	// if err != nil {
	// 	log.Printf("Erro: %v\n", err)
	// 	return
	// }

	fmt.Println("✓ Carrega arquivos .txt, .md, .log")
	fmt.Println("✓ Exemplo:")
	fmt.Println("  loader := knowledge.NewTextLoader(\"./readme.md\")")
	fmt.Println("  docs, err := loader.Load()")
	fmt.Println()
}

func demoDirectoryLoader() {
	fmt.Println("📁 2. DirectoryLoader - Diretórios completos")
	fmt.Println("-------------------------------------------")

	// Exemplo: carregar todos .md de um diretório recursivamente
	// loader := knowledge.NewDirectoryLoader("./docs", "*.md", true)
	// docs, err := loader.Load()

	fmt.Println("✓ Suporta glob patterns (*.txt, *.md)")
	fmt.Println("✓ Modo recursivo para subdiretórios")
	fmt.Println("✓ Exemplo:")
	fmt.Println("  loader := knowledge.NewDirectoryLoader(\"./docs\", \"*.md\", true)")
	fmt.Println("  docs, err := loader.Load()")
	fmt.Println()
}

func demoPDFLoader() {
	fmt.Println("📕 3. PDFLoader - Documentos PDF")
	fmt.Println("--------------------------------")

	// Exemplo: carregar um PDF
	// loader := knowledge.NewPDFLoader("./paper.pdf")
	// docs, err := loader.Load()

	fmt.Println("✓ Extração de texto de todas as páginas")
	fmt.Println("✓ Metadata com número de páginas")
	fmt.Println("✓ Exemplo:")
	fmt.Println("  loader := knowledge.NewPDFLoader(\"./paper.pdf\")")
	fmt.Println("  docs, err := loader.Load()")
	fmt.Println()
}

func demoCSVLoader() {
	fmt.Println("📊 4. CSVLoader - Tabelas CSV")
	fmt.Println("-----------------------------")

	// Exemplo: carregar um CSV
	// loader := knowledge.NewCSVLoader("./data.csv")
	// loader.HasHeader = true
	// loader.ContentFormat = "text" // ou "json"
	// docs, err := loader.Load()

	fmt.Println("✓ Formato texto ou JSON")
	fmt.Println("✓ Filtragem de colunas")
	fmt.Println("✓ Divisão por linhas")
	fmt.Println("✓ Exemplo:")
	fmt.Println("  loader := knowledge.NewCSVLoader(\"./data.csv\")")
	fmt.Println("  loader.ContentFormat = \"json\"")
	fmt.Println("  docs, err := loader.Load()")
	fmt.Println()
}

func demoJSONLoader() {
	fmt.Println("📋 5. JSONLoader - Dados JSON")
	fmt.Println("-----------------------------")

	// Exemplo: carregar um JSON
	// loader := knowledge.NewJSONLoader("./data.json")
	// loader.ContentFields = []string{"title", "content"}
	// loader.MetadataFields = []string{"author", "date"}
	// docs, err := loader.Load()

	fmt.Println("✓ Objetos e arrays")
	fmt.Println("✓ Extração seletiva de campos")
	fmt.Println("✓ Exemplo:")
	fmt.Println("  loader := knowledge.NewJSONLoader(\"./data.json\")")
	fmt.Println("  loader.ContentFields = []string{\"title\", \"body\"}")
	fmt.Println("  docs, err := loader.Load()")
	fmt.Println()
}

func demoHTMLLoader() {
	fmt.Println("🌐 6. HTMLLoader - Páginas HTML")
	fmt.Println("-------------------------------")

	// Exemplo: carregar HTML
	// loader := knowledge.NewHTMLLoader("./page.html")
	// loader.RemoveScripts = true
	// loader.Selectors = []string{"article", ".content"}
	// docs, err := loader.Load()

	fmt.Println("✓ Remove scripts/styles")
	fmt.Println("✓ Seletores CSS")
	fmt.Println("✓ Extração de meta tags")
	fmt.Println("✓ Exemplo:")
	fmt.Println("  loader := knowledge.NewHTMLLoader(\"./page.html\")")
	fmt.Println("  loader.Selectors = []string{\"article\"}")
	fmt.Println("  docs, err := loader.Load()")
	fmt.Println()
}

func demoURLLoader() {
	fmt.Println("🔗 7. URLLoader - Conteúdo web")
	fmt.Println("------------------------------")

	// Exemplo real: carregar de uma URL pública
	loader := knowledge.NewURLLoader("https://example.com")
	loader.Timeout = 10 // 10 segundos
	docs, err := loader.Load()

	if err != nil {
		fmt.Printf("⚠️  Erro ao carregar URL: %v\n", err)
	} else {
		fmt.Printf("✓ Carregado: %d documento(s)\n", len(docs))
		if len(docs) > 0 {
			fmt.Printf("  - ID: %s\n", docs[0].ID)
			fmt.Printf("  - Tamanho: %d caracteres\n", len(docs[0].Content))
			fmt.Printf("  - Tipo: %v\n", docs[0].Metadata["content_type"])
		}
	}

	fmt.Println("\n✓ Auto-detecção de content-type")
	fmt.Println("✓ Headers customizáveis")
	fmt.Println("✓ Exemplo:")
	fmt.Println("  loader := knowledge.NewURLLoader(\"https://example.com/article\")")
	fmt.Println("  docs, err := loader.Load()")
	fmt.Println()
}

func demoMultiURLLoader() {
	fmt.Println("🔗🔗 8. MultiURLLoader - Múltiplos URLs")
	fmt.Println("---------------------------------------")

	// Exemplo: carregar múltiplos URLs
	urls := []string{
		"https://example.com",
		"https://www.ietf.org/rfc/rfc2616.txt",
	}

	loader := knowledge.NewMultiURLLoader(urls)
	loader.MaxConcurrent = 2
	loader.ContinueOnErr = true
	docs, err := loader.Load()

	if err != nil {
		fmt.Printf("⚠️  Alguns URLs falharam: %v\n", err)
	}

	fmt.Printf("✓ Carregados: %d documento(s) de %d URLs\n", len(docs), len(urls))

	fmt.Println("\n✓ Carregamento concorrente")
	fmt.Println("✓ Rate limiting (MaxConcurrent)")
	fmt.Println("✓ Exemplo:")
	fmt.Println("  urls := []string{\"url1\", \"url2\", \"url3\"}")
	fmt.Println("  loader := knowledge.NewMultiURLLoader(urls)")
	fmt.Println("  docs, err := loader.Load()")
	fmt.Println()
}

func demoReaderLoader() {
	fmt.Println("📥 9. ReaderLoader - Streams")
	fmt.Println("----------------------------")

	// Exemplo: carregar de um buffer
	// content := []byte("Este é um exemplo de conteúdo")
	// reader := bytes.NewReader(content)
	// loader := knowledge.NewReaderLoader(reader, "exemplo-id", nil)
	// docs, err := loader.Load()

	fmt.Println("✓ Funciona com qualquer io.Reader")
	fmt.Println("✓ HTTP responses, stdin, buffers")
	fmt.Println("✓ Exemplo:")
	fmt.Println("  reader := bytes.NewReader(data)")
	fmt.Println("  loader := knowledge.NewReaderLoader(reader, \"id\", nil)")
	fmt.Println("  docs, err := loader.Load()")
	fmt.Println()
}

func demoChunkers(ctx context.Context) {
	fmt.Println("✂️  10. Chunkers - Divisão de documentos")
	fmt.Println("---------------------------------------")

	// Exemplo: chunkar um documento
	doc := knowledge.Document{
		ID:      "exemplo",
		Content: "Este é um exemplo de documento. Ele possui várias sentenças. E será dividido em chunks menores para processamento. Cada chunk mantém contexto através de overlap.",
		Source:  "demo",
		Metadata: map[string]interface{}{
			"tipo": "exemplo",
		},
	}

	// Character Chunker
	charChunker := knowledge.NewCharacterChunker(50, 10)
	charChunks, err := charChunker.Chunk(doc)
	if err != nil {
		log.Printf("Erro ao chunkar: %v\n", err)
	} else {
		fmt.Printf("✓ CharacterChunker: %d chunks gerados\n", len(charChunks))
	}

	// Sentence Chunker
	sentChunker := knowledge.NewSentenceChunker(100, 30)
	sentChunks, err := sentChunker.Chunk(doc)
	if err != nil {
		log.Printf("Erro ao chunkar: %v\n", err)
	} else {
		fmt.Printf("✓ SentenceChunker: %d chunks gerados\n", len(sentChunks))
	}

	// Paragraph Chunker
	paraChunker := knowledge.NewParagraphChunker(200)
	paraChunks, err := paraChunker.Chunk(doc)
	if err != nil {
		log.Printf("Erro ao chunkar: %v\n", err)
	} else {
		fmt.Printf("✓ ParagraphChunker: %d chunks gerados\n", len(paraChunks))
	}

	fmt.Println("\n✓ CharacterChunker - Por caracteres")
	fmt.Println("✓ SentenceChunker - Por sentenças")
	fmt.Println("✓ ParagraphChunker - Por parágrafos")
	fmt.Println()
}
