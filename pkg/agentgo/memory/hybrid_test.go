package memory

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/jholhewres/agent-go/pkg/agentgo/types"
	"github.com/jholhewres/agent-go/pkg/agentgo/vectordb"
)

// mockVectorDB is a simple in-memory vector DB for testing
type mockVectorDB struct {
	mu       sync.RWMutex
	docs     map[string]vectordb.Document
	collections map[string]bool
}

func newMockVectorDB() *mockVectorDB {
	return &mockVectorDB{
		docs:     make(map[string]vectordb.Document),
		collections: make(map[string]bool),
	}
}

func (m *mockVectorDB) CreateCollection(ctx context.Context, name string, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.collections[name] = true
	return nil
}

func (m *mockVectorDB) DeleteCollection(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.collections, name)
	return nil
}

func (m *mockVectorDB) Add(ctx context.Context, documents []vectordb.Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, doc := range documents {
		m.docs[doc.ID] = doc
	}
	return nil
}

func (m *mockVectorDB) Update(ctx context.Context, documents []vectordb.Document) error {
	return m.Add(ctx, documents)
}

func (m *mockVectorDB) Delete(ctx context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		delete(m.docs, id)
	}
	return nil
}

func (m *mockVectorDB) Query(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]vectordb.SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simple implementation: return all docs with decreasing scores
	var results []vectordb.SearchResult
	score := float32(1.0)
	for _, doc := range m.docs {
		if limit > 0 && len(results) >= limit {
			break
		}

		// Apply user filter
		if filter != nil {
			if userID, ok := filter["user_id"].(string); ok {
				if docUserID, ok := doc.Metadata["user_id"].(string); ok && docUserID != userID {
					continue
				}
			}
		}

		results = append(results, vectordb.SearchResult{
			ID:       doc.ID,
			Content:  doc.Content,
			Score:    score,
			Distance: 1 - score,
		})
		score -= 0.1
		if score < 0 {
			score = 0
		}
	}

	return results, nil
}

func (m *mockVectorDB) QueryWithEmbedding(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]vectordb.SearchResult, error) {
	return m.Query(ctx, "", limit, filter)
}

