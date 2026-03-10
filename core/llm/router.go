package llm

import (
	"context"
)

// Router defines the logic for selecting the most appropriate LLM provider
// based on task complexity, cost, and availability.
type Router interface {
	// Select returns the best LLM provider for the given context.
	Select(ctx context.Context, task string) (Provider, error)
}

// Provider is a specialized interface that the Router works with.
// It abstracts the underlying LLMAdapter to add routing-specific metadata.
type Provider interface {
	Name() string
	Status() string // "healthy", "degraded", "offline"
	Priority() int  // 0 is highest
}
