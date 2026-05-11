package connectors

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestNewPGVectorConnectorAppliesDefaults(t *testing.T) {
	connector, err := NewPGVectorConnector(PGVectorConfig{ConnectionURL: "postgres://local"}, nil)
	if err != nil {
		t.Fatalf("NewPGVectorConnector returned error: %v", err)
	}

	if connector.Name() != "pgvector" {
		t.Fatalf("unexpected connector name: %s", connector.Name())
	}
	if connector.config.DefaultTable != "items" {
		t.Fatalf("unexpected default table: %s", connector.config.DefaultTable)
	}
	if connector.config.IDColumn != "id" {
		t.Fatalf("unexpected id column: %s", connector.config.IDColumn)
	}
	if connector.config.VectorColumn != "embedding" {
		t.Fatalf("unexpected vector column: %s", connector.config.VectorColumn)
	}
	if connector.config.Metric != PGVectorMetricCosine {
		t.Fatalf("unexpected metric: %s", connector.config.Metric)
	}
}

func TestNewPGVectorConnectorCreatesDatabaseAdapterFromConnectionURL(t *testing.T) {
	connector, err := NewPGVectorConnector(PGVectorConfig{ConnectionURL: "postgres://user:pass@localhost:5432/db"}, nil)
	if err != nil {
		t.Fatalf("NewPGVectorConnector returned error: %v", err)
	}
	if connector.db == nil {
		t.Fatalf("expected connection URL to create a database adapter")
	}
}

func TestNewPGVectorConnectorRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config PGVectorConfig
	}{
		{
			name:   "missing connection url without injected db",
			config: PGVectorConfig{},
		},
		{
			name:   "invalid table identifier",
			config: PGVectorConfig{ConnectionURL: "postgres://local", DefaultTable: "bad-table"},
		},
		{
			name:   "invalid id column",
			config: PGVectorConfig{ConnectionURL: "postgres://local", IDColumn: "id;drop"},
		},
		{
			name:   "unsupported metric",
			config: PGVectorConfig{ConnectionURL: "postgres://local", Metric: "dot"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPGVectorConnector(tt.config, nil)
			if err == nil {
				t.Fatalf("expected invalid config to fail")
			}
		})
	}
}

func TestPGVectorConnectorConnectPingsDatabaseAndChecksExtension(t *testing.T) {
	db := &fakePGVectorDB{
		extensionInstalled: true,
	}
	connector := mustPGVectorConnector(t, db)

	if err := connector.Connect(context.Background()); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	if !db.pinged {
		t.Fatalf("expected Ping to be called")
	}
	if !strings.Contains(db.lastRowSQL, "pg_extension") {
		t.Fatalf("expected extension check SQL, got %s", db.lastRowSQL)
	}
}

func TestPGVectorConnectorConnectRejectsMissingExtension(t *testing.T) {
	db := &fakePGVectorDB{extensionInstalled: false}
	connector := mustPGVectorConnector(t, db)

	err := connector.Connect(context.Background())
	if err == nil {
		t.Fatalf("expected missing pgvector extension to fail")
	}
	if !strings.Contains(err.Error(), "pgvector") {
		t.Fatalf("expected pgvector error, got %v", err)
	}
}

func TestPGVectorConnectorCountUsesSafeTableIdentifier(t *testing.T) {
	db := &fakePGVectorDB{count: 42}
	connector := mustPGVectorConnector(t, db)

	count, err := connector.Count(context.Background(), "vectors")
	if err != nil {
		t.Fatalf("Count returned error: %v", err)
	}
	if count != 42 {
		t.Fatalf("expected count 42, got %d", count)
	}
	if !strings.Contains(db.lastRowSQL, `FROM "vectors"`) {
		t.Fatalf("expected quoted table identifier, got %s", db.lastRowSQL)
	}
}

func TestPGVectorConnectorCountRejectsUnsafeTable(t *testing.T) {
	connector := mustPGVectorConnector(t, &fakePGVectorDB{})

	_, err := connector.Count(context.Background(), "items;drop")
	if err == nil {
		t.Fatalf("expected unsafe table to fail")
	}
}

