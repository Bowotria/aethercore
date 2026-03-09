package security

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

func TestManifestValidator_MissingSignature(t *testing.T) {
	kr := NewKeyRing()
	validator := NewManifestValidator(kr)
	ok, err := validator.Verify([]byte("{}"), "")
	if ok || err == nil {
		t.Errorf("Expected error for missing signature")
	}
}

func TestManifestValidator_InvalidHexEncoding(t *testing.T) {
	kr := NewKeyRing()
	validator := NewManifestValidator(kr)
	ok, err := validator.Verify([]byte("{}"), "not-a-hex-string!")
	if ok || err == nil {
		t.Errorf("Expected error for invalid hex encoding")
	}
}

func TestManifestValidator_CanonicalJSONSerialization(t *testing.T) {
	kr := NewKeyRing()
	validator := NewManifestValidator(kr)

	raw := []byte(`{"z":1,"a":2}`)
	canonical, err := validator.canonicalize(raw)
	if err != nil {
		t.Errorf("Expected canonicalize to succeed")
	}
	if string(canonical) != `{"a":2,"z":1}` {
		t.Errorf("Expected strictly alphabetical JSON keys")
	}
}

func TestManifestValidator_SignatureMismatch(t *testing.T) {
	kr := NewKeyRing()
	validator := NewManifestValidator(kr)

	sig := "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	ok, err := validator.Verify([]byte(`{"a":2}`), sig)
	if ok || err == nil {
		t.Errorf("Expected signature mismatch error")
	}
}

func TestManifestValidator_ValidSignatureApproval(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	kr := NewKeyRing()
	kr.trustedKeys = append(kr.trustedKeys, pub)

	validator := NewManifestValidator(kr)

	msg := []byte(`{"a":2,"z":1}`)
	canon, _ := validator.canonicalize(msg)
	sigBytes := ed25519.Sign(priv, canon)
	sigHex := hex.EncodeToString(sigBytes)

	ok, err := validator.Verify(msg, sigHex)
	if !ok || err != nil {
		t.Errorf("Expected valid signature to be approved, got error: %v", err)
	}
}
