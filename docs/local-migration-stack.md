# Local Milvus to pgvector Migration Stack

The local migration stack provides the database services needed for the first Milvus to pgvector migration-and-verification MVP. It is intentionally local-only and should not be used as production infrastructure.

## Services

The stack is defined in:

```text
deployments/docker-compose.migration.yml
```

It includes:

- `milvus-etcd`: etcd dependency for Milvus standalone.
- `milvus-minio`: object storage dependency for Milvus standalone.
- `milvus-standalone`: Milvus source vector database.
- `postgres-pgvector`: PostgreSQL target database with pgvector initialized.

## Ports

Default host ports:

| Service | Port | Purpose |
| --- | ---: | --- |
| Milvus | 19530 | gRPC SDK endpoint |
| Milvus | 9091 | HTTP health/metrics endpoint |
| PostgreSQL | 5432 | PostgreSQL endpoint |
| MinIO | 9000 | S3-compatible API |
| MinIO | 9001 | MinIO console |
| etcd | 2379 | etcd client endpoint |

Ports can be overridden with environment variables such as:

```bash
VDB_GUARDIAN_POSTGRES_PORT=15432 \
VDB_GUARDIAN_MILVUS_PORT=19531 \
docker compose -f deployments/docker-compose.migration.yml up -d
```

## Local-only credentials

The PostgreSQL service uses local-only defaults:

```text
POSTGRES_DB=vdb_guardian
POSTGRES_USER=vdb_guardian
POSTGRES_PASSWORD=vdb_guardian_local_password
```

These values are not production credentials. Do not reuse them outside local development.

## Validate without starting containers

```bash
make migration-stack-config
```

or:

```bash
scripts/check-migration-stack.sh config
```

This only validates the Compose file and does not create containers, networks, or volumes.

## Start the stack

Starting Docker has side effects: it creates containers, networks, and volumes.

```bash
make migration-stack-up
```

Equivalent command:

```bash
docker compose -f deployments/docker-compose.migration.yml up -d
```

## Check status

```bash
make migration-stack-status
```

## Health checks

After the stack is running, verify PostgreSQL and pgvector:

```bash
scripts/check-migration-stack.sh postgres
```

Verify the Milvus gRPC port:

```bash
scripts/check-migration-stack.sh milvus-port
```

## Stop and remove containers

```bash
make migration-stack-down
```

This removes containers and the Compose network. Named volumes are preserved unless removed manually.

## pgvector seed smoke check

After the PostgreSQL pgvector service is healthy, the target-side seed loop can be verified with:

```bash
go run ./cmd/vdbg seed-pgvector \
  --fixture testdata/migration/synthetic-small.json \
  --connection-url '[REDACTED]' \
  --table items \
  --id-column id \
  --vector-column embedding
```

Then verify the seeded row count and vector dimensions:

```bash
docker compose -f deployments/docker-compose.migration.yml exec -T postgres-pgvector psql \
  -U vdb_guardian \
  -d vdb_guardian \
  -c "SELECT COUNT(*) AS seeded_records FROM items; SELECT id, vector_dims(embedding) AS dims FROM items ORDER BY id LIMIT 3;"
```

For the committed small fixture, the expected row count is `100` and the vector dimension is `8`.

## Current limitations

This stack now supports validating the pgvector target-side seed loop. It does not yet seed Milvus, run Milvus-to-pgvector migrations, or execute the full migrate-and-verify workflow. Those capabilities will be added in the migration MVP steps that follow.