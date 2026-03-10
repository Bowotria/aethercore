package memory

import (
	"context"
	"errors"
	"sync"
)

// ZestDBStorage is a lightweight, high-performance persistence layer.
// In this Layer 0 implementation, it uses a thread-safe map as a proxy for the Rust-based ZestDB.
type ZestDBStorage struct {
	mu   sync.RWMutex
	data map[string]MemoryEntry
}

// NewZestDBStorage initializes a new instance of the ZestDB adapter.
func NewZestDBStorage() *ZestDBStorage {
	return &ZestDBStorage{
		data: make(map[string]MemoryEntry),
	}
}

// Put saves a new entry to the in-memory persistence layer.
func (s *ZestDBStorage) Put(ctx context.Context, entry MemoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.ID == "" {
		return errors.New("memory_entry_id_required")
	}

	s.data[entry.ID] = entry
	return nil
}
