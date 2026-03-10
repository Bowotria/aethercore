package llm

import (
	"context"
	"testing"
)

func TestLatencyRouter_Select(t *testing.T) {
	t.Run("selects fastest healthy provider", func(t *testing.T) {
		p1 := &MockProvider{
			name:     "slow-model",
			status:   StatusHealthy,
			metadata: ModelMetadata{LatencyMillis: 2000},
		}
		p2 := &MockProvider{
			name:     "fast-model",
			status:   StatusHealthy,
			metadata: ModelMetadata{LatencyMillis: 200},
		}

		router := NewLatencyRouter([]Provider{p1, p2})
		got, err := router.Select(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name() != "fast-model" {
			t.Errorf("expected fast-model, got %s", got.Name())
		}
	})

	t.Run("selects slow if fast is offline", func(t *testing.T) {
		p1 := &MockProvider{
			name:     "slow-model",
			status:   StatusHealthy,
			metadata: ModelMetadata{LatencyMillis: 2000},
		}
		p2 := &MockProvider{
			name:     "fast-model",
			status:   StatusOffline,
			metadata: ModelMetadata{LatencyMillis: 200},
		}

		router := NewLatencyRouter([]Provider{p1, p2})
		got, err := router.Select(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name() != "slow-model" {
			t.Errorf("expected slow-model, got %s", got.Name())
		}
	})
}
