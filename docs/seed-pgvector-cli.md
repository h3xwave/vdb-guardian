# seed-pgvector CLI

`vdbg seed-pgvector` writes deterministic synthetic fixture records into a real PostgreSQL table backed by pgvector.

This is the first real database write command for the Milvus to pgvector migration-and-verification MVP. It does not start Docker or create services; it connects to an already-running PostgreSQL instance using the supplied connection URL.

## Command

```bash
go run ./cmd/vdbg seed-pgvector \
  --fixture testdata/migration/synthetic-small.json \
  --connection-url '[REDACTED]' \
  --table items \
  --id-column id \
  --vector-column embedding
```

Required flags:

- `--fixture`: path to a synthetic fixture JSON file.
- `--connection-url`: PostgreSQL connection URL for the pgvector database.

Optional flags:

- `--table`: target table name, default `items`.
- `--id-column`: text primary key column, default `id`.
- `--vector-column`: pgvector column, default `embedding`.

Do not commit real connection URLs. Use `[REDACTED]` in documentation, issue comments, and shared logs.

## Behavior

The command:

1. Loads the synthetic fixture JSON.
2. Uses the fixture `dimension` as the pgvector column dimension.
3. Creates a pgx-backed seeding adapter.
4. Runs `migration.PGVectorSeeder`.
5. Prints a small summary.

The seeder executes:

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

```sql
CREATE TABLE IF NOT EXISTS "items" (
  "id" TEXT PRIMARY KEY,
  "embedding" vector(8) NOT NULL
);
```

```sql
INSERT INTO "items" ("id", "embedding")
VALUES ($1, $2::vector)
ON CONFLICT ("id")
DO UPDATE SET "embedding" = EXCLUDED."embedding";
```

The command is idempotent for the same fixture because records are upserted by ID.

## Safety

The command validates table and column identifiers before executing SQL. Identifiers must match:

```text
^[A-Za-z_][A-Za-z0-9_]*$
```

The command rejects unsafe names such as:

```text
items;drop
public.items
"items"
items-name
```

Vector values are encoded as pgvector literals and rejected when they are empty, `NaN`, or infinite.

## Current limitations

Implemented:

- Real pgx-backed PostgreSQL execution.
- Synthetic fixture loading.
- pgvector extension/table creation.
- Record upsert.
- CLI option parsing and unit tests.

Not yet implemented:

- Automatic Docker stack startup.
- Integration test against the local migration stack.
- Milvus real SDK seeding command.
- End-to-end migrate-and-verify command.
- Index creation such as HNSW or IVFFlat.
- Metadata columns or schema mapping.

## MVP role

This command enables the target-side real database write loop:

```text
synthetic fixture JSON
        ↓
vdbg seed-pgvector
        ↓
PostgreSQL pgvector table
        ↓
PGVectorConnector.Search
        ↓
target fingerprint artifact
```

A later Milvus seed command and migrate-and-verify CLI will complete the source-to-target runnable database loop.
