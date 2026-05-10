package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PythonRunner invokes the Python retrieval behavior fingerprint engine through
// a subprocess. It preserves the Engine interface so the Go control plane can
// later replace subprocess execution with gRPC, HTTP, or a native Go engine
// without changing job orchestration code.
type PythonRunner struct {
	// PythonPath is the Python executable used to run the installed fingerprint engine module.
	PythonPath string
	// Module is the Python module executed with `python -m`; defaults to vdb_fingerprint_engine.cli.
	Module string
	// WorkDir is the working directory for the Python subprocess.
	WorkDir string
}

type compareInputJSON struct {
	JobID                 string `json:"job_id"`
	SourceFingerprintPath string `json:"source_fingerprint_path"`
	TargetFingerprintPath string `json:"target_fingerprint_path"`
}

type compareOutputJSON struct {
	JobID            string            `json:"job_id"`
	ConsistencyScore float64           `json:"consistency_score"`
	Metrics          metricSummaryJSON `json:"metrics"`
}

type metricSummaryJSON struct {
	FingerprintDistance float64 `json:"fingerprint_distance"`
	BoundaryFlipRate    float64 `json:"boundary_flip_rate"`
}

// NewPythonRunner creates a Python subprocess-backed Engine implementation. The
// caller supplies the Python executable and working directory so local uv-managed
// environments, production virtual environments, and packaged deployments can
// all use the same runner without hardcoded machine-specific paths.
func NewPythonRunner(pythonPath string, workDir string) PythonRunner {
	return PythonRunner{
		PythonPath: pythonPath,
		Module:     "vdb_fingerprint_engine.cli",
		WorkDir:    workDir,
	}
}

// Compare writes a JSON CompareInput payload, invokes the Python engine compare
// command, and reads the resulting JSON CompareOutput payload. It honors context
// cancellation through exec.CommandContext and includes captured stderr/stdout in
// errors so failed engine runs are diagnosable.
func (r PythonRunner) Compare(ctx context.Context, input CompareInput) (CompareOutput, error) {
	if r.PythonPath == "" {
		return CompareOutput{}, fmt.Errorf("python path must not be empty")
	}
	if _, err := os.Stat(r.PythonPath); err != nil {
		return CompareOutput{}, fmt.Errorf("stat python path %q: %w", r.PythonPath, err)
	}

	module := r.Module
	if module == "" {
		module = "vdb_fingerprint_engine.cli"
	}

	tempDir, err := os.MkdirTemp("", "vdb-guardian-engine-*")
	if err != nil {
		return CompareOutput{}, fmt.Errorf("create engine temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, "input.json")
	outputPath := filepath.Join(tempDir, "output.json")
	if err := writeCompareInput(inputPath, input); err != nil {
		return CompareOutput{}, err
	}

	cmd := exec.CommandContext(ctx, r.PythonPath, "-m", module, "compare", "--input", inputPath, "--output", outputPath)
	if r.WorkDir != "" {
		cmd.Dir = r.WorkDir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return CompareOutput{}, fmt.Errorf("run python engine compare canceled: %w", ctxErr)
		}
		return CompareOutput{}, fmt.Errorf("run python engine compare: %w: stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}

	output, err := readCompareOutput(outputPath)
	if err != nil {
		return CompareOutput{}, err
	}
	return output, nil
}

// writeCompareInput serializes the Go CompareInput into the snake_case JSON
// protocol consumed by the Python fingerprint engine CLI.
func writeCompareInput(path string, input CompareInput) error {
	payload := compareInputJSON{
		JobID:                 input.JobID,
		SourceFingerprintPath: input.SourceFingerprintPath,
		TargetFingerprintPath: input.TargetFingerprintPath,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal engine input: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write engine input %q: %w", path, err)
	}
	return nil
}

// readCompareOutput deserializes the Python engine's snake_case JSON response
// into the Go CompareOutput type used by job orchestration and reporting.
func readCompareOutput(path string) (CompareOutput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CompareOutput{}, fmt.Errorf("read engine output %q: %w", path, err)
	}
	var payload compareOutputJSON
	if err := json.Unmarshal(data, &payload); err != nil {
		return CompareOutput{}, fmt.Errorf("decode engine output %q: %w", path, err)
	}
	return CompareOutput{
		JobID:            payload.JobID,
		ConsistencyScore: payload.ConsistencyScore,
		Metrics: MetricSummary{
			FingerprintDistance: payload.Metrics.FingerprintDistance,
			BoundaryFlipRate:    payload.Metrics.BoundaryFlipRate,
		},
	}, nil
}
