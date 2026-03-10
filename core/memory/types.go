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
