# seed-milvus CLI

`vdbg seed-milvus` seeds deterministic synthetic vector records into a real Milvus collection through the Milvus Go SDK.

This command is the source-side write loop for the Milvus to pgvector migration-and-verification MVP.

## Command

```bash
go run ./cmd/vdbg seed-milvus \
  --fixture testdata/migration/synthetic-small.json \
  --address localhost:19530 \
  --collection items \
  --id-field id \
  --vector-field embedding \
  --metric cosine
```

## Behavior

The command:

1. Loads a synthetic fixture JSON file.
2. Infers the Milvus vector dimension from `dataset.dimension`.
3. Connects to an already-running Milvus endpoint.
4. Drops the target collection if it exists.
5. Creates a minimal collection with:
   - a `VarChar` primary key field;
   - a `FloatVector` field;
   - `AutoID=false`.
6. Creates a FLAT vector index with the requested metric.
7. Loads the collection.
8. Inserts all fixture records as columnar Milvus data.
9. Flushes the collection.
10. Prints a compact seed summary.

## Output

Example:

```text
milvus fixture seeded
fixture: testdata/migration/synthetic-small.json
collection: items
dimension: 8
records_total: 100
records_seeded: 100
```

## Safety

`seed-milvus` recreates the configured collection. It is intentionally destructive for the selected collection and should only be used against local development or disposable test databases.

The command does not start Docker automatically. Start the local stack separately:

```bash
make migration-stack-up
scripts/check-migration-stack.sh milvus-port
```

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--fixture` | required | Synthetic fixture JSON file. |
| `--address` | required | Milvus gRPC address, for example `localhost:19530`. |
| `--collection` | `items` | Collection to drop, recreate, and seed. |
| `--id-field` | `id` | VarChar primary key field. |
| `--vector-field` | `embedding` | FloatVector field. |
| `--metric` | `cosine` | Vector metric, currently `cosine` or `l2`. |

## Validation

The underlying `MilvusSeeder` validates:

- dimension range `1..2000`;
- fixture dimension matches seeder dimension;
- non-empty record IDs;
- record vector length matches dimension;
- no `NaN` or infinite vector values;
- simple collection and field identifiers;
- supported metrics.

Identifiers must match:

```text
^[A-Za-z_][A-Za-z0-9_]*$
```

## Current limitations

- No partitions.
- No metadata fields.
- No query vector insertion.
- No production bulk import path.
- The command always recreates the selected collection.

The next source-side steps are a real Milvus search smoke CLI and a source fingerprint artifact CLI.
