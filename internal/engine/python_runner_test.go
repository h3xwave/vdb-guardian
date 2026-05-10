package engine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPythonRunnerCompareInvokesPythonEngine(t *testing.T) {
	repoRoot := repositoryRoot(t)
	pythonPath := filepath.Join(repoRoot, "python", ".venv", "bin", "python")
	if _, err := os.Stat(pythonPath); err != nil {
		t.Fatalf("expected python virtual environment to exist at %s: %v", pythonPath, err)
	}

	runner := NewPythonRunner(pythonPath, filepath.Join(repoRoot, "python"))
	output, err := runner.Compare(context.Background(), CompareInput{
		JobID:                 "job-1",
		SourceFingerprintPath: "source.json",
		TargetFingerprintPath: "target.json",
	})
	if err != nil {
		t.Fatalf("expected python runner compare to succeed: %v", err)
	}

	if output.JobID != "job-1" {
		t.Fatalf("expected job id job-1, got %q", output.JobID)
	}
	if output.ConsistencyScore < 0 || output.ConsistencyScore > 1 {
		t.Fatalf("expected consistency score in [0,1], got %f", output.ConsistencyScore)
	}
	if output.Metrics.FingerprintDistance != 0 {
		t.Fatalf("expected minimal protocol fingerprint distance 0, got %f", output.Metrics.FingerprintDistance)
	}
}

func TestPythonRunnerCompareReturnsErrorWhenPythonPathMissing(t *testing.T) {
	runner := NewPythonRunner("/tmp/vdb-guardian-missing-python", t.TempDir())

	_, err := runner.Compare(context.Background(), CompareInput{JobID: "job-1"})
	if err == nil {
		t.Fatal("expected missing python path to return an error")
	}
	if !strings.Contains(err.Error(), "python") {
		t.Fatalf("expected error to mention python path, got %v", err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("expected runtime caller to locate test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}
