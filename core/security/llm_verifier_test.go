package security

import (
	"context"
	"testing"
)

func TestLLMVerifier_Safe(t *testing.T) {
	verifier := NewLLMVerifier()
	res := verifier.Scan(context.Background(), "test", GuardConfig{})
	if !res.IsSafe {
		t.Errorf("Expected IsSafe=true")
	}
}

func TestLLMVerifier_ParseRejection(t *testing.T) {
	verifier := NewLLMVerifier()
	res := verifier.parseResponse(`{"is_safe": false, "reason": "Jailbreak attempt detected"}`)
	if res.IsSafe {
		t.Errorf("Expected IsSafe=false")
	}
	if len(res.Violations) == 0 {
		t.Errorf("Expected violation reason to be captured")
	}
}
