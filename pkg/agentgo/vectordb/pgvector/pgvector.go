package pgvector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/vectordb"
	"github.com/pgvector/pgvector-go"
)

// PgVector implements vectordb.VectorDB using PostgreSQL with pgvector extension
type PgVector struct {
	db              *sql.DB
	tableName       string
	dimension       int
	indexType       string // "ivfflat" or "hnsw"
	indexParams     map[string]interface{}
	embedFunc       vectordb.EmbeddingFunction
	collectionName  string
}

// Config holds pgvector configuration
type Config struct {
	DB              *sql.DB
	TableName       string
	CollectionName  string
	Dimension       int
	IndexType       string                 // "ivfflat" or "hnsw" (default: "hnsw")
	IndexParams     map[string]interface{} // Index-specific parameters
	EmbeddingFunc   vectordb.EmbeddingFunction
}

// New creates a new PgVector instance
func New(config Config) (*PgVector, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	if config.TableName == "" {
		config.TableName = "vector_documents"
	}

	if config.CollectionName == "" {
		config.CollectionName = "default"
	}

	if config.Dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}

	if config.IndexType == "" {
		config.IndexType = "hnsw"
	}

	if config.IndexType != "ivfflat" && config.IndexType != "hnsw" {
		return nil, fmt.Errorf("index_type must be 'ivfflat' or 'hnsw'")
	}

	pv := &PgVector{
		db:             config.DB,
		tableName:      config.TableName,
		dimension:      config.Dimension,
		indexType:      config.IndexType,
		indexParams:    config.IndexParams,
		embedFunc:      config.EmbeddingFunc,
		collectionName: config.CollectionName,
	}

	if err := pv.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return pv, nil
}

// migrate creates the necessary tables and indexes
func (pv *PgVector) migrate() error {
	ctx := context.Background()

	// Enable pgvector extension
	_, err := pv.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create table
	createTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255),
			collection VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			embedding vector(%d) NOT NULL,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (id, collection)
		)
	`, pv.tableName, pv.dimension)

	_, err = pv.db.ExecContext(ctx, createTable)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create vector index
	indexName := fmt.Sprintf("idx_%s_embedding_%s", pv.tableName, pv.indexType)
	var createIndex string

	if pv.indexType == "ivfflat" {
		lists := 100 // Default lists for IVFFlat
		if val, ok := pv.indexParams["lists"].(int); ok {
			lists = val
		}
		createIndex = fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s
			USING ivfflat (embedding vector_cosine_ops)
			WITH (lists = %d)
		`, indexName, pv.tableName, lists)
	} else {
		m := 16 // Default m for HNSW
		efConstruction := 64
		if val, ok := pv.indexParams["m"].(int); ok {
			m = val
		}
		if val, ok := pv.indexParams["ef_construction"].(int); ok {
			efConstruction = val
		}
		createIndex = fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s
			USING hnsw (embedding vector_cosine_ops)
			WITH (m = %d, ef_construction = %d)
		`, indexName, pv.tableName, m, efConstruction)
	}

	_, err = pv.db.ExecContext(ctx, createIndex)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Create metadata index
	metadataIndex := fmt.Sprintf("idx_%s_metadata", pv.tableName)
	createMetadataIndex := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s ON %s USING GIN(metadata)
	`, metadataIndex, pv.tableName)

	_, err = pv.db.ExecContext(ctx, createMetadataIndex)
	if err != nil {
		return fmt.Errorf("failed to create metadata index: %w", err)
	}

	return nil
}

// CreateCollection creates a new collection (pgvector uses single table with collection column)
func (pv *PgVector) CreateCollection(ctx context.Context, name string, metadata map[string]interface{}) error {
	// pgvector doesn't need explicit collection creation, just store the name
	pv.collectionName = name
	return nil
}

