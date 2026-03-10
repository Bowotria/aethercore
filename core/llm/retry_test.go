package llm

import (
	"context"
	"errors"
	"testing"
)

type ErrorMockProvider struct {
	MockProvider
	failCount int
	calls     int
}

func (m *ErrorMockProvider) Execute(ctx context.Context, task string) (string, error) {
	m.calls++
	if m.calls <= m.failCount {
		return "", errors.New("rate limit exceeded (429)")
	}
	return "success", nil
}

func TestRetryingProvider_Execute(t *testing.T) {
	t.Run("retries on temporary failure and eventually succeeds", func(t *testing.T) {
		base := &ErrorMockProvider{
			MockProvider: MockProvider{name: "gpt-4", status: StatusHealthy},
			failCount:    2,
		}

		// This should be a wrapper around LLMAdapter logic
		// For now we test the concept of retrying a provider call
		retryer := NewRetryingProvider(base, 3)
		got, err := retryer.Execute(context.Background(), "hello")
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if base.calls != 3 {
			t.Errorf("expected 3 calls, got %d", base.calls)
		}
		if got != "success" {
			t.Errorf("expected success, got %s", got)
		}
	})

	t.Run("fails after max retries", func(t *testing.T) {
		base := &ErrorMockProvider{
			MockProvider: MockProvider{name: "gpt-4", status: StatusHealthy},
			failCount:    5,
		}

		retryer := NewRetryingProvider(base, 3)
		_, err := retryer.Execute(context.Background(), "hello")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if base.calls != 3 {
			t.Errorf("expected 3 calls, got %d", base.calls)
		}
	})
}
