# compare-artifacts CLI

`vdbg compare-artifacts` compares existing source and target fingerprint artifacts through the Python fingerprint engine and writes a normalized result artifact.

It is the first real database artifact comparison bridge for the local migration MVP:

```text
Milvus source fingerprint artifact
  +
pgvector target fingerprint artifact
  ->
Python artifact comparison engine
  ->
<artifact-dir>/<job-id>-result.json
```

## When to use it

Use this command after both sides have produced Python-compatible fingerprint artifacts:

- source artifact from `vdbg build-milvus-artifact`
- target artifact from `vdbg build-pgvector-artifact`

The command does not connect to Milvus or PostgreSQL. It only reads local JSON artifact files and runs the Python engine through `internal/engine.PythonRunner`.

## Example

```bash
go run ./cmd/vdbg compare-artifacts \
  --source /tmp/vdb-guardian-source-fingerprint.json \
  --target /tmp/vdb-guardian-target-fingerprint.json \
  --artifact-dir /tmp/vdb-guardian-compare \
  --job-id real-artifact-smoke
```

Expected output shape:

```text
artifact comparison completed
job_id: real-artifact-smoke
consistency_score: 1.000000
fingerprint_distance: 0.000000
stable_neighbor_distance: 0.000000
boundary_candidate_distance: 0.000000
boundary_flip_rate: 0.000000
matched_queries: 10
missing_source_queries: 0
missing_target_queries: 0
source_fingerprint: /tmp/vdb-guardian-source-fingerprint.json
target_fingerprint: /tmp/vdb-guardian-target-fingerprint.json
result: /tmp/vdb-guardian-compare/real-artifact-smoke-result.json
```

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--source` | required | Path to the source fingerprint artifact JSON. |
| `--target` | required | Path to the target fingerprint artifact JSON. |
| `--artifact-dir` | required | Directory where the result artifact is written. |
| `--job-id` | `artifact-compare` | Job ID used in the result artifact filename. |

## Result artifact

The result is written to:

```text
<artifact-dir>/<job-id>-result.json
```

The JSON includes:

- `job_id`
- `state`
- `consistency_score`
- `metrics.fingerprint_distance`
- `metrics.stable_neighbor_distance`
- `metrics.boundary_candidate_distance`
- `metrics.boundary_flip_rate`
- `metrics.matched_query_count`
- `metrics.missing_source_query_count`
- `metrics.missing_target_query_count`

## Validation

The command validates before running the Python engine:

- source path is present and points to a file
- target path is present and points to a file
- artifact directory is present
- job ID is non-empty

The Python engine performs schema-level artifact validation, duplicate query checks, missing-query penalties, and metric aggregation.

## Limitations

- This is a comparison command, not a migration command.
- It assumes the artifact files were generated with compatible `top-k`, `stable-k`, and `boundary-k` settings.
- It does not yet produce a full human-readable migration report.
- It relies on the local Python environment discovered by the same mechanism as `offline-verify`.
