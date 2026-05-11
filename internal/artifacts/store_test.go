package artifacts

import (
	"context"
	"testing"
)

func TestMemoryStorePutGetAndExists(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	if err := store.Put(ctx, "jobs/job-1/report.md", []byte("report")); err != nil {
		t.Fatalf("expected put to succeed: %v", err)
	}

	exists, err := store.Exists(ctx, "jobs/job-1/report.md")
	if err != nil {
		t.Fatalf("expected exists to succeed: %v", err)
	}
	if !exists {
		t.Fatal("expected artifact to exist after put")
	}

	content, err := store.Get(ctx, "jobs/job-1/report.md")
	if err != nil {
		t.Fatalf("expected get to succeed: %v", err)
	}
	if string(content) != "report" {
		t.Fatalf("expected report content, got %q", string(content))
	}
}

func TestMemoryStoreRejectsEmptyPath(t *testing.T) {
	store := NewMemoryStore()

	if err := store.Put(context.Background(), "", []byte("invalid")); err == nil {
		t.Fatal("expected empty path to return an error")
	}
}
