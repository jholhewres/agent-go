package knowledge

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CSVLoader loads documents from CSV files
type CSVLoader struct {
	FilePath      string
	Delimiter     rune   // Default: ','
	HasHeader     bool   // Whether first row is header
	TextColumns   []int  // Indices of columns to include (nil = all)
	RowsPerDoc    int    // Number of rows per document (0 = all in one doc)
	ContentFormat string // "json" or "text" (default: "text")
}

// NewCSVLoader creates a new CSV loader
func NewCSVLoader(filePath string) *CSVLoader {
	return &CSVLoader{
		FilePath:      filePath,
		Delimiter:     ',',
		HasHeader:     true,
		RowsPerDoc:    0, // All rows in one document by default
		ContentFormat: "text",
	}
}

// Load loads a CSV file
func (l *CSVLoader) Load() ([]Document, error) {
	file, err := os.Open(l.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file %s: %w", l.FilePath, err)
	}
	defer file.Close()

	return l.loadFromReader(file, filepath.Base(l.FilePath), l.FilePath)
}

func (l *CSVLoader) loadFromReader(reader io.Reader, id, source string) ([]Document, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = l.Delimiter
	csvReader.TrimLeadingSpace = true

	// Read all records
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Extract header
	var headers []string
	startRow := 0
	if l.HasHeader && len(records) > 0 {
		headers = records[0]
		startRow = 1
	} else {
		// Generate generic headers
		if len(records) > 0 {
			for i := range records[0] {
				headers = append(headers, fmt.Sprintf("column_%d", i))
			}
		}
	}

	dataRows := records[startRow:]
	if len(dataRows) == 0 {
		return nil, fmt.Errorf("no data rows in CSV")
	}

	// Determine how to split into documents
	var documents []Document

	if l.RowsPerDoc == 0 {
		// All rows in one document
		content := l.formatRows(headers, dataRows)
		doc := Document{
			ID:      id,
			Content: content,
			Source:  source,
			Metadata: map[string]interface{}{
				"filename":   id,
				"path":       source,
				"ext":        ".csv",
				"file_type":  "csv",
				"rows":       len(dataRows),
				"columns":    len(headers),
				"headers":    headers,
				"has_header": l.HasHeader,
			},
		}
		documents = append(documents, doc)
	} else {
		// Split into multiple documents
		for i := 0; i < len(dataRows); i += l.RowsPerDoc {
			end := i + l.RowsPerDoc
			if end > len(dataRows) {
				end = len(dataRows)
			}

			chunk := dataRows[i:end]
			content := l.formatRows(headers, chunk)

			doc := Document{
				ID:      fmt.Sprintf("%s_rows_%d_%d", id, i+startRow, end+startRow-1),
				Content: content,
				Source:  source,
				Metadata: map[string]interface{}{
					"filename":   id,
					"path":       source,
					"ext":        ".csv",
					"file_type":  "csv",
					"row_start":  i + startRow,
					"row_end":    end + startRow - 1,
					"rows":       len(chunk),
					"columns":    len(headers),
					"headers":    headers,
					"has_header": l.HasHeader,
				},
			}
			documents = append(documents, doc)
		}
	}

	return documents, nil
}

func (l *CSVLoader) formatRows(headers []string, rows [][]string) string {
	var content strings.Builder

	if l.ContentFormat == "json" {
		// Format as JSON-like text
		content.WriteString("[\n")
		for i, row := range rows {
			content.WriteString("  {\n")
			for j, value := range row {
				if j < len(headers) {
					content.WriteString(fmt.Sprintf("    \"%s\": \"%s\"", headers[j], value))
					if j < len(row)-1 {
						content.WriteString(",")
					}
					content.WriteString("\n")
				}
			}
			content.WriteString("  }")
			if i < len(rows)-1 {
				content.WriteString(",")
			}
			content.WriteString("\n")
		}
		content.WriteString("]")
	} else {
		// Format as plain text with headers
		content.WriteString("Headers: " + strings.Join(headers, " | ") + "\n\n")

		for i, row := range rows {
			// Filter columns if specified
			var cols []string
			if l.TextColumns != nil {
				for _, idx := range l.TextColumns {
					if idx < len(row) {
						cols = append(cols, row[idx])
					}
				}
			} else {
				cols = row
			}

			content.WriteString(fmt.Sprintf("Row %d: %s\n", i+1, strings.Join(cols, " | ")))
		}
	}

	return content.String()
}

// CSVReaderLoader loads CSV from an io.Reader
type CSVReaderLoader struct {
	Reader        io.Reader
	ID            string
	Delimiter     rune
	HasHeader     bool
	RowsPerDoc    int
	ContentFormat string
	Metadata      map[string]interface{}
}

// NewCSVReaderLoader creates a new CSV reader loader
func NewCSVReaderLoader(reader io.Reader, id string, metadata map[string]interface{}) *CSVReaderLoader {
	return &CSVReaderLoader{
		Reader:        reader,
		ID:            id,
		Delimiter:     ',',
		HasHeader:     true,
		RowsPerDoc:    0,
		ContentFormat: "text",
		Metadata:      metadata,
	}
}

// Load loads CSV content from a reader
func (l *CSVReaderLoader) Load() ([]Document, error) {
	loader := &CSVLoader{
		Delimiter:     l.Delimiter,
		HasHeader:     l.HasHeader,
		RowsPerDoc:    l.RowsPerDoc,
		ContentFormat: l.ContentFormat,
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
