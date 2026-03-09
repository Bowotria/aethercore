package audit

import "time"

// Block represents a cryptographically secure envelope around an AuditEvent.
// Using a linked-list hash structure prevents silent tampering of historical events.
type Block struct {
	Index        uint64     `json:"index"`
	Timestamp    time.Time  `json:"timestamp"`
	Event        AuditEvent `json:"event"`
	PreviousHash string     `json:"previous_hash"`
	Hash         string     `json:"hash"`
}
