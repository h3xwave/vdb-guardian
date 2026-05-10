package migration

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestMilvusSDKSeedDBConnectUsesClientFactory(t *testing.T) {
	db := newMilvusSDKSeedDBWithClientFactory("localhost:19530", func(ctx context.Context, address string) (milvusSeedSDKClient, error) {
		if address != "localhost:19530" {
			t.Fatalf("unexpected address: %s", address)
		}
		return &fakeMilvusSeedSDKClient{}, nil
	})

	if err := db.Connect(context.Background()); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
}

func TestMilvusSDKSeedDBCreateCollectionDropsExistingAndPreparesCollection(t *testing.T) {
	client := &fakeMilvusSeedSDKClient{hasCollection: true}
	db := connectedFakeMilvusSDKSeedDB(client)

	err := db.CreateCollection(context.Background(), milvusCreateCollectionRequest{
		Collection:  "items",
		IDField:     "id",
		VectorField: "embedding",
		Dimension:   8,
		Metric:      "cosine",
	})
	if err != nil {
		t.Fatalf("CreateCollection returned error: %v", err)
	}
	if !client.dropCalled {
		t.Fatal("expected existing collection to be dropped")
	}
	if client.created.Collection != "items" || client.created.IDField != "id" || client.created.VectorField != "embedding" {
		t.Fatalf("unexpected create request: %#v", client.created)
	}
	if client.created.Dimension != 8 || client.created.Metric != "cosine" {
		t.Fatalf("unexpected create dimension/metric: %#v", client.created)
	}
	if !client.indexCreated || !client.loaded {
		t.Fatalf("expected index and load calls, index=%v load=%v", client.indexCreated, client.loaded)
	}
}

func TestMilvusSDKSeedDBCreateCollectionSkipsDropWhenMissing(t *testing.T) {
	client := &fakeMilvusSeedSDKClient{hasCollection: false}
	db := connectedFakeMilvusSDKSeedDB(client)

	if err := db.CreateCollection(context.Background(), milvusCreateCollectionRequest{Collection: "items", IDField: "id", VectorField: "embedding", Dimension: 8, Metric: "l2"}); err != nil {
		t.Fatalf("CreateCollection returned error: %v", err)
	}
	if client.dropCalled {
		t.Fatal("did not expect missing collection to be dropped")
	}
	if client.created.Metric != "l2" {
		t.Fatalf("expected l2 metric, got %#v", client.created)
	}
}

func TestMilvusSDKSeedDBInsertRecordsConvertsColumnsAndFlushes(t *testing.T) {
	client := &fakeMilvusSeedSDKClient{}
	db := connectedFakeMilvusSDKSeedDB(client)

	err := db.InsertRecords(context.Background(), milvusInsertRecordsRequest{
		Collection:  "items",
		IDField:     "id",
		VectorField: "embedding",
		Records: []milvusSeedRecord{
			{ID: "vec-1", Vector: []float64{0.1, 0.2}},
			{ID: "vec-2", Vector: []float64{0.3, 0.4}},
		},
	})
	if err != nil {
		t.Fatalf("InsertRecords returned error: %v", err)
	}
	if client.inserted.Collection != "items" || client.inserted.IDField != "id" || client.inserted.VectorField != "embedding" {
		t.Fatalf("unexpected insert request: %#v", client.inserted)
	}
	if !reflect.DeepEqual(client.inserted.IDs, []string{"vec-1", "vec-2"}) {
		t.Fatalf("unexpected ids: %#v", client.inserted.IDs)
	}
	assertMilvusSeedFloat32MatrixAlmostEqual(t, client.inserted.Vectors, [][]float32{{0.1, 0.2}, {0.3, 0.4}})
	if !client.flushed {
		t.Fatal("expected collection to be flushed after insert")
	}
}

func TestMilvusSDKSeedDBRejectsDisconnectedUsage(t *testing.T) {
	db := newMilvusSDKSeedDBWithClientFactory("localhost:19530", nil)
	if err := db.CreateCollection(context.Background(), milvusCreateCollectionRequest{Collection: "items"}); err == nil {
		t.Fatal("expected CreateCollection to reject disconnected client")
	}
	if err := db.InsertRecords(context.Background(), milvusInsertRecordsRequest{Collection: "items"}); err == nil {
		t.Fatal("expected InsertRecords to reject disconnected client")
	}
}

