package audit

import "testing"

func TestChainManager_Init(t *testing.T) {
	cm := NewChainManager()
	if cm == nil {
		t.Fatalf("expected non-nil manager")
	}
}
