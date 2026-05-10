package artifacts

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var errEmptyArtifactPath = errors.New("artifact path must not be empty")

// Store defines the storage boundary for generated fingerprints, comparison
// inputs, reports, and other job artifacts. Production implementations can write
// to local disk, S3, MinIO, or other object stores while callers keep the same
// interface.
type Store interface {
	// Put stores data at path, replacing any previous value for that path.
	Put(ctx context.Context, path string, data []byte) error
	// Get returns a copy of the data stored at path or an error when the artifact is missing.
	Get(ctx context.Context, path string) ([]byte, error)
	// Exists reports whether an artifact path has been stored.
	Exists(ctx context.Context, path string) (bool, error)
}

// MemoryStore is an in-memory Store implementation used by unit tests and early
// scaffolding. It copies data on write and read so callers cannot mutate internal
// state through shared byte slices.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemoryStore creates an empty in-memory artifact store. The store is safe for
// concurrent use by tests or lightweight local runners because all access is
// protected by a read-write mutex.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string][]byte)}
}

// Put stores a copy of data under path. It returns an error for empty paths so
// job runners do not accidentally write artifacts that cannot be addressed later.
func (s *MemoryStore) Put(ctx context.Context, path string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if path == "" {
		return errEmptyArtifactPath
	}

	copyData := append([]byte(nil), data...)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[path] = copyData
	return nil
}

// Get returns a copy of the artifact stored at path. A missing path returns an
// error with the requested path so callers can include useful diagnostic context
// in job failure reports.
func (s *MemoryStore) Get(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, errEmptyArtifactPath
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[path]
	if !ok {
		return nil, fmt.Errorf("artifact %q not found", path)
	}
	return append([]byte(nil), value...), nil
}

// Exists reports whether path is present in the store. It validates empty paths
// consistently with Put and Get so all artifact operations share the same input
// contract.
func (s *MemoryStore) Exists(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if path == "" {
		return false, errEmptyArtifactPath
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[path]
	return ok, nil
}
