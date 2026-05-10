package fingerprints

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBuildArtifactCreatesQueryFingerprints(t *testing.T) {
	artifact, err := BuildArtifact([]SearchResult{
		{
			QueryID: "q-1",
			Hits: []SearchHit{
				{ID: "a", Rank: 1, Score: 0.99},
				{ID: "b", Rank: 2, Score: 0.95},
				{ID: "c", Rank: 3, Score: 0.90},
				{ID: "d", Rank: 4, Score: 0.89},
				{ID: "e", Rank: 5, Score: 0.88},
			},
		},
	}, BuildOptions{TopK: 3, StableK: 2, BoundaryK: 2})
	if err != nil {
		t.Fatalf("expected artifact build to succeed: %v", err)
	}

	if len(artifact.Fingerprints) != 1 {
		t.Fatalf("expected one fingerprint, got %d", len(artifact.Fingerprints))
	}
	fingerprint := artifact.Fingerprints[0]
	assertEqualStrings(t, fingerprint.TopKIDs, []string{"a", "b", "c"})
	assertEqualStrings(t, fingerprint.StableNeighbors, []string{"a", "b"})
	assertEqualStrings(t, fingerprint.BoundaryCandidates, []string{"b", "c", "d", "e"})
}

func TestBuildArtifactSortsHitsByRank(t *testing.T) {
	artifact, err := BuildArtifact([]SearchResult{
		{
			QueryID: "q-1",
			Hits: []SearchHit{
				{ID: "c", Rank: 3, Score: 0.90},
				{ID: "a", Rank: 1, Score: 0.99},
				{ID: "b", Rank: 2, Score: 0.95},
				{ID: "d", Rank: 4, Score: 0.89},
			},
		},
	}, BuildOptions{TopK: 2, StableK: 1, BoundaryK: 1})
	if err != nil {
		t.Fatalf("expected artifact build to succeed: %v", err)
	}

	fingerprint := artifact.Fingerprints[0]
	assertEqualStrings(t, fingerprint.TopKIDs, []string{"a", "b"})
	assertEqualStrings(t, fingerprint.StableNeighbors, []string{"a"})
	assertEqualStrings(t, fingerprint.BoundaryCandidates, []string{"b", "c"})
}

func TestBuildArtifactRejectsInvalidOptions(t *testing.T) {
	cases := []struct {
		name    string
		options BuildOptions
	}{
		{name: "top k", options: BuildOptions{TopK: 0, StableK: 1, BoundaryK: 1}},
		{name: "stable k", options: BuildOptions{TopK: 3, StableK: 4, BoundaryK: 1}},
		{name: "boundary k", options: BuildOptions{TopK: 3, StableK: 2, BoundaryK: 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildArtifact([]SearchResult{{QueryID: "q-1", Hits: validHits()}}, tc.options)
			if err == nil {
				t.Fatal("expected invalid options to return an error")
			}
		})
	}
}

func TestBuildArtifactRejectsDuplicateQueryIDs(t *testing.T) {
	_, err := BuildArtifact([]SearchResult{
		{QueryID: "q-1", Hits: validHits()},
		{QueryID: "q-1", Hits: validHits()},
	}, BuildOptions{TopK: 3, StableK: 2, BoundaryK: 1})

	if err == nil {
		t.Fatal("expected duplicate query ids to return an error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate query id error, got %v", err)
	}
}

func TestBuildArtifactRejectsInsufficientHits(t *testing.T) {
	_, err := BuildArtifact([]SearchResult{
		{QueryID: "q-1", Hits: []SearchHit{{ID: "a", Rank: 1, Score: 0.99}}},
	}, BuildOptions{TopK: 3, StableK: 2, BoundaryK: 1})

	if err == nil {
		t.Fatal("expected insufficient hits to return an error")
	}
	if !strings.Contains(err.Error(), "at least top_k") {
		t.Fatalf("expected top_k error, got %v", err)
	}
}

func TestWriteArtifactWritesExpectedJSON(t *testing.T) {
	path := t.TempDir() + "/artifact.json"
	artifact := Artifact{Fingerprints: []QueryFingerprint{
		{
			QueryID:            "q-1",
			StableNeighbors:    []string{"a", "b"},
			BoundaryCandidates: []string{"b", "c"},
			TopKIDs:            []string{"a", "b", "c"},
		},
	}}

	if err := WriteArtifact(path, artifact); err != nil {
		t.Fatalf("expected artifact write to succeed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if !strings.Contains(string(data), "stable_neighbors") {
		t.Fatalf("expected snake_case stable_neighbors field, got %s", string(data))
	}

	var decoded Artifact
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode artifact: %v", err)
	}
	assertEqualStrings(t, decoded.Fingerprints[0].TopKIDs, []string{"a", "b", "c"})
}

func validHits() []SearchHit {
	return []SearchHit{
		{ID: "a", Rank: 1, Score: 0.99},
		{ID: "b", Rank: 2, Score: 0.95},
		{ID: "c", Rank: 3, Score: 0.90},
	}
}

func assertEqualStrings(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
