package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/jholhewres/agent-go/pkg/agentgo/types"
	"github.com/jholhewres/agent-go/pkg/agentgo/vectordb"
)

// SearchableMemory extends Memory interface with search capabilities
// SearchableMemory 扩展 Memory 接口，添加搜索功能
type SearchableMemory interface {
	Memory

	// Search searches for relevant messages using hybrid vector + text search
	// Search 使用混合向量+文本搜索查找相关消息
	// Returns messages sorted by relevance score (descending)
	// 返回按相关性分数排序的消息（降序）
	Search(ctx context.Context, query string, limit int, userID ...string) ([]SearchResult, error)

	// SearchWithOptions searches with advanced options
	// SearchWithOptions 使用高级选项搜索
	SearchWithOptions(ctx context.Context, query string, options SearchOptions, userID ...string) ([]SearchResult, error)
}

// SearchOptions configures advanced search behavior
// SearchOptions 配置高级搜索行为
type SearchOptions struct {
	// Limit is the maximum number of results to return
	// Limit 是返回的最大结果数
	Limit int

	// MinScore is the minimum relevance score (0-1)
	// MinScore 是最小相关性分数（0-1）
	MinScore float64

	// VectorWeight is the weight for vector similarity (default: 0.7)
	// VectorWeight 是向量相似度的权重（默认：0.7）
	VectorWeight float64

	// TextWeight is the weight for text similarity (default: 0.3)
	// TextWeight 是文本相似度的权重（默认：0.3）
	TextWeight float64

	// FilterByRole limits search to specific message roles
	// FilterByRole 将搜索限制为特定消息角色
	FilterByRole []types.Role

	// IncludeRecent indicates whether to always include recent messages
	// IncludeRecent 表示是否始终包含最近的消息
	IncludeRecent bool

	// RecentCount is the number of recent messages to include
	// RecentCount 是要包含的最近消息数
	RecentCount int
}

// SearchResult represents a memory search result with relevance score
// SearchResult 表示带有相关性分数的内存搜索结果
type SearchResult struct {
	Message     *types.Message `json:"message"`
	Score       float64        `json:"score"`       // Combined relevance score (0-1)
	VectorScore float64        `json:"vector_score"` // Vector similarity score
	TextScore   float64        `json:"text_score"`   // Text similarity score
	Source      string         `json:"source"`       // "short_term" or "long_term"
}

// HybridMemoryConfig configures the hybrid memory behavior
// HybridMemoryConfig 配置混合内存行为
type HybridMemoryConfig struct {
	// MaxShortTermMessages is the number of messages to keep in short-term memory
	// MaxShortTermMessages 是短期内存中保留的消息数
	MaxShortTermMessages int

	// LongTermThreshold is when to move messages to long-term (0 = disabled)
	// LongTermThreshold 是何时将消息移动到长期存储（0 = 禁用）
	LongTermThreshold int

	// VectorDB is the vector database for long-term storage
	// VectorDB 是用于长期存储的向量数据库
	VectorDB vectordb.VectorDB

	// Embedder generates embeddings for vector search
	// Embedder 生成向量搜索的嵌入
	Embedder vectordb.EmbeddingFunction

	// CollectionName is the vector DB collection name
	// CollectionName 是向量数据库集合名称
	CollectionName string

	// Default search options
	// 默认搜索选项
	DefaultVectorWeight float64
	DefaultTextWeight   float64
	DefaultMinScore     float64
}

// HybridMemory combines short-term (InMemory) and long-term (VectorDB) storage
// HybridMemory 结合了短期（InMemory）和长期（VectorDB）存储
type HybridMemory struct {
	shortTerm *InMemory
	longTerm  vectordb.VectorDB
	embedder  vectordb.EmbeddingFunction
	config    HybridMemoryConfig
	mu        sync.RWMutex
}

// NewHybridMemory creates a new hybrid memory instance
// NewHybridMemory 创建一个新的混合内存实例
func NewHybridMemory(config HybridMemoryConfig) (*HybridMemory, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid hybrid memory config: %w", err)
	}

	// Set defaults
	if config.MaxShortTermMessages <= 0 {
		config.MaxShortTermMessages = 100
	}
	if config.DefaultVectorWeight <= 0 {
		config.DefaultVectorWeight = 0.7
	}
	if config.DefaultTextWeight <= 0 {
		config.DefaultTextWeight = 0.3
	}
	if config.DefaultMinScore <= 0 {
		config.DefaultMinScore = 0.1
	}
	if config.CollectionName == "" {
		config.CollectionName = "agent_memory"
	}

	// Create collection if needed
	ctx := context.Background()
	if err := config.VectorDB.CreateCollection(ctx, config.CollectionName, nil); err != nil {
		// Collection might already exist, log but continue
		// 集合可能已存在，记录日志但继续
	}

	return &HybridMemory{
		shortTerm: NewInMemory(config.MaxShortTermMessages),
		longTerm:  config.VectorDB,
		embedder:  config.Embedder,
		config:    config,
	}, nil
}

