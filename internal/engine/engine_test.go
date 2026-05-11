package engine

import (
	"context"
	"testing"
)

type fakeEngine struct{}

func (fakeEngine) Compare(context.Context, CompareInput) (CompareOutput, error) {
	return CompareOutput{
		JobID:            "job-1",
		ConsistencyScore: 0.82,
		Metrics: MetricSummary{
			FingerprintDistance: 0.18,
			BoundaryFlipRate:    0.31,
		},
	}, nil
}

func TestEngineInterfaceReturnsComparisonOutput(t *testing.T) {
	var fingerprintEngine Engine = fakeEngine{}

	output, err := fingerprintEngine.Compare(context.Background(), CompareInput{JobID: "job-1"})
	if err != nil {
		t.Fatalf("expected compare to succeed: %v", err)
	}

	if output.JobID != "job-1" {
		t.Fatalf("expected job id job-1, got %q", output.JobID)
	}

	if output.ConsistencyScore < 0 || output.ConsistencyScore > 1 {
		t.Fatalf("expected score in [0,1], got %f", output.ConsistencyScore)
	}
}
