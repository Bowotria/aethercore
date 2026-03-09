package security

import (
	"context"
	"testing"
)

func TestOrchestratorGuard_ShortCircuit(t *testing.T) {
	failScanner := &MockPromptGuard{ForceFail: true, MockReason: "First scanner failed"}
	safeScanner := &MockPromptGuard{ForceFail: false}

	orchestrator := NewOrchestratorGuard(failScanner, safeScanner)
	res := orchestrator.Scan(context.Background(), "trigger failure", GuardConfig{})

	if res.IsSafe {
		t.Errorf("Expected IsSafe=false due to first scanner rejecting it")
	}
}
