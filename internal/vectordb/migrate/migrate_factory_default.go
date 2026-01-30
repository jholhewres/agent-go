package migrate

import (
    "fmt"

    "github.com/jholhewres/agent-go/pkg/agentgo/vectordb"
    "github.com/jholhewres/agent-go/pkg/agentgo/vectordb/chromadb"
)

// defaultFactory provides Chroma provider by default; Redis is optional and disabled here.
func defaultFactory(opts Options) (vectordb.VectorDB, error) {
    switch opts.Provider {
    case "chroma", "chromadb", "chromad b":
        cfg := chromadb.Config{BaseURL: opts.ChromaBaseURL, CollectionName: opts.Collection, Database: opts.ChromaDatabase, Tenant: opts.ChromaTenant}
        return chromadb.New(cfg)
    case "redis":
        return nil, fmt.Errorf("redis vectordb provider not enabled (build with -tags redis)")
    default:
        return nil, fmt.Errorf("unsupported provider: %s", opts.Provider)
    }
}

