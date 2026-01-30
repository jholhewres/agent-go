package knowledge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// HTMLLoader loads documents from HTML files
type HTMLLoader struct {
	FilePath        string
	RemoveScripts   bool     // Remove <script> tags
	RemoveStyles    bool     // Remove <style> tags
	ExtractMetaTags bool     // Extract meta tags as metadata
	Selectors       []string // CSS selectors to extract specific content (nil = extract all)
	PreserveLinks   bool     // Keep links in content
}

// NewHTMLLoader creates a new HTML loader
func NewHTMLLoader(filePath string) *HTMLLoader {
	return &HTMLLoader{
		FilePath:        filePath,
		RemoveScripts:   true,
		RemoveStyles:    true,
		ExtractMetaTags: true,
		PreserveLinks:   false,
	}
}

// Load loads an HTML file
func (l *HTMLLoader) Load() ([]Document, error) {
	file, err := os.Open(l.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open HTML file %s: %w", l.FilePath, err)
	}
	defer file.Close()

	return l.loadFromReader(file, filepath.Base(l.FilePath), l.FilePath)
}

func (l *HTMLLoader) loadFromReader(reader io.Reader, id, source string) ([]Document, error) {
	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Remove scripts and styles if requested
	if l.RemoveScripts {
		doc.Find("script").Remove()
		doc.Find("noscript").Remove()
	}
	if l.RemoveStyles {
		doc.Find("style").Remove()
	}

	// Extract metadata from meta tags
	metadata := map[string]interface{}{
		"filename":  id,
		"path":      source,
		"ext":       filepath.Ext(source),
		"file_type": "html",
	}

	if l.ExtractMetaTags {
		doc.Find("meta").Each(func(i int, s *goquery.Selection) {
			if name, exists := s.Attr("name"); exists {
				if content, exists := s.Attr("content"); exists {
					metadata["meta_"+name] = content
				}
			}
			if property, exists := s.Attr("property"); exists {
				if content, exists := s.Attr("content"); exists {
					metadata["meta_"+property] = content
				}
			}
		})

		// Extract title
		title := doc.Find("title").Text()
		if title != "" {
			metadata["title"] = strings.TrimSpace(title)
		}
	}

	// Extract content
	var content string
	if len(l.Selectors) > 0 {
		// Extract from specific selectors
		var parts []string
		for _, selector := range l.Selectors {
			doc.Find(selector).Each(func(i int, s *goquery.Selection) {
				text := l.extractText(s)
				if text != "" {
					parts = append(parts, text)
				}
			})
		}
		content = strings.Join(parts, "\n\n")
	} else {
		// Extract from body
		body := doc.Find("body")
		if body.Length() == 0 {
			// No body tag, use entire document
			content = l.extractText(doc.Selection)
		} else {
			content = l.extractText(body)
		}
	}

	content = l.cleanText(content)

	if content == "" {
		return nil, fmt.Errorf("no text content extracted from HTML")
	}

	document := Document{
		ID:       id,
		Content:  content,
		Source:   source,
		Metadata: metadata,
	}

	return []Document{document}, nil
}

func (l *HTMLLoader) extractText(selection *goquery.Selection) string {
	if l.PreserveLinks {
		// Extract text with links
		var text strings.Builder
		selection.Contents().Each(func(i int, s *goquery.Selection) {
			if goquery.NodeName(s) == "#text" {
				text.WriteString(s.Text())
			} else if goquery.NodeName(s) == "a" {
				linkText := s.Text()
				href, exists := s.Attr("href")
				if exists && href != "" {
					text.WriteString(fmt.Sprintf("%s (%s)", linkText, href))
				} else {
					text.WriteString(linkText)
				}
			} else {
				text.WriteString(l.extractText(s))
			}
		})
		return text.String()
	}

	return selection.Text()
}

func (l *HTMLLoader) cleanText(text string) string {
	// Remove excessive whitespace
	lines := strings.Split(text, "\n")
	var cleaned []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

// HTMLReaderLoader loads HTML from an io.Reader
type HTMLReaderLoader struct {
	Reader          io.Reader
	ID              string
	RemoveScripts   bool
	RemoveStyles    bool
	ExtractMetaTags bool
	Selectors       []string
	PreserveLinks   bool
	Metadata        map[string]interface{}
}

// NewHTMLReaderLoader creates a new HTML reader loader
func NewHTMLReaderLoader(reader io.Reader, id string, metadata map[string]interface{}) *HTMLReaderLoader {
	return &HTMLReaderLoader{
		Reader:          reader,
		ID:              id,
		RemoveScripts:   true,
		RemoveStyles:    true,
		ExtractMetaTags: true,
		Metadata:        metadata,
	}
}

// Load loads HTML content from a reader
func (l *HTMLReaderLoader) Load() ([]Document, error) {
	loader := &HTMLLoader{
		RemoveScripts:   l.RemoveScripts,
		RemoveStyles:    l.RemoveStyles,
		ExtractMetaTags: l.ExtractMetaTags,
		Selectors:       l.Selectors,
		PreserveLinks:   l.PreserveLinks,
	}

	docs, err := loader.loadFromReader(l.Reader, l.ID, "")
	if err != nil {
		return nil, err
	}

	// Add custom metadata
	if l.Metadata != nil {
		for i := range docs {
			for k, v := range l.Metadata {
				docs[i].Metadata[k] = v
			}
		}
	}

	return docs, nil
}
