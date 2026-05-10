# Fingerprint Engine Specification

The fingerprint engine computes retrieval behavior differences between source and target vector databases.

## Initial metrics

The first implementation includes:

- Boundary candidate selection.
- Jaccard distance.
- Boundary flip rate.
- Weighted fingerprint distance.

## Boundary candidates

Boundary candidates are hits near the topK decision boundary whose score is close to the K-th result. They are important because migration-related indexing, distance, or filtering differences often cause these candidates to enter or leave visible topK results.

## Distance metrics

The engine returns normalized values in `[0, 1]` where possible. Lower fingerprint distance means source and target retrieval behavior are more similar. Higher consistency score means better migration consistency.

## Protocol direction

The Go control plane invokes the Python engine with a subprocess runner and a JSON file protocol:

```text
python -m vdb_fingerprint_engine.cli compare --input input.json --output output.json
```

The Python engine returns a compact JSON summary and will write detailed artifacts through the artifact boundary in later phases.

See `docs/engine-protocol.md` for the current schema.

## Current compare command

The current `compare` command validates input/output wiring and returns neutral perfect-consistency metrics. It does not yet read source and target fingerprint artifacts. Artifact-backed comparison is the next algorithm implementation step.
