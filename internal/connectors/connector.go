package connectors

import "context"

// SearchRequest describes a normalized vector search operation that every
// database connector must understand. It intentionally hides database-specific
// parameter names so Milvus, pgvector, and future connectors can feed comparable
// retrieval results into the fingerprint engine.
type SearchRequest struct {
	// Collection identifies the source collection, table, index, or namespace.
	Collection string
	// QueryVector contains the query embedding used for nearest-neighbor search.
	QueryVector []float64
	// TopK is the business-visible result count used for ordinary topK comparison.
	TopK int
	// ExpandK is the larger result count used to observe topK boundary candidates.
	ExpandK int
	// Filters contains normalized metadata constraints such as tenant or document type.
	Filters map[string]string
	// Params contains connector-specific search parameters after explicit opt-in.
	Params map[string]string
}

// SearchHit represents one normalized nearest-neighbor result returned by a
// connector. The fingerprint engine consumes this structure to calculate stable
// neighbors, boundary candidates, score curves, and ranking differences.
type SearchHit struct {
	// ID is the stable vector or document identifier used to compare source and target results.
	ID string
	// Rank is the one-based rank assigned by the source or target vector database.
	Rank int
	// Score is the normalized similarity score or distance-derived score for comparison.
	Score float64
	// Metadata contains optional normalized fields returned with the vector hit.
	Metadata map[string]string
}

// SearchResponse contains normalized search hits for a single query. It is kept
// deliberately small so connector implementations can stream or batch responses
// later without leaking database-specific SDK objects into core packages.
type SearchResponse struct {
	// Hits contains ranked results ordered by ascending rank.
	Hits []SearchHit
}

// Connector defines the enterprise boundary between vdb-guardian and concrete
// vector databases. Implementations must honor context cancellation and return
// normalized SearchResponse values suitable for retrieval behavior comparison.
type Connector interface {
	// Name returns a stable connector identifier for logs, configuration, and reports.
	Name() string
	// Connect initializes the connector and validates that the target database is reachable.
	Connect(ctx context.Context) error
	// Count returns the number of records in a collection or table for migration completeness checks.
	Count(ctx context.Context, collection string) (int64, error)
	// Search executes a normalized vector search request and returns comparable ranked hits.
	Search(ctx context.Context, req SearchRequest) (SearchResponse, error)
	// Close releases any network connections, pools, or local resources held by the connector.
	Close() error
}
