package audit

import "context"

// Logger defines the interface for the cryptographically linked append-only audit trail.
type Logger interface {
	LogEvent(ctx context.Context, event *Event) error
	VerifyChain() (bool, error)
}
