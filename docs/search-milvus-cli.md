# search-milvus CLI

`vdbg search-milvus` validates the source-side Milvus read path by counting a seeded collection and searching one query vector from a synthetic fixture.

It is a read-only smoke command for the Milvus to pgvector migration-and-verification MVP.

## Command

```bash
go run ./cmd/vdbg search-milvus \
  --fixture testdata/migration/synthetic-small.json \
  --address localhost:19530 \
  --collection items \
  --id-field id \
  --vector-field embedding \
  --top-k 3 \
  --expand-k 5 \
  --query-index 0 \
  --metric cosine
```

## Prerequisites

Seed the Milvus collection first:

```bash
go run ./cmd/vdbg seed-milvus \
  --fixture testdata/migration/synthetic-small.json \
  --address localhost:19530 \
  --collection items \
  --id-field id \
  --vector-field embedding \
  --metric cosine
```

The command does not start Docker and does not create collections.

## Behavior

The command:

1. Loads a synthetic fixture JSON file.
2. Selects one query by `--query-index`.
3. Connects to a real Milvus endpoint through the connector SDK adapter.
4. Counts the configured collection.
5. Searches the selected query vector with `expand-k` as the SDK search limit.
6. Prints the row count and ranked hits.

`top-k` is the business-visible comparison window. `expand-k` is the larger boundary-observation window used by later fingerprint artifact generation.

## Output

Example:

```text
milvus search smoke ok
fixture: testdata/migration/synthetic-small.json
collection: items
records_count: 100
query_id: query-000001
top_k: 3
expand_k: 5
hits: 5
hit rank=1 id=vec-000033 score=0.8164
```

Scores are normalized by the Milvus connector so larger values are better. For L2 search, the connector converts distance to a negative score.

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--fixture` | required | Synthetic fixture JSON file. |
| `--address` | required | Milvus gRPC address, for example `localhost:19530`. |
| `--collection` | `items` | Milvus collection to count and search. |
| `--id-field` | `id` | Primary key field returned in search results. |
| `--vector-field` | `embedding` | FloatVector field to search. |
| `--top-k` | `3` | Business-visible topK result count. |
| `--expand-k` | `5` | Expanded result count for boundary smoke checks. Must be `>= top-k`. |
| `--query-index` | `0` | Zero-based fixture query index. |
| `--metric` | `cosine` | Search metric, currently `cosine` or `l2`. |

## Current limitations

- Searches one fixture query only.
- Does not build fingerprint artifacts; use the upcoming Milvus artifact CLI for all-query artifact generation.
- Depends on a collection previously created by `seed-milvus` or equivalent setup.