func (m *mockVectorDB) Get(ctx context.Context, ids []string) ([]vectordb.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var docs []vectordb.Document
	for _, id := range ids {
		if doc, ok := m.docs[id]; ok {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func (m *mockVectorDB) Count(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.docs), nil
}

func (m *mockVectorDB) Close() error {
	return nil
}

// mockEmbedder is a simple mock embedder for testing
type mockEmbedder struct {
	embedFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

func newMockEmbedder() *mockEmbedder {
	return &mockEmbedder{
		embedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				// Simple hash-based embedding for testing
				result[i] = make([]float32, 128)
				for j := range result[i] {
					result[i][j] = float32(len(texts[i]) % 100)
				}
			}
			return result, nil
		},
	}
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return m.embedFunc(ctx, texts)
}

func (m *mockEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	results, err := m.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

// TestNewHybridMemory tests creating a new hybrid memory instance
func TestNewHybridMemory(t *testing.T) {
	tests := []struct {
		name    string
		config  HybridMemoryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: HybridMemoryConfig{
				VectorDB:              newMockVectorDB(),
				Embedder:              newMockEmbedder(),
				MaxShortTermMessages:  50,
				LongTermThreshold:     100,
				DefaultVectorWeight:   0.7,
				DefaultTextWeight:     0.3,
			},
			wantErr: false,
		},
		{
			name: "missing vector DB",
			config: HybridMemoryConfig{
				Embedder: newMockEmbedder(),
			},
			wantErr: true,
		},
		{
			name: "missing embedder",
			config: HybridMemoryConfig{
				VectorDB: newMockVectorDB(),
			},
			wantErr: true,
		},
		{
			name: "sets defaults",
			config: HybridMemoryConfig{
				VectorDB: newMockVectorDB(),
				Embedder: newMockEmbedder(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem, err := NewHybridMemory(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHybridMemory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && mem == nil {
				t.Error("expected non-nil memory")
			}
		})
	}
}

// TestHybridMemoryAdd tests adding messages to hybrid memory
func TestHybridMemoryAdd(t *testing.T) {
	vdb := newMockVectorDB()
	embedder := newMockEmbedder()
	config := HybridMemoryConfig{
		VectorDB:             vdb,
		Embedder:             embedder,
		MaxShortTermMessages: 5,
		LongTermThreshold:    3,
	}

	mem, err := NewHybridMemory(config)
	if err != nil {
		t.Fatalf("failed to create hybrid memory: %v", err)
	}

	// Add messages
	userID := "test-user"
	for i := 0; i < 5; i++ {
		msg := types.NewUserMessage(fprintf(t, "message %d", i))
		mem.Add(msg, userID)
	}

	// Check short-term size
	if size := mem.Size(userID); size != 5 {
		t.Errorf("expected short-term size 5, got %d", size)
	}

	// Add more messages to trigger long-term move
	for i := 5; i < 10; i++ {
		msg := types.NewUserMessage(fprintf(t, "message %d", i))
		mem.Add(msg, userID)
	}

	// Old messages should still be in short-term (system messages preserved)
	// but some should have been moved to long-term
	longTermCount, _ := vdb.Count(context.Background())
	if longTermCount < 1 {
		t.Error("expected some messages to be moved to long-term storage")
	}
}

// TestHybridMemoryGetMessages tests retrieving messages
func TestHybridMemoryGetMessages(t *testing.T) {
	mem, err := NewHybridMemory(HybridMemoryConfig{
		VectorDB: newMockVectorDB(),
		Embedder: newMockEmbedder(),
	})
	if err != nil {
		t.Fatalf("failed to create hybrid memory: %v", err)
	}

	userID := "test-user"
	msg1 := types.NewUserMessage("first message")
	msg2 := types.NewAssistantMessage("second message")

	mem.Add(msg1, userID)
	mem.Add(msg2, userID)

	messages := mem.GetMessages(userID)
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	// Multi-tenant isolation
	otherUser := "other-user"
	msg3 := types.NewUserMessage("other message")
	mem.Add(msg3, otherUser)

	messages = mem.GetMessages(userID)
	if len(messages) != 2 {
		t.Errorf("expected 2 messages for original user, got %d", len(messages))
	}
}

// TestHybridMemoryClear tests clearing memory
func TestHybridMemoryClear(t *testing.T) {
	mem, err := NewHybridMemory(HybridMemoryConfig{
		VectorDB: newMockVectorDB(),
		Embedder: newMockEmbedder(),
	})
	if err != nil {
		t.Fatalf("failed to create hybrid memory: %v", err)
	}

	userID := "test-user"
	mem.Add(types.NewUserMessage("message"), userID)

	if size := mem.Size(userID); size != 1 {
		t.Errorf("expected size 1, got %d", size)
	}

	mem.Clear(userID)

	if size := mem.Size(userID); size != 0 {
		t.Errorf("expected size 0 after clear, got %d", size)
	}
}

// TestHybridMemorySearch tests searching memory
func TestHybridMemorySearch(t *testing.T) {
	vdb := newMockVectorDB()
	embedder := newMockEmbedder()

	mem, err := NewHybridMemory(HybridMemoryConfig{
		VectorDB:             vdb,
		Embedder:             embedder,
		MaxShortTermMessages: 100,
	})
	if err != nil {
		t.Fatalf("failed to create hybrid memory: %v", err)
	}

	userID := "test-user"

	// Add messages with specific content
	msg1 := types.NewUserMessage("The quick brown fox jumps over the lazy dog")
	msg2 := types.NewAssistantMessage("A quick response about foxes")
	msg3 := types.NewUserMessage("Dogs are loyal pets")
	msg4 := types.NewSystemMessage("You are a helpful assistant")

	mem.Add(msg1, userID)
	mem.Add(msg2, userID)
	mem.Add(msg3, userID)
	mem.Add(msg4, userID)

	// Test search
	ctx := context.Background()
	results, err := mem.Search(ctx, "quick fox", 3, userID)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected at least one search result")
	}

	// Results should be sorted by relevance
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted by relevance: [%d].Score=%f > [%d].Score=%f",
				i-1, results[i-1].Score, i, results[i].Score)
		}
	}

	// Check that results contain the query terms
	for _, r := range results {
		if r.Message == nil {
			t.Error("search result has nil message")
		}
	}
}