// validateConfig validates the hybrid memory configuration
func validateConfig(config HybridMemoryConfig) error {
	if config.VectorDB == nil {
		return fmt.Errorf("VectorDB is required")
	}
	if config.Embedder == nil {
		return fmt.Errorf("Embedder is required")
	}
	return nil
}

// Add appends a message to memory
// Add 将消息添加到内存
func (m *HybridMemory) Add(message *types.Message, userID ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid := getUserID(userID...)

	// Add to short-term memory
	// 添加到短期内存
	m.shortTerm.Add(message, uid)

	// Check if we need to move old messages to long-term
	// 检查是否需要将旧消息移动到长期存储
	if m.config.LongTermThreshold > 0 {
		m.moveToLongTerm(uid)
	}
}

// moveToLongTerm moves old messages from short-term to long-term storage
func (m *HybridMemory) moveToLongTerm(userID string) {
	allMessages := m.shortTerm.GetMessages(userID)

	// Keep only recent messages in short-term
	// 在短期内存中仅保留最近的消息
	if len(allMessages) <= m.config.LongTermThreshold {
		return
	}

	// Messages to move (exclude system messages and recent ones)
	// 要移动的消息（排除系统消息和最近的消息）
	messagesToMove := allMessages[:len(allMessages)-m.config.LongTermThreshold]
	var toMove []*types.Message
	for _, msg := range messagesToMove {
		// Don't move system messages
		// 不移动系统消息
		if msg.Role != types.RoleSystem {
			toMove = append(toMove, msg)
		}
	}

	if len(toMove) == 0 {
		return
	}

	// Generate embeddings and store in vector DB
	// 生成嵌入并存储在向量数据库中
	ctx := context.Background()
	documents := make([]vectordb.Document, 0, len(toMove))

	for _, msg := range toMove {
		// Skip if already in long-term (check by ID)
		// 如果已在长期存储中则跳过（通过 ID 检查）
		existing, err := m.longTerm.Get(ctx, []string{msg.ID})
		if err == nil && len(existing) > 0 {
			continue
		}

		// Generate embedding
		// 生成嵌入
		embedding, err := m.embedder.EmbedSingle(ctx, msg.Content)
		if err != nil {
			// Log error but continue
			// 记录错误但继续
			continue
		}

		documents = append(documents, vectordb.Document{
			ID:        msg.ID,
			Content:   msg.Content,
			Embedding: embedding,
			Metadata: map[string]interface{}{
				"user_id":   userID,
				"role":      string(msg.Role),
				"timestamp": msg.Metadata, // Assuming timestamp in metadata
			},
		})
	}

	if len(documents) > 0 {
		_ = m.longTerm.Add(ctx, documents)
	}
}

// GetMessages returns all messages for a specific user
// GetMessages 返回特定用户的所有消息
func (m *HybridMemory) GetMessages(userID ...string) []*types.Message {
	return m.shortTerm.GetMessages(userID...)
}

// Clear removes all messages for a specific user
// Clear 删除特定用户的所有消息
func (m *HybridMemory) Clear(userID ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid := getUserID(userID...)

	// Clear short-term
	// 清除短期内存
	m.shortTerm.Clear(uid)

	// Clear long-term for this user (delete by filter)
	// 清除此用户的长期存储（通过过滤器删除）
	ctx := context.Background()
	// Note: This requires vector DB to support delete by filter
	// Many vector DBs don't support this efficiently, so we skip for now
	// 注意：这需要向量数据库支持按过滤器删除
	// 许多向量数据库不高效支持此功能，所以我们暂时跳过
	_ = ctx
}

// Size returns the number of messages in short-term memory
// Size 返回短期内存中的消息数
func (m *HybridMemory) Size(userID ...string) int {
	return m.shortTerm.Size(userID...)
}

// Search performs hybrid search combining vector and text similarity
// Search 执行混合搜索，结合向量和文本相似度
func (m *HybridMemory) Search(ctx context.Context, query string, limit int, userID ...string) ([]SearchResult, error) {
	options := SearchOptions{
		Limit:        limit,
		VectorWeight: m.config.DefaultVectorWeight,
		TextWeight:   m.config.DefaultTextWeight,
		MinScore:     m.config.DefaultMinScore,
	}
	return m.SearchWithOptions(ctx, query, options, userID...)
}

