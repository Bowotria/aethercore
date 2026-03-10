package memory

import (
	"time"
)

// MemoryEntry represents a single unit of persistent episodic or semantic memory.
type MemoryEntry struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}

// SearchOptions defines filters for querying the memory store.
type SearchOptions struct {
	Limit     int
	Tags      []string
	MinScore  float64
	StartTime time.Time
	EndTime   time.Time
}
