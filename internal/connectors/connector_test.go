package connectors

import (
	"context"
	"testing"
)

type fakeConnector struct{}

func (fakeConnector) Name() string                                 { return "fake" }
func (fakeConnector) Connect(context.Context) error                { return nil }
func (fakeConnector) Count(context.Context, string) (int64, error) { return 42, nil }
func (fakeConnector) Search(context.Context, SearchRequest) (SearchResponse, error) {
	return SearchResponse{Hits: []SearchHit{{ID: "doc-1", Rank: 1, Score: 0.99}}}, nil
}
func (fakeConnector) Close() error { return nil }

func TestConnectorInterfaceSupportsSearchContract(t *testing.T) {
	var connector Connector = fakeConnector{}

	if connector.Name() != "fake" {
		t.Fatalf("expected connector name fake, got %q", connector.Name())
	}

	response, err := connector.Search(context.Background(), SearchRequest{
		Collection:  "items",
		QueryVector: []float64{0.1, 0.2},
		TopK:        10,
		ExpandK:     20,
	})
	if err != nil {
		t.Fatalf("expected search to succeed: %v", err)
	}

	if len(response.Hits) != 1 || response.Hits[0].ID != "doc-1" {
		t.Fatalf("expected one search hit with id doc-1, got %#v", response.Hits)
	}
}