// SearchWithOptions performs advanced hybrid search
// SearchWithOptions 执行高级混合搜索
func (m *HybridMemory) SearchWithOptions(ctx context.Context, query string, options SearchOptions, userID ...string) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uid := getUserID(userID...)

	// Set defaults
	if options.Limit <= 0 {
		options.Limit = 5
	}
	if options.VectorWeight <= 0 {
		options.VectorWeight = m.config.DefaultVectorWeight
	}
	if options.TextWeight <= 0 {
		options.TextWeight = m.config.DefaultTextWeight
	}
	if options.MinScore < 0 {
		options.MinScore = m.config.DefaultMinScore
	}

	// Normalize weights
	totalWeight := options.VectorWeight + options.TextWeight
	if totalWeight > 0 {
		options.VectorWeight /= totalWeight
		options.TextWeight /= totalWeight
	}

	// Collect all results
	resultMap := make(map[string]*SearchResult)

	// 1. Search short-term memory (text-based)
	// 1. 搜索短期内存（基于文本）
	shortTermResults := m.searchShortTerm(query, uid, options)
	for i := range shortTermResults {
		if _, exists := resultMap[shortTermResults[i].Message.ID]; !exists {
			resultMap[shortTermResults[i].Message.ID] = &shortTermResults[i]
		}
	}

	// 2. Search long-term memory (vector-based)
	// 2. 搜索长期内存（基于向量）
	longTermResults, err := m.searchLongTerm(ctx, query, uid, options)
	if err == nil {
		for i := range longTermResults {
			if existing, exists := resultMap[longTermResults[i].Message.ID]; exists {
				// Merge scores: take the maximum
				// 合并分数：取最大值
				if longTermResults[i].VectorScore > existing.VectorScore {
					existing.VectorScore = longTermResults[i].VectorScore
					existing.Score = existing.TextScore*options.TextWeight + existing.VectorScore*options.VectorWeight
				}
			} else {
				resultMap[longTermResults[i].Message.ID] = &longTermResults[i]
			}
		}
	}

	// 3. Filter by role if specified
	// 3. 如果指定，按角色过滤
	var results []SearchResult
	for _, result := range resultMap {
		if result.Score < options.MinScore {
			continue
		}
		if len(options.FilterByRole) > 0 {
			roleMatch := false
			for _, role := range options.FilterByRole {
				if result.Message.Role == role {
					roleMatch = true
					break
				}
			}
			if !roleMatch {
				continue
			}
		}
		results = append(results, *result)
	}

	// 4. Sort by score (descending)
	// 4. 按分数排序（降序）
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 5. Apply limit
	// 5. 应用限制
	if len(results) > options.Limit {
		results = results[:options.Limit]
	}

	return results, nil
}

// searchShortTerm performs text-based search on short-term memory
func (m *HybridMemory) searchShortTerm(query string, userID string, options SearchOptions) []SearchResult {
	messages := m.shortTerm.GetMessages(userID)
	queryLower := strings.ToLower(query)

	var results []SearchResult
	for _, msg := range messages {
		// Calculate text similarity using simple keyword matching
		// 使用简单的关键词匹配计算文本相似度
		textScore := calculateTextSimilarity(queryLower, strings.ToLower(msg.Content))

		if textScore > 0 {
			results = append(results, SearchResult{
				Message:     msg,
				Score:       textScore * options.TextWeight, // Will be combined later
				VectorScore: 0,
				TextScore:   textScore,
				Source:      "short_term",
			})
		}
	}

	return results
}

// searchLongTerm performs vector-based search on long-term memory
func (m *HybridMemory) searchLongTerm(ctx context.Context, query string, userID string, options SearchOptions) ([]SearchResult, error) {
	// Query vector DB with user filter
	// 使用用户过滤器查询向量数据库
	filter := map[string]interface{}{"user_id": userID}
	vectorResults, err := m.longTerm.Query(ctx, query, options.Limit*2, filter) // Get more candidates
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, vr := range vectorResults {
		// Reconstruct message from search result
		// 从搜索结果重建消息
		role := types.RoleAssistant
		if r, ok := vr.Metadata["role"].(string); ok {
			role = types.Role(r)
		}

		msg := &types.Message{
			ID:      vr.ID,
			Content: vr.Content,
			Role:    role,
		}

		// VectorDB scores are typically 0-1 (cosine similarity)
		// VectorDB 分数通常是 0-1（余弦相似度）
		vectorScore := float64(vr.Score)

		results = append(results, SearchResult{
			Message:     msg,
			Score:       vectorScore * options.VectorWeight, // Will be combined later
			VectorScore: vectorScore,
			TextScore:   0,
			Source:      "long_term",
		})
	}

	return results, nil
}

// calculateTextSimilarity calculates simple text similarity score
// Uses word overlap as a simple metric (more sophisticated methods could be added)
func calculateTextSimilarity(query, text string) float64 {
	queryWords := strings.Fields(query)
	if len(queryWords) == 0 {
		return 0
	}

	textWords := strings.Fields(text)
	textWordSet := make(map[string]bool)
	for _, w := range textWords {
		textWordSet[w] = true
	}

	matches := 0
	for _, qw := range queryWords {
		if textWordSet[qw] {
			matches++
		}
	}

	// Jaccard-like similarity
	// 类似 Jaccard 的相似度
	return float64(matches) / float64(len(queryWords))
}
