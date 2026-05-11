package connectors

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMemoryConnectorSearchReturnsRequestedResults(t *testing.T) {
	connector := NewMemoryConnector("memory-source", map[string][]SearchHit{
		"q-1": {
			{ID: "c", Rank: 3, Score: 0.90},
			{ID: "a", Rank: 1, Score: 0.99},
			{ID: "b", Rank: 2, Score: 0.95},
		},
	})

	response, err := connector.Search(context.Background(), SearchRequest{Collection: "q-1", TopK: 2, ExpandK: 3})
	if err != nil {
		t.Fatalf("expected search to succeed: %v", err)
	}
	if len(response.Hits) != 3 {
		t.Fatalf("expected expand_k hits, got %d", len(response.Hits))
	}
	assertHitIDs(t, response.Hits, []string{"a", "b", "c"})
}

func TestMemoryConnectorSearchRejectsMissingQuery(t *testing.T) {
	connector := NewMemoryConnector("memory-source", map[string][]SearchHit{
		"q-1": validConnectorHits(),
	})

	_, err := connector.Search(context.Background(), SearchRequest{Collection: "missing", TopK: 2, ExpandK: 3})

	if err == nil {
		t.Fatal("expected missing query to return an error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing query error, got %v", err)
	}
}

func TestMemoryConnectorSearchRejectsInvalidTopK(t *testing.T) {
	connector := NewMemoryConnector("memory-source", map[string][]SearchHit{"q-1": validConnectorHits()})

	_, err := connector.Search(context.Background(), SearchRequest{Collection: "q-1", TopK: 0, ExpandK: 3})

	if err == nil {
		t.Fatal("expected invalid top_k to return an error")
	}
	if !strings.Contains(err.Error(), "top_k") {
		t.Fatalf("expected top_k error, got %v", err)
	}
}

func TestMemoryConnectorSearchRejectsInsufficientHits(t *testing.T) {
	connector := NewMemoryConnector("memory-source", map[string][]SearchHit{
		"q-1": {{ID: "a", Rank: 1, Score: 0.99}},
	})

	_, err := connector.Search(context.Background(), SearchRequest{Collection: "q-1", TopK: 2, ExpandK: 3})

	if err == nil {
		t.Fatal("expected insufficient hits to return an error")
	}
	if !strings.Contains(err.Error(), "expand_k") {
		t.Fatalf("expected expand_k error, got %v", err)
	}
}

func TestMemoryConnectorSearchDoesNotExposeInternalSlices(t *testing.T) {
	connector := NewMemoryConnector("memory-source", map[string][]SearchHit{"q-1": validConnectorHits()})

	first, err := connector.Search(context.Background(), SearchRequest{Collection: "q-1", TopK: 2, ExpandK: 3})
	if err != nil {
		t.Fatalf("expected first search to succeed: %v", err)
	}
	first.Hits[0].ID = "mutated"

	second, err := connector.Search(context.Background(), SearchRequest{Collection: "q-1", TopK: 2, ExpandK: 3})
	if err != nil {
		t.Fatalf("expected second search to succeed: %v", err)
	}
	assertHitIDs(t, second.Hits, []string{"a", "b", "c"})
}

func TestMemoryConnectorSearchHonorsContextCancellation(t *testing.T) {
	connector := NewMemoryConnector("memory-source", map[string][]SearchHit{"q-1": validConnectorHits()})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := connector.Search(ctx, SearchRequest{Collection: "q-1", TopK: 2, ExpandK: 3})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func validConnectorHits() []SearchHit {
	return []SearchHit{
		{ID: "a", Rank: 1, Score: 0.99},
		{ID: "b", Rank: 2, Score: 0.95},
		{ID: "c", Rank: 3, Score: 0.90},
	}
}

func assertHitIDs(t *testing.T, hits []SearchHit, want []string) {
	t.Helper()
	if len(hits) != len(want) {
		t.Fatalf("expected %v, got %#v", want, hits)
	}
	for i := range want {
		if hits[i].ID != want[i] {
			t.Fatalf("expected %v, got %#v", want, hits)
		}
	}
}