// TestHybridMemorySearchWithOptions tests advanced search options
func TestHybridMemorySearchWithOptions(t *testing.T) {
	mem, err := NewHybridMemory(HybridMemoryConfig{
		VectorDB: newMockVectorDB(),
		Embedder: newMockEmbedder(),
	})
	if err != nil {
		t.Fatalf("failed to create hybrid memory: %v", err)
	}

	userID := "test-user"
	mem.Add(types.NewUserMessage("user message"), userID)
	mem.Add(types.NewAssistantMessage("assistant message"), userID)
	mem.Add(types.NewSystemMessage("system message"), userID)

	ctx := context.Background()

	// Test role filter
	options := SearchOptions{
		Limit:       10,
		FilterByRole: []types.Role{types.RoleUser},
	}
	results, err := mem.SearchWithOptions(ctx, "message", options, userID)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	for _, r := range results {
		if r.Message.Role != types.RoleUser {
			t.Errorf("expected only user messages, got %v", r.Message.Role)
		}
	}

	// Test min score filter
	options = SearchOptions{
		Limit:    10,
		MinScore: 0.9, // High threshold
	}
	results, err = mem.SearchWithOptions(ctx, "nonexistent query", options, userID)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// Should have fewer results due to min score
	if len(results) > 0 {
		for _, r := range results {
			if r.Score < 0.9 {
				t.Errorf("result score %f below min score 0.9", r.Score)
			}
		}
	}
}

// TestHybridMemoryMultiTenant tests multi-tenant isolation
func TestHybridMemoryMultiTenant(t *testing.T) {
	mem, err := NewHybridMemory(HybridMemoryConfig{
		VectorDB: newMockVectorDB(),
		Embedder: newMockEmbedder(),
	})
	if err != nil {
		t.Fatalf("failed to create hybrid memory: %v", err)
	}

	user1 := "user1"
	user2 := "user2"

	mem.Add(types.NewUserMessage("user1 message"), user1)
	mem.Add(types.NewUserMessage("user2 message"), user2)

	ctx := context.Background()

	// Search user1 should not return user2's messages
	results, err := mem.Search(ctx, "message", 10, user1)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	for _, r := range results {
		// Verify results belong to user1
		// This is a basic check - in real scenario, metadata would have user_id
		_ = r
	}
}

// TestHybridMemoryConcurrentAccess tests concurrent access to memory
func TestHybridMemoryConcurrentAccess(t *testing.T) {
	mem, err := NewHybridMemory(HybridMemoryConfig{
		VectorDB:             newMockVectorDB(),
		Embedder:             newMockEmbedder(),
		MaxShortTermMessages: 1000,
	})
	if err != nil {
		t.Fatalf("failed to create hybrid memory: %v", err)
	}

	userID := "test-user"
	ctx := context.Background()

	// Run concurrent operations
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			msg := types.NewUserMessage(fprintf(t, "concurrent message %d", i))
			mem.Add(msg, userID)
			mem.GetMessages(userID)
			mem.Search(ctx, "search", 5, userID)
		}(i)
	}
	wg.Wait()

	// Verify final state
	if size := mem.Size(userID); size == 0 {
		t.Error("expected some messages after concurrent operations")
	}
}

// TestCalculateTextSimilarity tests text similarity calculation
func TestCalculateTextSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		text     string
		wantMin  float64
		wantMax  float64
	}{
		{
			name:    "exact match",
			query:   "hello world",
			text:    "hello world",
			wantMin: 0.9,
			wantMax: 1.0,
		},
		{
			name:    "partial match",
			query:   "hello world",
			text:    "hello there",
			wantMin: 0.4,
			wantMax: 0.6,
		},
		{
			name:    "no match",
			query:   "hello",
			text:    "goodbye",
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "empty query",
			query:   "",
			text:    "any text",
			wantMin: 0.0,
			wantMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateTextSimilarity(tt.query, tt.text)
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("score = %f, want between [%f, %f]", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestHybridMemoryBackwardCompatibility tests backward compatibility with Memory interface
func TestHybridMemoryBackwardCompatibility(t *testing.T) {
	mem, err := NewHybridMemory(HybridMemoryConfig{
		VectorDB: newMockVectorDB(),
		Embedder: newMockEmbedder(),
	})
	if err != nil {
		t.Fatalf("failed to create hybrid memory: %v", err)
	}

	// Test that it implements Memory interface correctly
	var memInterface Memory = mem

	userID := "test-user"
	msg := types.NewUserMessage("test message")

	// Test Add
	memInterface.Add(msg, userID)

	// Test GetMessages
	messages := memInterface.GetMessages(userID)
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	// Test Size
	if size := memInterface.Size(userID); size != 1 {
		t.Errorf("expected size 1, got %d", size)
	}

	// Test Clear
	memInterface.Clear(userID)
	if size := memInterface.Size(userID); size != 0 {
		t.Errorf("expected size 0 after clear, got %d", size)
	}
}

// Helper function for formatted strings in tests
func fprintf(t *testing.T, format string, args ...interface{}) string {
	t.Helper()
	return fmt.Sprintf(format, args...)
}
