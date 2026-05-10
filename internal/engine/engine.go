package engine

import "context"

// CompareInput is the stable Go-side request passed to a fingerprint engine. In
// the first enterprise scaffold this type only carries artifact locations; later
// implementations can use the same boundary for Python subprocess, gRPC, or a
// native Go engine.
type CompareInput struct {
	// JobID links the comparison request to a durable verification job.
	JobID string
	// SourceFingerprintPath points to the artifact containing source retrieval behavior fingerprints.
	SourceFingerprintPath string
	// TargetFingerprintPath points to the artifact containing target retrieval behavior fingerprints.
	TargetFingerprintPath string
}

// MetricSummary contains the primary retrieval behavior consistency metrics that
// reports, APIs, and future CI gates can consume without understanding engine
// internals.
type MetricSummary struct {
	// FingerprintDistance is the normalized overall distance between source and target fingerprints.
	FingerprintDistance float64
	// StableNeighborDistance is the average Jaccard distance between stable-neighbor sets.
	StableNeighborDistance float64
	// BoundaryCandidateDistance is the average Jaccard distance between boundary-candidate sets.
	BoundaryCandidateDistance float64
	// BoundaryFlipRate measures how often topK boundary candidates enter or leave visible results.
	BoundaryFlipRate float64
	// MatchedQueryCount is the number of query IDs found in both source and target artifacts.
	MatchedQueryCount int
	// MissingSourceQueryCount is the number of target query IDs missing from the source artifact.
	MissingSourceQueryCount int
	// MissingTargetQueryCount is the number of source query IDs missing from the target artifact.
	MissingTargetQueryCount int
}

// CompareOutput is the normalized response returned by any fingerprint engine
// implementation. Keeping the response small gives the Go control plane a stable
// contract while detailed artifacts remain in the artifact store.
type CompareOutput struct {
	// JobID identifies the verification job associated with this comparison result.
	JobID string
	// ConsistencyScore is a normalized score in [0, 1], where higher means more consistent.
	ConsistencyScore float64
	// Metrics contains the main decomposed fingerprint comparison values.
	Metrics MetricSummary
}

// Engine defines the boundary between Go orchestration and retrieval behavior
// fingerprint algorithms. Implementations must honor context cancellation and
// return deterministic output for a given set of source and target artifacts.
type Engine interface {
	// Compare compares source and target retrieval behavior fingerprints and returns consistency metrics.
	Compare(ctx context.Context, input CompareInput) (CompareOutput, error)
}
