package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/h3xwave/vdb-guardian/internal/connectors"
	"github.com/h3xwave/vdb-guardian/internal/migration"
)

func TestParseMigrateOptions(t *testing.T) {
	options, err := parseMigrateOptions([]string{
		"--milvus-address", "localhost:19530",
		"--source-collection", "source_items",
		"--milvus-id-field", "vector_id",
		"--milvus-vector-field", "embedding",
		"--pgvector-connection-url", "postgres://[REDACTED]",
		"--target-table", "target_items",
		"--pgvector-id-column", "vector_id",
		"--pgvector-vector-column", "embedding",
		"--dimension", "8",
		"--batch-size", "25",
	})
	if err != nil {
		t.Fatalf("parseMigrateOptions returned error: %v", err)
	}
	if options.MilvusConfig.Address != "localhost:19530" {
		t.Fatalf("unexpected milvus address: %s", options.MilvusConfig.Address)
	}
	if options.MigrationConfig.SourceCollection != "source_items" {
		t.Fatalf("unexpected source collection: %s", options.MigrationConfig.SourceCollection)
	}
	if options.MilvusConfig.IDField != "vector_id" || options.MilvusConfig.VectorField != "embedding" {
		t.Fatalf("unexpected milvus fields: %+v", options.MilvusConfig)
	}
	if options.PGVectorConfig.ConnectionURL != "postgres://[REDACTED]" {
		t.Fatalf("unexpected connection url: %s", options.PGVectorConfig.ConnectionURL)
	}
	if options.MigrationConfig.TargetTable != "target_items" {
		t.Fatalf("unexpected target table: %s", options.MigrationConfig.TargetTable)
	}
	if options.PGVectorConfig.IDColumn != "vector_id" || options.PGVectorConfig.VectorColumn != "embedding" {
		t.Fatalf("unexpected pgvector columns: %+v", options.PGVectorConfig)
	}
	if options.MigrationConfig.Dimension != 8 || options.MigrationConfig.BatchSize != 25 {
		t.Fatalf("unexpected migration config: %+v", options.MigrationConfig)
	}
}

func TestParseMigrateOptionsRejectsMissingRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing milvus address", args: []string{"--pgvector-connection-url", "postgres://[REDACTED]", "--dimension", "8"}, want: "milvus-address"},
		{name: "missing connection url", args: []string{"--milvus-address", "localhost:19530", "--dimension", "8"}, want: "pgvector-connection-url"},
		{name: "missing dimension", args: []string{"--milvus-address", "localhost:19530", "--pgvector-connection-url", "postgres://[REDACTED]"}, want: "dimension"},
		{name: "bad dimension", args: []string{"--milvus-address", "localhost:19530", "--pgvector-connection-url", "postgres://[REDACTED]", "--dimension", "0"}, want: "dimension"},
		{name: "bad batch", args: []string{"--milvus-address", "localhost:19530", "--pgvector-connection-url", "postgres://[REDACTED]", "--dimension", "8", "--batch-size", "0"}, want: "batch-size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseMigrateOptions(tt.args)
			if err == nil {
				t.Fatal("expected invalid options to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}

func TestRunMigrateWithInjectedRunner(t *testing.T) {
	fake := &fakeMigrateRunner{}
	err := runMigrateWithFactory(context.Background(), []string{
		"--milvus-address", "localhost:19530",
		"--source-collection", "items",
		"--pgvector-connection-url", "postgres://[REDACTED]",
		"--target-table", "items",
		"--dimension", "8",
	}, fake.newRunner)
	if err != nil {
		t.Fatalf("runMigrateWithFactory returned error: %v", err)
	}
	if fake.milvus.Address != "localhost:19530" {
		t.Fatalf("unexpected milvus config: %+v", fake.milvus)
	}
	if fake.pgvector.ConnectionURL != "postgres://[REDACTED]" {
		t.Fatalf("unexpected pgvector config: %+v", fake.pgvector)
	}
	if fake.config.SourceCollection != "items" || fake.config.TargetTable != "items" || fake.config.Dimension != 8 {
		t.Fatalf("unexpected migration config: %+v", fake.config)
	}
	if !fake.migrated {
		t.Fatal("expected runner to be executed")
	}
}

func TestRunMigratePropagatesFactoryAndRunnerErrors(t *testing.T) {
	err := runMigrateWithFactory(context.Background(), []string{
		"--milvus-address", "localhost:19530",
		"--pgvector-connection-url", "postgres://[REDACTED]",
		"--dimension", "8",
	}, func(connectors.MilvusConfig, connectors.PGVectorConfig, migration.VectorMigrationConfig) (migrateRunner, error) {
		return nil, errors.New("factory failed")
	})
	if err == nil || !strings.Contains(err.Error(), "factory failed") {
		t.Fatalf("expected factory error, got %v", err)
	}

	fake := &fakeMigrateRunner{err: errors.New("migrate failed")}
	err = runMigrateWithFactory(context.Background(), []string{
		"--milvus-address", "localhost:19530",
		"--pgvector-connection-url", "postgres://[REDACTED]",
		"--dimension", "8",
	}, fake.newRunner)
	if err == nil || !strings.Contains(err.Error(), "migrate failed") {
		t.Fatalf("expected runner error, got %v", err)
	}
}

type fakeMigrateRunner struct {
	milvus   connectors.MilvusConfig
	pgvector connectors.PGVectorConfig
	config   migration.VectorMigrationConfig
	migrated bool
	err      error
}

func (f *fakeMigrateRunner) newRunner(milvus connectors.MilvusConfig, pgvector connectors.PGVectorConfig, config migration.VectorMigrationConfig) (migrateRunner, error) {
	f.milvus = milvus
	f.pgvector = pgvector
	f.config = config
	return f, nil
}

func (f *fakeMigrateRunner) Migrate(ctx context.Context) (migration.VectorMigrationResult, error) {
	if err := ctx.Err(); err != nil {
		return migration.VectorMigrationResult{}, err
	}
	if f.err != nil {
		return migration.VectorMigrationResult{}, f.err
	}
	f.migrated = true
	return migration.VectorMigrationResult{
		SourceCollection: f.config.SourceCollection,
		TargetTable:      f.config.TargetTable,
		Dimension:        f.config.Dimension,
		RecordsRead:      100,
		RecordsWritten:   100,
	}, nil
}
