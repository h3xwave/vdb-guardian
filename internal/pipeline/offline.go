package pipeline

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/huxinweidev-cloud/vdb-guardian/internal/connectors"
	"github.com/huxinweidev-cloud/vdb-guardian/internal/fingerprints"
	"github.com/huxinweidev-cloud/vdb-guardian/internal/jobs"
)

// OfflinePipeline orchestrates a database-free local verification flow. It reads
// deterministic search results from source and target connectors, builds Python-
// compatible fingerprint artifacts, and delegates artifact comparison to the
// local verification runner.
type OfflinePipeline struct {
	// SourceConnector provides normalized source-side search results.
	SourceConnector connectors.Connector
	// TargetConnector provides normalized target-side search results.
	TargetConnector connectors.Connector
	// Runner compares generated source and target fingerprint artifacts.
	Runner jobs.VerificationRunner
	// ArtifactDir is the local directory where fingerprints and result artifacts are written.
	ArtifactDir string
	// BuildOptions controls topK, stable-neighbor, and boundary-candidate derivation.
	BuildOptions fingerprints.BuildOptions
}

// OfflineRequest describes one local offline verification run. QueryIDs are used
// as connector collection keys by the memory connector and can later map to real
// query definitions when Milvus and pgvector connectors are available.
type OfflineRequest struct {
	// JobID identifies the local pipeline run and prefixes generated artifact files.
	JobID string
	// QueryIDs lists the verification queries to collect from both connectors.
	QueryIDs []string
	// TopK is the visible result count requested from connectors.
	TopK int
	// ExpandK is the expanded hit count requested so boundary candidates can be built.
	ExpandK int
}

// OfflineResult contains generated fingerprint artifact paths and the final
// verification result produced by the local runner.
type OfflineResult struct {
	// JobID identifies the completed offline pipeline run.
	JobID string
	// SourceFingerprintPath points to the generated source fingerprint artifact.
	SourceFingerprintPath string
	// TargetFingerprintPath points to the generated target fingerprint artifact.
	TargetFingerprintPath string
	// VerificationResult is the runner output generated from comparing the two artifacts.
	VerificationResult jobs.VerificationResult
}

// NewOfflinePipeline creates a local offline verification pipeline. Dependencies
// are injected so tests can use memory connectors and fake engines while future
// production wiring can provide concrete database connectors and PythonRunner.
func NewOfflinePipeline(
	source connectors.Connector,
	target connectors.Connector,
	runner jobs.VerificationRunner,
	artifactDir string,
	options fingerprints.BuildOptions,
) OfflinePipeline {
	return OfflinePipeline{
		SourceConnector: source,
		TargetConnector: target,
		Runner:          runner,
		ArtifactDir:     artifactDir,
		BuildOptions:    options,
	}
}

// Run executes the local offline pipeline from connector search through result
// artifact writing. It returns before writing success artifacts when validation,
// connector search, artifact construction, artifact writing, or engine comparison
// fails.
func (p OfflinePipeline) Run(ctx context.Context, request OfflineRequest) (OfflineResult, error) {
	if err := ctx.Err(); err != nil {
		return OfflineResult{}, err
	}
	if err := p.validate(request); err != nil {
		return OfflineResult{}, err
	}

	sourceResults, targetResults, err := p.collectSearchResults(ctx, request)
	if err != nil {
		return OfflineResult{}, err
	}

	sourceArtifact, err := fingerprints.BuildArtifact(sourceResults, p.BuildOptions)
	if err != nil {
		return OfflineResult{}, fmt.Errorf("build source fingerprint artifact: %w", err)
	}
	targetArtifact, err := fingerprints.BuildArtifact(targetResults, p.BuildOptions)
	if err != nil {
		return OfflineResult{}, fmt.Errorf("build target fingerprint artifact: %w", err)
	}

	sourcePath := filepath.Join(p.ArtifactDir, fmt.Sprintf("%s-source-fingerprint.json", request.JobID))
	targetPath := filepath.Join(p.ArtifactDir, fmt.Sprintf("%s-target-fingerprint.json", request.JobID))
	if err := fingerprints.WriteArtifact(sourcePath, sourceArtifact); err != nil {
		return OfflineResult{}, fmt.Errorf("write source fingerprint artifact: %w", err)
	}
	if err := fingerprints.WriteArtifact(targetPath, targetArtifact); err != nil {
		return OfflineResult{}, fmt.Errorf("write target fingerprint artifact: %w", err)
	}

	verificationResult, err := p.Runner.Run(ctx, jobs.VerificationRequest{
		JobID:                 request.JobID,
		SourceFingerprintPath: sourcePath,
		TargetFingerprintPath: targetPath,
	})
	if err != nil {
		return OfflineResult{}, fmt.Errorf("run verification runner: %w", err)
	}

	return OfflineResult{
		JobID:                 request.JobID,
		SourceFingerprintPath: sourcePath,
		TargetFingerprintPath: targetPath,
		VerificationResult:    verificationResult,
	}, nil
}

func (p OfflinePipeline) validate(request OfflineRequest) error {
	if p.SourceConnector == nil || p.TargetConnector == nil {
		return errors.New("offline pipeline source and target connectors must not be nil")
	}
	if p.Runner.Engine == nil {
		return errors.New("offline pipeline verification runner engine must not be nil")
	}
	if p.ArtifactDir == "" {
		return errors.New("offline pipeline artifact dir must not be empty")
	}
	if request.JobID == "" {
		return errors.New("offline request job id must not be empty")
	}
	if len(request.QueryIDs) == 0 {
		return errors.New("offline request query_ids must not be empty")
	}
	if request.TopK <= 0 {
		return errors.New("offline request top_k must be greater than zero")
	}
	if request.ExpandK < request.TopK {
		return errors.New("offline request expand_k must be greater than or equal to top_k")
	}
	return nil
}

func (p OfflinePipeline) collectSearchResults(
	ctx context.Context,
	request OfflineRequest,
) ([]fingerprints.SearchResult, []fingerprints.SearchResult, error) {
	sourceResults := make([]fingerprints.SearchResult, 0, len(request.QueryIDs))
	targetResults := make([]fingerprints.SearchResult, 0, len(request.QueryIDs))
	for _, queryID := range request.QueryIDs {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		sourceResponse, err := p.SourceConnector.Search(ctx, connectors.SearchRequest{
			Collection: queryID,
			TopK:       request.TopK,
			ExpandK:    request.ExpandK,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("source search query %q: %w", queryID, err)
		}
		targetResponse, err := p.TargetConnector.Search(ctx, connectors.SearchRequest{
			Collection: queryID,
			TopK:       request.TopK,
			ExpandK:    request.ExpandK,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("target search query %q: %w", queryID, err)
		}
		sourceResults = append(sourceResults, toFingerprintSearchResult(queryID, sourceResponse))
		targetResults = append(targetResults, toFingerprintSearchResult(queryID, targetResponse))
	}
	return sourceResults, targetResults, nil
}

func toFingerprintSearchResult(queryID string, response connectors.SearchResponse) fingerprints.SearchResult {
	hits := make([]fingerprints.SearchHit, 0, len(response.Hits))
	for _, hit := range response.Hits {
		hits = append(hits, fingerprints.SearchHit{
			ID:    hit.ID,
			Rank:  hit.Rank,
			Score: hit.Score,
		})
	}
	return fingerprints.SearchResult{QueryID: queryID, Hits: hits}
}
