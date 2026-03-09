package security

import "testing"

func TestKeyRing_LoadKey(t *testing.T) {
	kr := NewKeyRing()
	if len(kr.trustedKeys) != 0 {
		t.Errorf("Expected 0 keys")
	}
}