func TestMilvusSDKSeedDBPropagatesClientErrors(t *testing.T) {
	client := &fakeMilvusSeedSDKClient{err: errMilvusSeedSDKFake}
	db := connectedFakeMilvusSDKSeedDB(client)

	if err := db.CreateCollection(context.Background(), milvusCreateCollectionRequest{Collection: "items", IDField: "id", VectorField: "embedding", Dimension: 8, Metric: "cosine"}); !errors.Is(err, errMilvusSeedSDKFake) {
		t.Fatalf("expected create error to propagate, got %v", err)
	}
	if err := db.InsertRecords(context.Background(), milvusInsertRecordsRequest{Collection: "items", IDField: "id", VectorField: "embedding", Records: []milvusSeedRecord{{ID: "vec-1", Vector: []float64{0.1}}}}); !errors.Is(err, errMilvusSeedSDKFake) {
		t.Fatalf("expected insert error to propagate, got %v", err)
	}
}

func TestMilvusSDKSeedDBCloseReleasesClient(t *testing.T) {
	client := &fakeMilvusSeedSDKClient{}
	db := connectedFakeMilvusSDKSeedDB(client)
	if err := db.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !client.closed {
		t.Fatal("expected client to be closed")
	}
}

func connectedFakeMilvusSDKSeedDB(client *fakeMilvusSeedSDKClient) *milvusSDKSeedDB {
	db := newMilvusSDKSeedDBWithClientFactory("localhost:19530", func(ctx context.Context, address string) (milvusSeedSDKClient, error) {
		return client, nil
	})
	if err := db.Connect(context.Background()); err != nil {
		panic(err)
	}
	return db
}

type fakeMilvusSeedSDKClient struct {
	hasCollection bool
	err           error

	dropCalled   bool
	indexCreated bool
	loaded       bool
	flushed      bool
	closed       bool
	created      milvusSDKSeedCreateCollectionRequest
	inserted     milvusSDKSeedInsertRequest
}

func (c *fakeMilvusSeedSDKClient) HasCollection(ctx context.Context, collection string) (bool, error) {
	if c.err != nil {
		return false, c.err
	}
	return c.hasCollection, nil
}

func (c *fakeMilvusSeedSDKClient) DropCollection(ctx context.Context, collection string) error {
	if c.err != nil {
		return c.err
	}
	c.dropCalled = true
	return nil
}

func (c *fakeMilvusSeedSDKClient) CreateCollection(ctx context.Context, req milvusSDKSeedCreateCollectionRequest) error {
	if c.err != nil {
		return c.err
	}
	c.created = req
	return nil
}

func (c *fakeMilvusSeedSDKClient) CreateIndex(ctx context.Context, collection string, vectorField string, metric string) error {
	if c.err != nil {
		return c.err
	}
	c.indexCreated = true
	return nil
}

func (c *fakeMilvusSeedSDKClient) LoadCollection(ctx context.Context, collection string) error {
	if c.err != nil {
		return c.err
	}
	c.loaded = true
	return nil
}

func (c *fakeMilvusSeedSDKClient) Insert(ctx context.Context, req milvusSDKSeedInsertRequest) error {
	if c.err != nil {
		return c.err
	}
	c.inserted = req
	return nil
}

func (c *fakeMilvusSeedSDKClient) Flush(ctx context.Context, collection string) error {
	if c.err != nil {
		return c.err
	}
	c.flushed = true
	return nil
}

func (c *fakeMilvusSeedSDKClient) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.closed = true
	return nil
}

func assertMilvusSeedFloat32MatrixAlmostEqual(t *testing.T, got [][]float32, want [][]float32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected vector row count: got %d want %d", len(got), len(want))
	}
	for row := range want {
		if len(got[row]) != len(want[row]) {
			t.Fatalf("unexpected vector dimension at row %d: got %d want %d", row, len(got[row]), len(want[row]))
		}
		for col := range want[row] {
			if diff := got[row][col] - want[row][col]; diff < -1e-6 || diff > 1e-6 {
				t.Fatalf("vector[%d][%d] mismatch: got %f want %f", row, col, got[row][col], want[row][col])
			}
		}
	}
}

var errMilvusSeedSDKFake = errors.New("fake milvus seed sdk error")
