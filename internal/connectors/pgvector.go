package connectors

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	// PGVectorMetricCosine selects pgvector cosine distance search through the
	// `<=>` operator and returns a similarity-like score of `1 - distance`.
	PGVectorMetricCosine = "cosine"

	// PGVectorMetricL2 selects pgvector Euclidean distance search through the
	// `<->` operator and returns a negative distance score so larger values remain
	// better in normalized SearchHit values.
	PGVectorMetricL2 = "l2"
)

var pgvectorIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// PGVectorConfig configures the minimal pgvector connector used by the migration
// MVP.
//
// The connector intentionally supports simple table and column identifiers only.
// This keeps dynamic SQL safe while the first Milvus to pgvector migration loop
// is being built. Schema-qualified names, quoted identifiers, filters, and index
// tuning are planned later.
type PGVectorConfig struct {
	Name          string
	ConnectionURL string
	DefaultTable  string
	IDColumn      string
	VectorColumn  string
	Metric        string
}

// PGVectorConnector implements normalized vector search against PostgreSQL with
// the pgvector extension.
//
// The first implementation is intentionally minimal: it can verify pgvector is
// installed, count rows in a table, and execute topK/expandK vector searches that
// return normalized SearchResponse values for fingerprint artifact generation.
type PGVectorConnector struct {
	config PGVectorConfig
	db     pgvectorDB
}

type pgvectorDB interface {
	Ping(ctx context.Context) error
	QueryRow(ctx context.Context, sql string, args ...any) pgvectorRow
	Query(ctx context.Context, sql string, args ...any) (pgvectorRows, error)
	Close() error
}

type pgvectorRow interface {
	Scan(dest ...any) error
}

type pgvectorRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

type pgxPGVectorDB struct {
	connectionURL string
	conn          *pgx.Conn
}

func newPGXPGVectorDB(connectionURL string) *pgxPGVectorDB {
	return &pgxPGVectorDB{connectionURL: connectionURL}
}

func (db *pgxPGVectorDB) Ping(ctx context.Context) error {
	conn, err := db.connection(ctx)
	if err != nil {
		return err
	}
	return conn.Ping(ctx)
}

func (db *pgxPGVectorDB) QueryRow(ctx context.Context, sql string, args ...any) pgvectorRow {
	conn, err := db.connection(ctx)
	if err != nil {
		return pgvectorErrorRow{err: err}
	}
	return conn.QueryRow(ctx, sql, args...)
}

func (db *pgxPGVectorDB) Query(ctx context.Context, sql string, args ...any) (pgvectorRows, error) {
	conn, err := db.connection(ctx)
	if err != nil {
		return nil, err
	}
	return conn.Query(ctx, sql, args...)
}

func (db *pgxPGVectorDB) Close() error {
	if db.conn == nil {
		return nil
	}
	ctx := context.Background()
	return db.conn.Close(ctx)
}

func (db *pgxPGVectorDB) connection(ctx context.Context) (*pgx.Conn, error) {
	if db.conn != nil {
		return db.conn, nil
	}
	conn, err := pgx.Connect(ctx, db.connectionURL)
	if err != nil {
		return nil, fmt.Errorf("connect pgvector database: %w", err)
	}
	db.conn = conn
	return db.conn, nil
}

type pgvectorErrorRow struct {
	err error
}

func (row pgvectorErrorRow) Scan(dest ...any) error {
	return row.err
}

// NewPGVectorConnector validates configuration and returns a pgvector connector.
//
// A database handle may be injected for tests. When no database handle is
// injected, the ConnectionURL is required so a real PostgreSQL adapter can be
// attached in the next integration step without changing the connector API.
func NewPGVectorConnector(config PGVectorConfig, db pgvectorDB) (PGVectorConnector, error) {
	config = applyPGVectorDefaults(config)
	if err := validatePGVectorConfig(config, db); err != nil {
		return PGVectorConnector{}, err
	}
	if db == nil {
		db = newPGXPGVectorDB(config.ConnectionURL)
	}
	return PGVectorConnector{config: config, db: db}, nil
}

// Name returns the stable connector name used in logs, configuration, and
// reports.
func (c PGVectorConnector) Name() string {
	return c.config.Name
}

// Connect verifies that the configured PostgreSQL database is reachable and has
// the pgvector extension installed.
//
// It returns an error when the connector was created without a database adapter,
// the context is canceled, the ping fails, or pgvector is not installed.
func (c PGVectorConnector) Connect(ctx context.Context) error {
	if c.db == nil {
		return errors.New("pgvector database adapter is not configured")
	}
	if err := c.db.Ping(ctx); err != nil {
		return fmt.Errorf("ping pgvector database: %w", err)
	}
	var installed bool
	if err := c.db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&installed); err != nil {
		return fmt.Errorf("check pgvector extension: %w", err)
	}
	if !installed {
		return errors.New("pgvector extension is not installed")
	}
	return nil
}

// Count returns the number of rows in the requested pgvector table.
//
// If collection is empty, the connector uses DefaultTable. Only simple SQL
// identifiers are accepted to prevent injection through dynamic table names.
func (c PGVectorConnector) Count(ctx context.Context, collection string) (int64, error) {
	if c.db == nil {
		return 0, errors.New("pgvector database adapter is not configured")
	}
	table, err := c.tableForCollection(collection)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quotePGVectorIdentifier(table))
	var count int64
	if err := c.db.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count pgvector rows: %w", err)
	}
	return count, nil
}

