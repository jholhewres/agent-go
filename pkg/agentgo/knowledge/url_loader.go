package knowledge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// URLLoader loads documents from URLs (web pages, APIs, etc.)
type URLLoader struct {
	URL            string
	Method         string            // HTTP method (default: GET)
	Headers        map[string]string // Custom headers
	Timeout        time.Duration     // Request timeout (default: 30s)
	FollowRedirect bool              // Follow redirects (default: true)
	UserAgent      string            // User agent string
	ContentType    string            // Expected content type (html, json, pdf, text)
	AutoDetect     bool              // Auto-detect content type from response
}

// NewURLLoader creates a new URL loader
func NewURLLoader(url string) *URLLoader {
	return &URLLoader{
		URL:            url,
		Method:         "GET",
		Timeout:        30 * time.Second,
		FollowRedirect: true,
		UserAgent:      "AgentGo-Knowledge-Loader/1.0",
		AutoDetect:     true,
	}
}

// Load fetches content from URL and loads it as document
func (l *URLLoader) Load() ([]Document, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.Timeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, l.Method, l.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", l.UserAgent)
	for key, value := range l.Headers {
		req.Header.Set(key, value)
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: l.Timeout,
	}

	if !l.FollowRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", l.URL, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Detect content type
	contentType := l.ContentType
	if l.AutoDetect || contentType == "" {
		contentType = l.detectContentType(resp.Header.Get("Content-Type"), body)
	}

	// Route to appropriate loader based on content type
	return l.loadByContentType(contentType, bytes.NewReader(body))
}

func (l *URLLoader) detectContentType(headerContentType string, body []byte) string {
	// Check Content-Type header first
	if headerContentType != "" {
		lowerType := strings.ToLower(headerContentType)
		switch {
		case strings.Contains(lowerType, "html"):
			return "html"
		case strings.Contains(lowerType, "json"):
			return "json"
		case strings.Contains(lowerType, "pdf"):
			return "pdf"
		case strings.Contains(lowerType, "xml"):
			return "xml"
		case strings.Contains(lowerType, "text"):
			return "text"
		}
	}

	// Try to detect from content
	if len(body) > 0 {
		preview := string(body[:min(512, len(body))])
		lowerPreview := strings.ToLower(preview)

		if strings.HasPrefix(preview, "%PDF") {
			return "pdf"
		}
		if strings.Contains(lowerPreview, "<!doctype html") || strings.Contains(lowerPreview, "<html") {
			return "html"
		}
		if strings.HasPrefix(strings.TrimSpace(preview), "{") || strings.HasPrefix(strings.TrimSpace(preview), "[") {
			return "json"
		}
		if strings.Contains(lowerPreview, "<?xml") {
			return "xml"
		}
	}

	return "text"
}

func (l *URLLoader) loadByContentType(contentType string, reader io.Reader) ([]Document, error) {
	id := l.getIDFromURL()
	metadata := map[string]interface{}{
		"url":          l.URL,
		"content_type": contentType,
		"source_type":  "url",
	}

	switch contentType {
	case "html":
		loader := NewHTMLReaderLoader(reader, id, metadata)
		return loader.Load()

	case "json":
		loader := NewJSONReaderLoader(reader, id, metadata)
		return loader.Load()

	case "pdf":
		// For PDF, we need a ReadSeeker
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read PDF data: %w", err)
		}
		loader := NewPDFReaderLoader(data, id, metadata)
		return loader.Load()

	case "text", "xml":
		fallthrough
	default:
		// Default to text loader
		loader := NewReaderLoader(reader, id, metadata)
		return loader.Load()
	}
}

func (l *URLLoader) getIDFromURL() string {
	// Extract a reasonable ID from URL
	parts := strings.Split(strings.TrimRight(l.URL, "/"), "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if lastPart != "" && !strings.Contains(lastPart, "?") {
			return lastPart
		}
	}

	// Fall back to domain name
	if strings.HasPrefix(l.URL, "http://") {
		return strings.TrimPrefix(l.URL, "http://")
	}
	if strings.HasPrefix(l.URL, "https://") {
		return strings.TrimPrefix(l.URL, "https://")
	}

	return l.URL
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MultiURLLoader loads documents from multiple URLs
type MultiURLLoader struct {
	URLs           []string
	Timeout        time.Duration
	MaxConcurrent  int // Maximum concurrent requests (default: 5)
	ContinueOnErr  bool
	CommonHeaders  map[string]string
	CommonMetadata map[string]interface{}
}

// NewMultiURLLoader creates a new multi-URL loader
func NewMultiURLLoader(urls []string) *MultiURLLoader {
	return &MultiURLLoader{
		URLs:          urls,
		Timeout:       30 * time.Second,
		MaxConcurrent: 5,
		ContinueOnErr: true,
	}
}

// Load fetches content from multiple URLs concurrently
func (l *MultiURLLoader) Load() ([]Document, error) {
	type result struct {
		docs []Document
		err  error
	}

	results := make(chan result, len(l.URLs))
	semaphore := make(chan struct{}, l.MaxConcurrent)

	// Launch concurrent loaders
	for _, url := range l.URLs {
		go func(u string) {
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			loader := NewURLLoader(u)
			loader.Timeout = l.Timeout
			loader.Headers = l.CommonHeaders

			docs, err := loader.Load()

			// Add common metadata
			if l.CommonMetadata != nil {
				for i := range docs {
					for k, v := range l.CommonMetadata {
						docs[i].Metadata[k] = v
					}
				}
			}

			results <- result{docs: docs, err: err}
		}(url)
	}

	// Collect results
	var allDocs []Document
	var errors []error

	for i := 0; i < len(l.URLs); i++ {
		res := <-results
		if res.err != nil {
			errors = append(errors, res.err)
			if !l.ContinueOnErr {
				return nil, fmt.Errorf("failed to load URLs: %v", errors)
			}
		} else {
			allDocs = append(allDocs, res.docs...)
		}
	}

	if len(allDocs) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("all URLs failed to load: %v", errors)
	}

	return allDocs, nil
}

// WebCrawler crawls web pages starting from a URL
type WebCrawler struct {
	StartURL      string
	MaxDepth      int               // Maximum crawl depth (default: 2)
	MaxPages      int               // Maximum pages to crawl (default: 10)
	SameDomain    bool              // Only crawl same domain (default: true)
	IncludeFilter []string          // URL patterns to include
	ExcludeFilter []string          // URL patterns to exclude
	Timeout       time.Duration     // Request timeout per page
	Headers       map[string]string // Custom headers
}

// NewWebCrawler creates a new web crawler
func NewWebCrawler(startURL string) *WebCrawler {
	return &WebCrawler{
		StartURL:   startURL,
		MaxDepth:   2,
		MaxPages:   10,
		SameDomain: true,
		Timeout:    30 * time.Second,
	}
}

// Load crawls web pages and loads them as documents
// Note: This is a basic implementation. For production, consider using a dedicated crawler library
func (c *WebCrawler) Load() ([]Document, error) {
	// TODO: Implement web crawling logic
	// For now, just load the start URL
	loader := NewURLLoader(c.StartURL)
	loader.Timeout = c.Timeout
	loader.Headers = c.Headers
	return loader.Load()
}
