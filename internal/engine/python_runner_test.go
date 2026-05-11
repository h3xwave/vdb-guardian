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

	// Determine Python executable path based on OS
	var pythonPath string
	if runtime.GOOS == "windows" {
		pythonPath = filepath.Join(repoRoot, "python", ".venv", "Scripts", "python.exe")
	} else {
		pythonPath = filepath.Join(repoRoot, "python", ".venv", "bin", "python")
	}

	if _, err := os.Stat(pythonPath); err != nil {
		t.Skipf("skipping test: python virtual environment not found at %s", pythonPath)
	}

	artifactDir := t.TempDir()
	sourcePath := filepath.Join(artifactDir, "source.json")
	targetPath := filepath.Join(artifactDir, "target.json")
	writeFingerprintArtifact(t, sourcePath, `{"fingerprints":[{"query_id":"q-1","stable_neighbors":["a","b","c"],"boundary_candidates":["d","e"],"top_k_ids":["a","b","c","d"]}]}`)
	writeFingerprintArtifact(t, targetPath, `{"fingerprints":[{"query_id":"q-1","stable_neighbors":["a","b","x"],"boundary_candidates":["d","f"],"top_k_ids":["a","b","x","f"]}]}`)

	runner := NewPythonRunner(pythonPath, filepath.Join(repoRoot, "python"))
	output, err := runner.Compare(context.Background(), CompareInput{
		JobID:                 "job-1",
		SourceFingerprintPath: sourcePath,
		TargetFingerprintPath: targetPath,
	})
	if err != nil {
		t.Fatalf("expected python runner compare to succeed: %v", err)
	}

	if output.JobID != "job-1" {
		t.Fatalf("expected job id job-1, got %q", output.JobID)
	}
	if output.ConsistencyScore >= 1 || output.ConsistencyScore <= 0 {
		t.Fatalf("expected changed artifacts to produce score in (0,1), got %f", output.ConsistencyScore)
	}
	if output.Metrics.FingerprintDistance <= 0 {
		t.Fatalf("expected artifact-backed fingerprint distance > 0, got %f", output.Metrics.FingerprintDistance)
	}
	if output.Metrics.BoundaryFlipRate <= 0 {
		t.Fatalf("expected artifact-backed boundary flip rate > 0, got %f", output.Metrics.BoundaryFlipRate)
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

func writeFingerprintArtifact(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fingerprint artifact %s: %v", path, err)
	}
}
