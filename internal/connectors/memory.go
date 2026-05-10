package connectors

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// MemoryConnector is a deterministic in-memory connector for local verification
// and tests. It implements the Connector contract without contacting a real
// vector database, which lets the control plane exercise search normalization,
// fingerprint artifact building, and verification orchestration before Milvus or
// pgvector connectors are available.
type MemoryConnector struct {
	// name is the stable connector identifier used in logs, tests, and reports.
	name string
	// results maps a query identifier to precomputed ranked hits.
	results map[string][]SearchHit
}

// NewMemoryConnector creates a connector backed by precomputed query results.
// The input map and slices are deep-copied so later caller mutations cannot
// change connector behavior during deterministic local verification tests.
func NewMemoryConnector(name string, results map[string][]SearchHit) MemoryConnector {
	if name == "" {
		name = "memory"
	}
	copied := make(map[string][]SearchHit, len(results))
	for queryID, hits := range results {
		copied[queryID] = cloneAndSortHits(hits)
	}
	return MemoryConnector{name: name, results: copied}
}

// Name returns the stable connector identifier configured at construction time.
func (c MemoryConnector) Name() string {
	return c.name
}

// Connect validates the in-memory connector context. It performs no network I/O
// because all search results are already loaded in memory.
func (c MemoryConnector) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

// Count returns the number of precomputed hits for the provided query key. In
// the memory connector, the collection argument represents the query identifier
// so tests can use the same Connector interface without database-specific state.
func (c MemoryConnector) Count(ctx context.Context, collection string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	hits, ok := c.results[collection]
	if !ok {
		return 0, fmt.Errorf("memory connector query %q not found", collection)
	}
	return int64(len(hits)), nil
}

// Search returns deterministic ranked hits for the request collection key. The
// collection field acts as a query identifier until real connectors provide
// query-vector based search; returned slices are copied so callers cannot mutate
// connector state.
func (c MemoryConnector) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if err := ctx.Err(); err != nil {
		return SearchResponse{}, err
	}
	if req.TopK <= 0 {
		return SearchResponse{}, errors.New("top_k must be greater than zero")
	}
	if req.ExpandK <= 0 {
		return SearchResponse{}, errors.New("expand_k must be greater than zero")
	}
	if req.ExpandK < req.TopK {
		return SearchResponse{}, errors.New("expand_k must be greater than or equal to top_k")
	}
	if req.Collection == "" {
		return SearchResponse{}, errors.New("memory connector collection query key must not be empty")
	}

	hits, ok := c.results[req.Collection]
	if !ok {
		return SearchResponse{}, fmt.Errorf("memory connector query %q not found", req.Collection)
	}
	if len(hits) < req.ExpandK {
		return SearchResponse{}, fmt.Errorf("memory connector query %q must contain at least expand_k hits", req.Collection)
	}
	return SearchResponse{Hits: cloneHits(hits[:req.ExpandK])}, nil
}

// Close releases memory connector resources. It is currently a no-op because no
// external handles are acquired, but it preserves the Connector lifecycle shape.
func (c MemoryConnector) Close() error {
	return nil
}

func cloneAndSortHits(hits []SearchHit) []SearchHit {
	copied := cloneHits(hits)
	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].Rank < copied[j].Rank
	})
	return copied
}

func cloneHits(hits []SearchHit) []SearchHit {
	copied := make([]SearchHit, len(hits))
	for i, hit := range hits {
		copied[i] = hit
		if hit.Metadata != nil {
			copied[i].Metadata = make(map[string]string, len(hit.Metadata))
			for key, value := range hit.Metadata {
				copied[i].Metadata[key] = value
			}
		}
	}
	return copied
}