// DeleteCollection deletes all documents in a collection
func (pv *PgVector) DeleteCollection(ctx context.Context, name string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE collection = $1`, pv.tableName)
	_, err := pv.db.ExecContext(ctx, query, name)
	return err
}

// Add adds documents to the collection
func (pv *PgVector) Add(ctx context.Context, docs []vectordb.Document) error {
	return pv.upsert(ctx, docs, false)
}

// Update updates existing documents
func (pv *PgVector) Update(ctx context.Context, docs []vectordb.Document) error {
	return pv.upsert(ctx, docs, true)
}

// upsert inserts or updates documents
func (pv *PgVector) upsert(ctx context.Context, docs []vectordb.Document, updateOnly bool) error {
	if len(docs) == 0 {
		return nil
	}

	tx, err := pv.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var stmt *sql.Stmt
	if updateOnly {
		stmt, err = tx.PrepareContext(ctx, fmt.Sprintf(`
			UPDATE %s SET content = $1, embedding = $2, metadata = $3
			WHERE id = $4 AND collection = $5
		`, pv.tableName))
	} else {
		stmt, err = tx.PrepareContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (id, collection, content, embedding, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id, collection) DO UPDATE
			SET content = EXCLUDED.content, embedding = EXCLUDED.embedding, metadata = EXCLUDED.metadata
		`, pv.tableName))
	}

	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, doc := range docs {
		metadataJSON, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		if doc.CreatedAt.IsZero() {
			doc.CreatedAt = time.Now()
		}

		if updateOnly {
			_, err = stmt.ExecContext(ctx, doc.Content, pgvector.NewVector(doc.Embedding), metadataJSON, doc.ID, pv.collectionName)
		} else {
			_, err = stmt.ExecContext(ctx, doc.ID, pv.collectionName, doc.Content, pgvector.NewVector(doc.Embedding), metadataJSON, doc.CreatedAt)
		}

		if err != nil {
			return fmt.Errorf("failed to upsert document: %w", err)
		}
	}

	return tx.Commit()
}

// Query searches using text query (requires embedding function)
func (pv *PgVector) Query(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]vectordb.SearchResult, error) {
	if pv.embedFunc == nil {
		return nil, fmt.Errorf("embedding function is required for text queries")
	}

	// Generate embedding for query
	embedding, err := pv.embedFunc.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	return pv.QueryWithEmbedding(ctx, embedding, limit, filter)
}

// QueryWithEmbedding searches using pre-computed embedding
func (pv *PgVector) QueryWithEmbedding(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]vectordb.SearchResult, error) {
	if len(embedding) != pv.dimension {
		return nil, fmt.Errorf("embedding dimension mismatch: expected %d, got %d", pv.dimension, len(embedding))
	}

	if limit <= 0 {
		limit = 10
	}

	// Build query
	query := fmt.Sprintf(`
		SELECT id, content, metadata, 1 - (embedding <=> $1) AS score, embedding <=> $1 AS distance
		FROM %s
		WHERE collection = $2
	`, pv.tableName)

	args := []interface{}{pgvector.NewVector(embedding), pv.collectionName}
	argIdx := 3

	// Add metadata filter
	if filter != nil && len(filter) > 0 {
		for key, value := range filter {
			query += fmt.Sprintf(" AND metadata->>'%s' = $%d", key, argIdx)
			args = append(args, value)
			argIdx++
		}
	}

	query += fmt.Sprintf(" ORDER BY embedding <=> $1 LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := pv.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	var results []vectordb.SearchResult
	for rows.Next() {
		var result vectordb.SearchResult
		var metadataJSON []byte

		err := rows.Scan(&result.ID, &result.Content, &metadataJSON, &result.Score, &result.Distance)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		json.Unmarshal(metadataJSON, &result.Metadata)
		results = append(results, result)
	}

	return results, rows.Err()
}

// Get retrieves documents by IDs
func (pv *PgVector) Get(ctx context.Context, ids []string) ([]vectordb.Document, error) {
	if len(ids) == 0 {
		return []vectordb.Document{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+1)
	args[0] = pv.collectionName

	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		SELECT id, content, embedding, metadata, created_at
		FROM %s
		WHERE collection = $1 AND id IN (%s)
	`, pv.tableName, strings.Join(placeholders, ","))

	rows, err := pv.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []vectordb.Document
	for rows.Next() {
		var doc vectordb.Document
		var embeddingVec pgvector.Vector
		var metadataJSON []byte

		err := rows.Scan(&doc.ID, &doc.Content, &embeddingVec, &metadataJSON, &doc.CreatedAt)
		if err != nil {
			return nil, err
		}

		doc.Embedding = embeddingVec.Slice()
		json.Unmarshal(metadataJSON, &doc.Metadata)
		documents = append(documents, doc)
	}

	return documents, rows.Err()
}

// Count returns the number of documents in the collection
func (pv *PgVector) Count(ctx context.Context) (int, error) {
	var count int
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE collection = $1`, pv.tableName)
	err := pv.db.QueryRowContext(ctx, query, pv.collectionName).Scan(&count)
	return count, err
}

// Delete deletes documents by IDs
func (pv *PgVector) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Build placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE id IN (%s)`,
		pv.tableName,
		strings.Join(placeholders, ","))

	_, err := pv.db.ExecContext(ctx, query, args...)
	return err
}

// Close closes the database connection
func (pv *PgVector) Close() error {
	return pv.db.Close()
}

// GetDimension returns the embedding dimension
func (pv *PgVector) GetDimension() int {
	return pv.dimension
}

// GetTableName returns the table name
func (pv *PgVector) GetTableName() string {
	return pv.tableName
}
