package knowledge

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// JSONLoader loads documents from JSON files
type JSONLoader struct {
	FilePath       string
	JSONPath       string   // JSONPath expression to extract content (optional)
	ContentFields  []string // Fields to use as content (if JSON is object/array of objects)
	MetadataFields []string // Fields to include in metadata
	TextTemplate   string   // Template for formatting content (e.g., "{title}: {body}")
}

// NewJSONLoader creates a new JSON loader
func NewJSONLoader(filePath string) *JSONLoader {
	return &JSONLoader{
		FilePath: filePath,
	}
}

// Load loads a JSON file
func (l *JSONLoader) Load() ([]Document, error) {
	file, err := os.Open(l.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSON file %s: %w", l.FilePath, err)
	}
	defer file.Close()

	return l.loadFromReader(file, filepath.Base(l.FilePath), l.FilePath)
}

func (l *JSONLoader) loadFromReader(reader io.Reader, id, source string) ([]Document, error) {
	// Read JSON content
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON: %w", err)
	}

	// Parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract documents based on JSON structure
	documents, err := l.extractDocuments(jsonData, id, source)
	if err != nil {
		return nil, err
	}

	if len(documents) == 0 {
		return nil, fmt.Errorf("no documents extracted from JSON")
	}

	return documents, nil
}

func (l *JSONLoader) extractDocuments(data interface{}, id, source string) ([]Document, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Single object
		return l.extractFromObject(v, id, source)

	case []interface{}:
		// Array of objects
		return l.extractFromArray(v, id, source)

	case string:
		// Plain string
		doc := Document{
			ID:      id,
			Content: v,
			Source:  source,
			Metadata: map[string]interface{}{
				"filename":  id,
				"path":      source,
				"ext":       ".json",
				"file_type": "json",
			},
		}
		return []Document{doc}, nil

	default:
		return nil, fmt.Errorf("unsupported JSON structure: %T", v)
	}
}

func (l *JSONLoader) extractFromObject(obj map[string]interface{}, id, source string) ([]Document, error) {
	// Extract content from specified fields or use all fields
	var contentParts []string

	if len(l.ContentFields) > 0 {
		// Use specified content fields
		for _, field := range l.ContentFields {
			if value, exists := obj[field]; exists {
				contentParts = append(contentParts, fmt.Sprintf("%v", value))
			}
		}
	} else {
		// Use all string fields as content
		for key, value := range obj {
			if str, ok := value.(string); ok {
				contentParts = append(contentParts, fmt.Sprintf("%s: %s", key, str))
			}
		}
	}

	if len(contentParts) == 0 {
		// Fall back to JSON string representation
		jsonBytes, _ := json.MarshalIndent(obj, "", "  ")
		contentParts = append(contentParts, string(jsonBytes))
	}

	content := strings.Join(contentParts, "\n\n")

	// Extract metadata
	metadata := map[string]interface{}{
		"filename":  id,
		"path":      source,
		"ext":       ".json",
		"file_type": "json",
	}

	// Add metadata fields
	if len(l.MetadataFields) > 0 {
		for _, field := range l.MetadataFields {
			if value, exists := obj[field]; exists {
				metadata[field] = value
			}
		}
	} else {
		// Add all non-content fields as metadata
		for key, value := range obj {
			// Skip if it's a content field
			isContentField := false
			for _, cf := range l.ContentFields {
				if key == cf {
					isContentField = true
					break
				}
			}
			if !isContentField {
				// Only add simple types to metadata
				switch value.(type) {
				case string, int, int64, float64, bool:
					metadata[key] = value
				}
			}
		}
	}

	doc := Document{
		ID:       id,
		Content:  content,
		Source:   source,
		Metadata: metadata,
	}

	return []Document{doc}, nil
}

func (l *JSONLoader) extractFromArray(arr []interface{}, baseID, source string) ([]Document, error) {
	var documents []Document

	for i, item := range arr {
		itemID := fmt.Sprintf("%s_item_%d", baseID, i)

		switch v := item.(type) {
		case map[string]interface{}:
			docs, err := l.extractFromObject(v, itemID, source)
			if err != nil {
				return nil, fmt.Errorf("failed to extract item %d: %w", i, err)
			}
			documents = append(documents, docs...)

		case string:
			doc := Document{
				ID:      itemID,
				Content: v,
				Source:  source,
				Metadata: map[string]interface{}{
					"filename":  baseID,
					"path":      source,
					"ext":       ".json",
					"file_type": "json",
					"index":     i,
				},
			}
			documents = append(documents, doc)

		default:
			// Convert to JSON string
			jsonBytes, _ := json.Marshal(item)
			doc := Document{
				ID:      itemID,
				Content: string(jsonBytes),
				Source:  source,
				Metadata: map[string]interface{}{
					"filename":  baseID,
					"path":      source,
					"ext":       ".json",
					"file_type": "json",
					"index":     i,
				},
			}
			documents = append(documents, doc)
		}
	}

	return documents, nil
}

// JSONReaderLoader loads JSON from an io.Reader
type JSONReaderLoader struct {
	Reader         io.Reader
	ID             string
	ContentFields  []string
	MetadataFields []string
	Metadata       map[string]interface{}
}

// NewJSONReaderLoader creates a new JSON reader loader
func NewJSONReaderLoader(reader io.Reader, id string, metadata map[string]interface{}) *JSONReaderLoader {
	return &JSONReaderLoader{
		Reader:   reader,
		ID:       id,
		Metadata: metadata,
	}
}

// Load loads JSON content from a reader
func (l *JSONReaderLoader) Load() ([]Document, error) {
	loader := &JSONLoader{
		ContentFields:  l.ContentFields,
		MetadataFields: l.MetadataFields,
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
