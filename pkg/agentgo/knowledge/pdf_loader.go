package knowledge

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// PDFLoader loads documents from PDF files
type PDFLoader struct {
	FilePath       string
	ExtractImages  bool // Future: extract images from PDF
	PageSeparator  string
	PreserveLayout bool // Try to preserve text layout
}

// NewPDFLoader creates a new PDF loader
func NewPDFLoader(filePath string) *PDFLoader {
	return &PDFLoader{
		FilePath:      filePath,
		PageSeparator: "\n\n---\n\n", // Separator between pages
	}
}

// Load loads a PDF file and extracts text content
func (l *PDFLoader) Load() ([]Document, error) {
	// Open PDF file
	file, reader, err := pdf.Open(l.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF %s: %w", l.FilePath, err)
	}
	defer file.Close()

	// Extract text from all pages
	var contentBuilder strings.Builder
	numPages := reader.NumPage()

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		// Extract text from page
		text, err := page.GetPlainText(nil)
		if err != nil {
			// Log warning but continue with other pages
			fmt.Printf("Warning: failed to extract text from page %d: %v\n", pageNum, err)
			continue
		}

		// Add page content
		if contentBuilder.Len() > 0 {
			contentBuilder.WriteString(l.PageSeparator)
		}
		contentBuilder.WriteString(strings.TrimSpace(text))
	}

	content := contentBuilder.String()
	if content == "" {
		return nil, fmt.Errorf("no text content extracted from PDF: %s", l.FilePath)
	}

	doc := Document{
		ID:      filepath.Base(l.FilePath),
		Content: content,
		Source:  l.FilePath,
		Metadata: map[string]interface{}{
			"filename":  filepath.Base(l.FilePath),
			"path":      l.FilePath,
			"ext":       filepath.Ext(l.FilePath),
			"pages":     numPages,
			"file_type": "pdf",
		},
	}

	return []Document{doc}, nil
}

// PDFDirectoryLoader loads all PDF files from a directory
type PDFDirectoryLoader struct {
	DirPath   string
	Recursive bool
}

// NewPDFDirectoryLoader creates a new PDF directory loader
func NewPDFDirectoryLoader(dirPath string, recursive bool) *PDFDirectoryLoader {
	return &PDFDirectoryLoader{
		DirPath:   dirPath,
		Recursive: recursive,
	}
}

// Load loads all PDF files from a directory
func (l *PDFDirectoryLoader) Load() ([]Document, error) {
	var documents []Document

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			if !l.Recursive && path != l.DirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file is PDF
		if strings.ToLower(filepath.Ext(path)) != ".pdf" {
			return nil
		}

		// Load PDF
		loader := NewPDFLoader(path)
		docs, err := loader.Load()
		if err != nil {
			// Log warning but continue with other files
			fmt.Printf("Warning: failed to load PDF %s: %v\n", path, err)
			return nil
		}

		documents = append(documents, docs...)
		return nil
	}

	err := filepath.Walk(l.DirPath, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", l.DirPath, err)
	}

	return documents, nil
}

// PDFReaderLoader loads PDF from a byte slice (in-memory PDF)
type PDFReaderLoader struct {
	Data     []byte
	ID       string
	Metadata map[string]interface{}
}

// NewPDFReaderLoader creates a new PDF reader loader from bytes
func NewPDFReaderLoader(data []byte, id string, metadata map[string]interface{}) *PDFReaderLoader {
	return &PDFReaderLoader{
		Data:     data,
		ID:       id,
		Metadata: metadata,
	}
}

// Load loads PDF content from bytes
func (l *PDFReaderLoader) Load() ([]Document, error) {
	reader, err := pdf.NewReader(bytes.NewReader(l.Data), int64(len(l.Data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	// Extract text from all pages
	var contentBuilder strings.Builder
	numPages := reader.NumPage()

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			fmt.Printf("Warning: failed to extract text from page %d: %v\n", pageNum, err)
			continue
		}

		if contentBuilder.Len() > 0 {
			contentBuilder.WriteString("\n\n---\n\n")
		}
		contentBuilder.WriteString(strings.TrimSpace(text))
	}

	content := contentBuilder.String()
	if content == "" {
		return nil, fmt.Errorf("no text content extracted from PDF")
	}

	metadata := l.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["pages"] = numPages
	metadata["file_type"] = "pdf"

	doc := Document{
		ID:       l.ID,
		Content:  content,
		Metadata: metadata,
	}

	return []Document{doc}, nil
}
