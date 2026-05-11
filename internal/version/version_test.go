package version

import "testing"

func TestInfoReturnsStableProjectMetadata(t *testing.T) {
	info := Info()

	if info.Name != "vdb-guardian" {
		t.Fatalf("expected project name vdb-guardian, got %q", info.Name)
	}

	if info.Version == "" {
		t.Fatal("expected version to be non-empty")
	}
}
