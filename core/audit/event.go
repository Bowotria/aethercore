package audit

import "time"

// Event represents a single immutable action within the system.
type Event struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Actor     string                 `json:"actor"`
	Metadata  map[string]interface{} `json:"metadata"`
}
