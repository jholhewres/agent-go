//go:build redis

package migrate

import (
	"fmt"

	"github.com/jholhewres/agent-go/pkg/agentgo/vectordb"
	"github.com/jholhewres/agent-go/pkg/agentgo/vectordb/chromadb"
	"github.com/jholhewres/agent-go/pkg/agentgo/vectordb/redisdb"
)

// defaultFactory compiled with redis tag supports both chroma and redis providers.
func defaultFactory(opts Options) (vectordb.VectorDB, error) {
	switch opts.Provider {
	case "chroma", "chromadb", "chromad b":
		cfg := chromadb.Config{BaseURL: opts.ChromaBaseURL, CollectionName: opts.Collection, Database: opts.ChromaDatabase, Tenant: opts.ChromaTenant}
		return chromadb.New(cfg)
	case "redis":
		cfg := redisdb.Config{Addr: opts.ChromaBaseURL /* reusing field for addr if provided */, CollectionName: opts.Collection}
		return redisdb.New(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", opts.Provider)
	}
}
