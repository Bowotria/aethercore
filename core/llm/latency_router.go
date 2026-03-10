package llm

import (
	"context"
	"errors"
	"sort"
)

// LatencyRouter selects the provider with the lowest expected latency.
type LatencyRouter struct {
	providers []Provider
}

func NewLatencyRouter(providers []Provider) *LatencyRouter {
	return &LatencyRouter{providers: providers}
}

func (r *LatencyRouter) Select(ctx context.Context, task string) (Provider, error) {
	var healthy []Provider
	for _, p := range r.providers {
		if p.Status() == StatusHealthy {
			healthy = append(healthy, p)
		}
	}

	if len(healthy) == 0 {
		return nil, errors.New("no healthy LLM providers available")
	}

	// Sort by latency (ascending)
	sort.Slice(healthy, func(i, j int) bool {
		return healthy[i].Metadata().LatencyMillis < healthy[j].Metadata().LatencyMillis
	})

	return healthy[0], nil
}