func TestPGVectorConnectorSearchReturnsRankedHits(t *testing.T) {
	db := &fakePGVectorDB{
		rows: []fakePGVectorSearchRow{
			{id: "a", score: 0.99},
			{id: "b", score: 0.95},
			{id: "c", score: 0.91},
		},
	}
	connector := mustPGVectorConnector(t, db)

	response, err := connector.Search(context.Background(), SearchRequest{
		Collection:  "items",
		QueryVector: []float64{0.1, 0.2, 0.3},
		TopK:        2,
		ExpandK:     3,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	expected := []SearchHit{
		{ID: "a", Rank: 1, Score: 0.99},
		{ID: "b", Rank: 2, Score: 0.95},
		{ID: "c", Rank: 3, Score: 0.91},
	}
	if !reflect.DeepEqual(response.Hits, expected) {
		t.Fatalf("unexpected hits: %#v", response.Hits)
	}
	if db.lastQueryArgs[0] != "[0.1,0.2,0.3]" {
		t.Fatalf("unexpected vector literal arg: %#v", db.lastQueryArgs[0])
	}
	if db.lastQueryArgs[1] != 3 {
		t.Fatalf("expected ExpandK limit 3, got %#v", db.lastQueryArgs[1])
	}
	if !strings.Contains(db.lastQuerySQL, `<=> $1::vector`) {
		t.Fatalf("expected cosine search SQL, got %s", db.lastQuerySQL)
	}
}

func TestPGVectorConnectorSearchRejectsInvalidRequest(t *testing.T) {
	connector := mustPGVectorConnector(t, &fakePGVectorDB{})
	tests := []struct {
		name string
		req  SearchRequest
	}{
		{name: "empty query vector", req: SearchRequest{TopK: 1, ExpandK: 1}},
		{name: "zero top k", req: SearchRequest{QueryVector: []float64{0.1}, TopK: 0, ExpandK: 1}},
		{name: "expand below top k", req: SearchRequest{QueryVector: []float64{0.1}, TopK: 2, ExpandK: 1}},
		{name: "unsafe collection", req: SearchRequest{Collection: "items;drop", QueryVector: []float64{0.1}, TopK: 1, ExpandK: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := connector.Search(context.Background(), tt.req)
			if err == nil {
				t.Fatalf("expected invalid request to fail")
			}
		})
	}
}

func TestPGVectorConnectorSearchSupportsL2Metric(t *testing.T) {
	db := &fakePGVectorDB{rows: []fakePGVectorSearchRow{{id: "a", score: -0.2}}}
	connector, err := NewPGVectorConnector(PGVectorConfig{Metric: PGVectorMetricL2}, db)
	if err != nil {
		t.Fatalf("NewPGVectorConnector returned error: %v", err)
	}

	_, err = connector.Search(context.Background(), SearchRequest{QueryVector: []float64{0.1}, TopK: 1, ExpandK: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if !strings.Contains(db.lastQuerySQL, `<-> $1::vector`) {
		t.Fatalf("expected l2 search SQL, got %s", db.lastQuerySQL)
	}
}

func TestFormatPGVectorLiteralRejectsInvalidValues(t *testing.T) {
	tests := [][]float64{
		{},
		{mustInf()},
	}
	for _, values := range tests {
		_, err := formatPGVectorLiteral(values)
		if err == nil {
			t.Fatalf("expected vector literal formatting to fail for %#v", values)
		}
	}
}

func TestPGVectorConnectorCloseClosesDatabase(t *testing.T) {
	db := &fakePGVectorDB{}
	connector := mustPGVectorConnector(t, db)

	if err := connector.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !db.closed {
		t.Fatalf("expected db to be closed")
	}
}

func mustInf() float64 {
	return math.Inf(1)
}

func mustPGVectorConnector(t *testing.T, db pgvectorDB) PGVectorConnector {
	t.Helper()
	connector, err := NewPGVectorConnector(PGVectorConfig{}, db)
	if err != nil {
		t.Fatalf("NewPGVectorConnector returned error: %v", err)
	}
	return connector
}

type fakePGVectorDB struct {
	pinged             bool
	closed             bool
	extensionInstalled bool
	count              int64
	rows               []fakePGVectorSearchRow
	lastRowSQL         string
	lastRowArgs        []any
	lastQuerySQL       string
	lastQueryArgs      []any
}

func (db *fakePGVectorDB) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	db.pinged = true
	return nil
}

func (db *fakePGVectorDB) QueryRow(ctx context.Context, sql string, args ...any) pgvectorRow {
	db.lastRowSQL = sql
	db.lastRowArgs = args
	if strings.Contains(sql, "pg_extension") {
		return fakePGVectorRow{values: []any{db.extensionInstalled}}
	}
	return fakePGVectorRow{values: []any{db.count}}
}

func (db *fakePGVectorDB) Query(ctx context.Context, sql string, args ...any) (pgvectorRows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	db.lastQuerySQL = sql
	db.lastQueryArgs = args
	return &fakePGVectorRows{rows: db.rows}, nil
}

func (db *fakePGVectorDB) Close() error {
	db.closed = true
	return nil
}

type fakePGVectorRow struct {
	values []any
	err    error
}

func (row fakePGVectorRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	if len(dest) != len(row.values) {
		return errors.New("unexpected scan destination count")
	}
	for index, value := range row.values {
		switch target := dest[index].(type) {
		case *bool:
			*target = value.(bool)
		case *int64:
			*target = value.(int64)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

type fakePGVectorSearchRow struct {
	id    string
	score float64
}

type fakePGVectorRows struct {
	rows  []fakePGVectorSearchRow
	index int
	err   error
}

func (rows *fakePGVectorRows) Next() bool {
	return rows.index < len(rows.rows)
}

func (rows *fakePGVectorRows) Scan(dest ...any) error {
	if len(dest) != 2 {
		return errors.New("unexpected scan destination count")
	}
	current := rows.rows[rows.index]
	rows.index++
	*(dest[0].(*string)) = current.id
	*(dest[1].(*float64)) = current.score
	return nil
}

func (rows *fakePGVectorRows) Err() error {
	return rows.err
}

func (rows *fakePGVectorRows) Close() {
}
