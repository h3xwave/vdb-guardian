# pgvector Connector

The minimal pgvector connector implements the shared `connectors.Connector` interface for PostgreSQL databases with the pgvector extension installed. It is part of the Milvus to pgvector migration-and-verification MVP.

## Current scope

Implemented capabilities:

- Validate connector configuration.
- Create a PostgreSQL adapter from a connection URL through `pgx`.
- Ping the PostgreSQL database.
- Check that the `vector` extension is installed.
- Count rows in a configured table.
- Execute vector search using `TopK` / `ExpandK` requests.
- Return normalized `connectors.SearchResponse` values.

Not yet implemented:

- Automatic schema/table creation.
- Fixture seeding into pgvector.
- Metadata filters.
- Schema-qualified identifiers.
- HNSW / IVFFlat index management.
- Integration tests against a running Docker stack.

## Configuration

The connector is configured with `PGVectorConfig`:

```go
type PGVectorConfig struct {
    Name          string
    ConnectionURL string
    DefaultTable  string
    IDColumn      string
    VectorColumn  string
    Metric        string
}
```

Defaults:

```text
Name:         pgvector
DefaultTable: items
IDColumn:     id
VectorColumn: embedding
Metric:       cosine
```

`ConnectionURL` should be supplied by local configuration or runtime flags. Do not commit real credentials.

## Search behavior

The connector uses `SearchRequest.Collection` as the table name. If it is empty, `DefaultTable` is used.

For cosine search:

```sql
SELECT id, 1 - (embedding <=> $1::vector) AS score
FROM items
ORDER BY embedding <=> $1::vector
LIMIT $2;
```

For L2 search:

```sql
SELECT id, -(embedding <-> $1::vector) AS score
FROM items
ORDER BY embedding <-> $1::vector
LIMIT $2;
```

The connector uses `ExpandK` as the SQL limit so the fingerprint builder can observe topK boundary candidates.

## Safe SQL identifiers

PostgreSQL table and column names cannot be passed as SQL parameters. The connector therefore accepts only simple identifiers:

```text
^[A-Za-z_][A-Za-z0-9_]*$
```

Supported examples:

```text
items
embedding
id
```

Rejected examples:

```text
public.items
items;drop
"items"
```

Schema-qualified names and quoted identifiers can be added later through explicit structured configuration.

## Vector literals

The first implementation sends query vectors as pgvector text literals:

```text
[0.1,0.2,0.3]
```

Values must be finite numbers. Empty vectors, NaN, and Inf are rejected before SQL execution.

## MVP role

This connector is the target-side search connector for the migration MVP. The expected later flow is:

```text
synthetic fixture records
        ↓
seed pgvector table
        ↓
Search(query vectors)
        ↓
connectors.SearchResponse
        ↓
fingerprint artifact builder
        ↓
verification runner
```
