# search-pgvector CLI

`vdbg search-pgvector` runs a target-side pgvector search smoke check against records that were seeded from a synthetic fixture.

The command is intentionally narrow. It validates that the real pgvector connector can connect, count rows, execute a normalized search, and print ranked hits for one fixture query. It does not start Docker and does not write data.

## Example

```bash
go run ./cmd/vdbg search-pgvector \
  --fixture testdata/migration/synthetic-small.json \
  --connection-url '[REDACTED]' \
  --table items \
  --top-k 3 \
  --expand-k 5 \
  --query-index 0 \
  --metric cosine
```

## Options

| Option | Required | Default | Description |
| --- | --- | --- | --- |
| `--fixture` | yes | none | Synthetic fixture JSON containing query vectors. |
| `--connection-url` | yes | none | PostgreSQL connection URL for the pgvector database. |
| `--table` | no | `items` | Table to count and search. |
| `--top-k` | no | `3` | Business-visible topK result count. |
| `--expand-k` | no | `5` | Expanded result count used for boundary smoke checks. Must be `>= top-k`. |
| `--query-index` | no | `0` | Zero-based query index from the fixture. |
| `--metric` | no | `cosine` | pgvector metric: `cosine` or `l2`. |

## Output

The command prints a compact, human-readable summary:

```text
pgvector search smoke ok
fixture: testdata/migration/synthetic-small.json
table: items
records_count: 100
query_id: query-000001
top_k: 3
expand_k: 5
hits: 5
hit rank=1 id=vec-000084 score=0.8460551500320435
```

`hits` follows `expand-k`, not `top-k`, because the fingerprint builder needs the expanded boundary window.

## Local migration stack smoke check

After explicit approval to run Docker side effects, start or reuse the local pgvector service and seed it first:

```bash
docker compose -f deployments/docker-compose.migration.yml up -d postgres-pgvector
scripts/check-migration-stack.sh postgres
go run ./cmd/vdbg seed-pgvector \
  --fixture testdata/migration/synthetic-small.json \
  --connection-url '[REDACTED]' \
  --table items \
  --id-column id \
  --vector-column embedding
```

Then run the search smoke command:

```bash
go run ./cmd/vdbg search-pgvector \
  --fixture testdata/migration/synthetic-small.json \
  --connection-url '[REDACTED]' \
  --table items \
  --top-k 3 \
  --expand-k 5 \
  --query-index 0 \
  --metric cosine
```

For the committed small fixture, the expected count is `100` and the command should print `5` hits when `--expand-k 5` is used.

## Safety

- Connection URLs are runtime-only and must not be committed.
- The command performs reads only: pgvector extension check, row count, and vector search.
- Docker is never started implicitly.
- The command verifies target-side pgvector search only; it does not prove Milvus source readiness or end-to-end migration correctness.

## Current limitations

Implemented:

- Real pgx-backed pgvector connector usage.
- Synthetic fixture query loading.
- Row count smoke check.
- Single-query normalized vector search.
- Unit tests through an injected connector factory.

Not yet implemented:

- Multi-query batch smoke checks.
- Fingerprint artifact writing from real pgvector results.
- Automated Docker integration test.
- Milvus source-side real search smoke check.
- End-to-end migrate-and-verify command.