// Search executes a normalized vector search against pgvector and returns ranked
// SearchHit values.
//
// ExpandK is used as the SQL LIMIT so the fingerprint builder can observe topK
// boundary candidates. Cosine search returns `1 - distance`; L2 search returns
// negative distance so higher normalized scores still represent better matches.
func (c PGVectorConnector) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if c.db == nil {
		return SearchResponse{}, errors.New("pgvector database adapter is not configured")
	}
	if err := validatePGVectorSearchRequest(req); err != nil {
		return SearchResponse{}, err
	}
	table, err := c.tableForCollection(req.Collection)
	if err != nil {
		return SearchResponse{}, err
	}
	literal, err := formatPGVectorLiteral(req.QueryVector)
	if err != nil {
		return SearchResponse{}, err
	}
	query := c.searchSQL(table)
	rows, err := c.db.Query(ctx, query, literal, req.ExpandK)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("query pgvector search: %w", err)
	}
	defer rows.Close()

	hits := make([]SearchHit, 0, req.ExpandK)
	for rows.Next() {
		var id string
		var score float64
		if err := rows.Scan(&id, &score); err != nil {
			return SearchResponse{}, fmt.Errorf("scan pgvector search row: %w", err)
		}
		hits = append(hits, SearchHit{ID: id, Rank: len(hits) + 1, Score: score})
	}
	if err := rows.Err(); err != nil {
		return SearchResponse{}, fmt.Errorf("iterate pgvector search rows: %w", err)
	}
	return SearchResponse{Hits: hits}, nil
}

// Close releases the underlying pgvector database adapter when one is present.
func (c PGVectorConnector) Close() error {
	if c.db == nil {
		return nil
	}
	return c.db.Close()
}

func applyPGVectorDefaults(config PGVectorConfig) PGVectorConfig {
	if config.Name == "" {
		config.Name = "pgvector"
	}
	if config.DefaultTable == "" {
		config.DefaultTable = "items"
	}
	if config.IDColumn == "" {
		config.IDColumn = "id"
	}
	if config.VectorColumn == "" {
		config.VectorColumn = "embedding"
	}
	if config.Metric == "" {
		config.Metric = PGVectorMetricCosine
	}
	return config
}

func validatePGVectorConfig(config PGVectorConfig, db pgvectorDB) error {
	if config.ConnectionURL == "" && db == nil {
		return errors.New("pgvector connection URL is required when no database adapter is injected")
	}
	if err := validatePGVectorIdentifier("default table", config.DefaultTable); err != nil {
		return err
	}
	if err := validatePGVectorIdentifier("id column", config.IDColumn); err != nil {
		return err
	}
	if err := validatePGVectorIdentifier("vector column", config.VectorColumn); err != nil {
		return err
	}
	if config.Metric != PGVectorMetricCosine && config.Metric != PGVectorMetricL2 {
		return fmt.Errorf("unsupported pgvector metric %q", config.Metric)
	}
	return nil
}

func validatePGVectorSearchRequest(req SearchRequest) error {
	if len(req.QueryVector) == 0 {
		return errors.New("pgvector query vector is required")
	}
	if req.TopK <= 0 {
		return errors.New("pgvector topK must be positive")
	}
	if req.ExpandK <= 0 {
		return errors.New("pgvector expandK must be positive")
	}
	if req.ExpandK < req.TopK {
		return errors.New("pgvector expandK must be greater than or equal to topK")
	}
	return nil
}

func (c PGVectorConnector) tableForCollection(collection string) (string, error) {
	table := collection
	if table == "" {
		table = c.config.DefaultTable
	}
	if err := validatePGVectorIdentifier("collection", table); err != nil {
		return "", err
	}
	return table, nil
}

func (c PGVectorConnector) searchSQL(table string) string {
	idColumn := quotePGVectorIdentifier(c.config.IDColumn)
	vectorColumn := quotePGVectorIdentifier(c.config.VectorColumn)
	quotedTable := quotePGVectorIdentifier(table)
	switch c.config.Metric {
	case PGVectorMetricL2:
		return fmt.Sprintf("SELECT %s, -(%s <-> $1::vector) AS score FROM %s ORDER BY %s <-> $1::vector LIMIT $2", idColumn, vectorColumn, quotedTable, vectorColumn)
	default:
		return fmt.Sprintf("SELECT %s, 1 - (%s <=> $1::vector) AS score FROM %s ORDER BY %s <=> $1::vector LIMIT $2", idColumn, vectorColumn, quotedTable, vectorColumn)
	}
}

func validatePGVectorIdentifier(label string, value string) error {
	if !pgvectorIdentifierPattern.MatchString(value) {
		return fmt.Errorf("invalid pgvector %s identifier %q", label, value)
	}
	return nil
}

func quotePGVectorIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func formatPGVectorLiteral(values []float64) (string, error) {
	if len(values) == 0 {
		return "", errors.New("pgvector vector literal requires at least one value")
	}
	parts := make([]string, len(values))
	for index, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return "", fmt.Errorf("pgvector vector value at index %d must be finite", index)
		}
		parts[index] = strconv.FormatFloat(value, 'g', -1, 64)
	}
	return "[" + strings.Join(parts, ",") + "]", nil
}
