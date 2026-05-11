package connectors

import (
	"context"
	"errors"
	"testing"
)

func TestMilvusSDKAdapterConnectUsesClientFactory(t *testing.T) {
	adapter := newMilvusSDKAdapterWithClientFactory("localhost:19530", func(ctx context.Context, address string) (milvusSDKClient, error) {
		if address != "localhost:19530" {
			t.Fatalf("unexpected address: %s", address)
		}
		return &fakeMilvusSDKClient{}, nil
	})

	if err := adapter.Connect(context.Background()); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	if adapter.client == nil {
		t.Fatalf("expected connected client to be stored")
	}
}

func TestMilvusSDKAdapterCountReadsRowCountStat(t *testing.T) {
	client := &fakeMilvusSDKClient{stats: map[string]string{"row_count": "42"}}
	adapter := &milvusSDKDB{address: "localhost:19530", client: client}

	count, err := adapter.Count(context.Background(), "items")
	if err != nil {
		t.Fatalf("Count returned error: %v", err)
	}
	if count != 42 {
		t.Fatalf("expected count 42, got %d", count)
	}
	if client.lastCountCollection != "items" {
		t.Fatalf("unexpected collection: %s", client.lastCountCollection)
	}
}

func TestMilvusSDKAdapterCountRejectsMissingOrInvalidRowCount(t *testing.T) {
	tests := []struct {
		name  string
		stats map[string]string
	}{
		{name: "missing row count", stats: map[string]string{}},
		{name: "invalid row count", stats: map[string]string{"row_count": "not-a-number"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &milvusSDKDB{address: "localhost:19530", client: &fakeMilvusSDKClient{stats: tt.stats}}
			_, err := adapter.Count(context.Background(), "items")
			if err == nil {
				t.Fatalf("expected invalid stats to fail")
			}
		})
	}
}

func TestMilvusSDKAdapterSearchConvertsVectorsAndHits(t *testing.T) {
	client := &fakeMilvusSDKClient{
		results: []milvusSDKSearchResult{
			{IDs: []string{"vec-1", "vec-2"}, Scores: []float32{0.9, 0.8}},
		},
	}
	adapter := &milvusSDKDB{address: "localhost:19530", client: client}

	hits, err := adapter.Search(context.Background(), milvusSearchRequest{
		Collection:  "items",
		IDField:     "id",
		VectorField: "embedding",
		QueryVector: []float64{0.1, 0.2, 0.3},
		Limit:       2,
		Metric:      MilvusMetricCosine,
		Params:      map[string]string{"nprobe": "8"},
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	assertMilvusRawHitsAlmostEqual(t, hits, []milvusRawHit{{ID: "vec-1", Score: 0.9}, {ID: "vec-2", Score: 0.8}})
	if client.lastSearch.Collection != "items" || client.lastSearch.VectorField != "embedding" || client.lastSearch.IDField != "id" {
		t.Fatalf("unexpected search request: %#v", client.lastSearch)
	}
	if client.lastSearch.Limit != 2 || client.lastSearch.Metric != MilvusMetricCosine {
		t.Fatalf("unexpected search limit/metric: %#v", client.lastSearch)
	}
	assertFloat32SlicesAlmostEqual(t, client.lastSearch.QueryVector, []float32{0.1, 0.2, 0.3})
	if client.lastSearch.Params["nprobe"] != "8" {
		t.Fatalf("expected search params to be forwarded, got %#v", client.lastSearch.Params)
	}
}

func TestMilvusSDKAdapterSearchRejectsUnexpectedResultShape(t *testing.T) {
	tests := []struct {
		name    string
		results []milvusSDKSearchResult
	}{
		{name: "no result sets", results: nil},
		{name: "mismatched ids and scores", results: []milvusSDKSearchResult{{IDs: []string{"vec-1"}, Scores: []float32{0.1, 0.2}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &milvusSDKDB{address: "localhost:19530", client: &fakeMilvusSDKClient{results: tt.results}}
			_, err := adapter.Search(context.Background(), milvusSearchRequest{
				Collection:  "items",
				IDField:     "id",
				VectorField: "embedding",
				QueryVector: []float64{0.1},
				Limit:       1,
				Metric:      MilvusMetricCosine,
			})
			if err == nil {
				t.Fatalf("expected invalid result shape to fail")
			}
		})
	}
}

func TestMilvusSDKAdapterCloseReleasesClient(t *testing.T) {
	client := &fakeMilvusSDKClient{}
	adapter := &milvusSDKDB{address: "localhost:19530", client: client}

	if err := adapter.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !client.closed {
		t.Fatalf("expected client close")
	}
	if adapter.client != nil {
		t.Fatalf("expected adapter client to be cleared")
	}
}

type fakeMilvusSDKClient struct {
	stats               map[string]string
	results             []milvusSDKSearchResult
	closed              bool
	lastCountCollection string
	lastSearch          milvusSDKSearchRequest
}

func (c *fakeMilvusSDKClient) Count(ctx context.Context, collection string) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.lastCountCollection = collection
	return c.stats, nil
}

func (c *fakeMilvusSDKClient) Search(ctx context.Context, req milvusSDKSearchRequest) ([]milvusSDKSearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.lastSearch = req
	return c.results, nil
}

func (c *fakeMilvusSDKClient) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.closed = true
	return nil
}

func assertMilvusRawHitsAlmostEqual(t *testing.T, got []milvusRawHit, want []milvusRawHit) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected hit length: got %d want %d", len(got), len(want))
	}
	for index := range want {
		if got[index].ID != want[index].ID {
			t.Fatalf("hit %d id mismatch: got %s want %s", index, got[index].ID, want[index].ID)
		}
		if diff := got[index].Score - want[index].Score; diff < -1e-6 || diff > 1e-6 {
			t.Fatalf("hit %d score mismatch: got %f want %f", index, got[index].Score, want[index].Score)
		}
	}
}

func assertFloat32SlicesAlmostEqual(t *testing.T, got []float32, want []float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected float slice length: got %d want %d", len(got), len(want))
	}
	for index := range want {
		if diff := got[index] - want[index]; diff < -1e-6 || diff > 1e-6 {
			t.Fatalf("float %d mismatch: got %f want %f", index, got[index], want[index])
		}
	}
}

var errMilvusSDKFake = errors.New("fake milvus sdk error")

func TestMilvusSDKAdapterPropagatesClientErrors(t *testing.T) {
	adapter := newMilvusSDKAdapterWithClientFactory("localhost:19530", func(ctx context.Context, address string) (milvusSDKClient, error) {
		return nil, errMilvusSDKFake
	})
	if !errors.Is(adapter.Connect(context.Background()), errMilvusSDKFake) {
		t.Fatalf("expected connect error to propagate")
	}
}
