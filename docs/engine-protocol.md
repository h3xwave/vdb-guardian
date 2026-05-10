# Engine Protocol

The engine protocol defines how the Go control plane invokes the Python retrieval behavior fingerprint engine.

## Current execution mode

The current implementation uses a Python subprocess:

```text
Go PythonRunner
  -> python -m vdb_fingerprint_engine.cli compare --input input.json --output output.json
  -> JSON CompareOutput
```

This keeps the first enterprise deployment simple while preserving a stable boundary that can later evolve into gRPC, HTTP, or a remote Python service.

## Go runner

The Go implementation lives in `internal/engine/python_runner.go`.

```go
runner := engine.NewPythonRunner("/path/to/python", "/path/to/repo/python")
output, err := runner.Compare(ctx, engine.CompareInput{
    JobID: "job-1",
    SourceFingerprintPath: "source.json",
    TargetFingerprintPath: "target.json",
})
```

The runner:

1. Creates a temporary working directory.
2. Writes `input.json`.
3. Runs the Python CLI compare command.
4. Reads `output.json`.
5. Converts snake_case JSON into Go structs.
6. Returns diagnostic errors with subprocess output when execution fails.

## Input JSON

```json
{
  "job_id": "job-1",
  "source_fingerprint_path": "source.json",
  "target_fingerprint_path": "target.json"
}
```

Fields:

- `job_id`: stable verification job identifier.
- `source_fingerprint_path`: artifact path for source retrieval behavior fingerprints.
- `target_fingerprint_path`: artifact path for target retrieval behavior fingerprints.

## Output JSON

```json
{
  "job_id": "job-1",
  "consistency_score": 1.0,
  "metrics": {
    "fingerprint_distance": 0.0,
    "boundary_flip_rate": 0.0
  }
}
```

Fields:

- `job_id`: copied from the input payload.
- `consistency_score`: normalized score in `[0, 1]`; higher means more consistent.
- `metrics.fingerprint_distance`: normalized fingerprint distance.
- `metrics.boundary_flip_rate`: normalized boundary candidate topK flip rate.

## Current limitation

The current Python compare command validates the protocol and returns neutral perfect-consistency metrics. Artifact-backed fingerprint comparison will be implemented in a later step.

This is intentional: the current change proves the Go/Python execution boundary before concrete connector and artifact comparison logic is added.

## Security notes

- Do not place credentials in engine input JSON.
- Do not log production DSNs or secret-bearing artifact paths.
- The Go runner writes temporary input files with `0600` permissions.
- Temporary files are removed after each run.