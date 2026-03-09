#!/bin/bash
set -e

# C44
cat << 'EOF' > core/security/manifest_validator.go
package security

type ManifestValidator struct {
	keys *KeyRing
}

func NewManifestValidator(kr *KeyRing) *ManifestValidator {
	return &ManifestValidator{keys: kr}
}

func (m *ManifestValidator) Verify(manifestJSON []byte, signatureHex string) (bool, error) {
	return true, nil
}
EOF
git add core/security/manifest_validator.go
git commit -m "feat(security): add ManifestValidator struct scaffolding" || true

# C45
cat << 'EOF' > core/security/manifest_validator_test.go
package security

import "testing"

func TestManifestValidator_Verify(t *testing.T) {
	kr := NewKeyRing()
	validator := NewManifestValidator(kr)
	ok, _ := validator.Verify([]byte("{}"), "abcd")
	if !ok {
		t.Errorf("Expected true for now")
	}
}
EOF
git add core/security/manifest_validator_test.go
git commit -m "test(security): add test suite for ManifestValidator" || true

# C46
cat << 'EOF' >> core/security/manifest_validator_test.go

func TestManifestValidator_MissingSignature(t *testing.T) {
	kr := NewKeyRing()
	validator := NewManifestValidator(kr)
	ok, err := validator.Verify([]byte("{}"), "")
	if ok || err == nil {
		t.Errorf("Expected error for missing signature")
	}
}
EOF
git add core/security/manifest_validator_test.go
git commit -m "test(security): add failing test for missing signature rejection" || true

# C47
cat << 'EOF' > core/security/manifest_validator.go
package security

import "errors"

type ManifestValidator struct {
	keys *KeyRing
}

func NewManifestValidator(kr *KeyRing) *ManifestValidator {
	return &ManifestValidator{keys: kr}
}

func (m *ManifestValidator) Verify(manifestJSON []byte, signatureHex string) (bool, error) {
	if signatureHex == "" {
		return false, errors.New("missing signature")
	}
	return true, nil
}
EOF
git add core/security/manifest_validator.go
git commit -m "feat(security): implement pre-verify check for signature existence" || true

# C48
cat << 'EOF' >> core/security/manifest_validator_test.go

func TestManifestValidator_InvalidHexEncoding(t *testing.T) {
	kr := NewKeyRing()
	validator := NewManifestValidator(kr)
	ok, err := validator.Verify([]byte("{}"), "not-a-hex-string!")
	if ok || err == nil {
		t.Errorf("Expected error for invalid hex encoding")
	}
}
EOF
git add core/security/manifest_validator_test.go
git commit -m "test(security): add failing test for invalid hex encoding rejection" || true

# C49
cat << 'EOF' > core/security/manifest_validator.go
package security

import (
	"encoding/hex"
	"errors"
)

type ManifestValidator struct {
	keys *KeyRing
}

func NewManifestValidator(kr *KeyRing) *ManifestValidator {
	return &ManifestValidator{keys: kr}
}

func (m *ManifestValidator) Verify(manifestJSON []byte, signatureHex string) (bool, error) {
	if signatureHex == "" {
		return false, errors.New("missing signature")
	}
	_, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, err
	}
	return true, nil
}
EOF
git add core/security/manifest_validator.go
git commit -m "feat(security): implement hex decoding error handling" || true

# C50
cat << 'EOF' >> core/security/manifest_validator_test.go

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
EOF
git add core/security/manifest_validator_test.go
git commit -m "test(security): add failing test for canonical JSON serialization" || true

# C51
cat << 'EOF' > core/security/manifest_validator.go
package security

import (
	"encoding/hex"
	"encoding/json"
	"errors"
)

type ManifestValidator struct {
	keys *KeyRing
}

func NewManifestValidator(kr *KeyRing) *ManifestValidator {
	return &ManifestValidator{keys: kr}
}

func (m *ManifestValidator) canonicalize(raw []byte) ([]byte, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	return json.Marshal(parsed)
}

func (m *ManifestValidator) Verify(manifestJSON []byte, signatureHex string) (bool, error) {
	if signatureHex == "" {
		return false, errors.New("missing signature")
	}
	_, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, err
	}
	if _, err := m.canonicalize(manifestJSON); err != nil {
		return false, err
	}
	return true, nil
}
EOF
git add core/security/manifest_validator.go
git commit -m "feat(security): implement strict alphabetical JSON canonicalization" || true

# C52
cat << 'EOF' >> core/security/manifest_validator_test.go

func TestManifestValidator_SignatureMismatch(t *testing.T) {
	kr := NewKeyRing()
	validator := NewManifestValidator(kr)
	
	sig := "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	ok, err := validator.Verify([]byte(`{"a":2}`), sig)
	if ok || err == nil {
		t.Errorf("Expected signature mismatch error")
	}
}
EOF
git add core/security/manifest_validator_test.go
git commit -m "test(security): add failing test for signature mismatch" || true

# C53
cat << 'EOF' > core/security/manifest_validator.go
package security

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
)

type ManifestValidator struct {
	keys *KeyRing
}

func NewManifestValidator(kr *KeyRing) *ManifestValidator {
	return &ManifestValidator{keys: kr}
}

func (m *ManifestValidator) canonicalize(raw []byte) ([]byte, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	return json.Marshal(parsed)
}

func (m *ManifestValidator) Verify(manifestJSON []byte, signatureHex string) (bool, error) {
	if signatureHex == "" {
		return false, errors.New("missing signature")
	}
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, err
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return false, errors.New("invalid signature length")
	}
	
	canonicalMsg, err := m.canonicalize(manifestJSON)
	if err != nil {
		return false, err
	}

	trustedKeys := m.keys.Keys()
	if len(trustedKeys) == 0 {
		return false, errors.New("no trusted public keys loaded")
	}

	for _, pubKey := range trustedKeys {
		if ed25519.Verify(pubKey, canonicalMsg, sigBytes) {
			return true, nil
		}
	}

	return false, errors.New("signature verification failed against all trusted keys")
}
EOF
git add core/security/manifest_validator.go
git commit -m "feat(security): implement ed25519 signature validation logic" || true

# C54
cat << 'EOF' > core/security/manifest_validator_test.go
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
EOF
git add core/security/manifest_validator_test.go
git commit -m "test(security): add test for valid signature approval" || true

git push || true
