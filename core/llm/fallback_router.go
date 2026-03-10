package llm

import (
	"context"
	"errors"
	"sort"
)

// FallbackRouter selects the highest priority healthy provider.
type FallbackRouter struct {
	providers []Provider
}

// NewFallbackRouter creates a router that prioritizes providers based on their Priority rank.
func NewFallbackRouter(providers []Provider) *FallbackRouter {
	// Sort providers by priority (ascending: 1 is higher than 2)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Priority() < providers[j].Priority()
	})
	return &FallbackRouter{providers: providers}
}

// Select returns the first healthy provider according to priority.
func (r *FallbackRouter) Select(ctx context.Context, task string) (Provider, error) {
	for _, p := range r.providers {
		if p.Status() == StatusHealthy {
			return p, nil
		}
	}
	return nil, errors.New("no healthy LLM providers available")
}
