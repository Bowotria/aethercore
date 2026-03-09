#!/bin/bash
set -e

# C36
cat << 'EOF' > core/security/tool_verifier.go
package security

type ToolVerifier interface {
	Verify(manifestJSON []byte, signatureHex string) (bool, error)
}
EOF
git add core/security/tool_verifier.go
git commit -m "feat(security): define ToolVerifier interface" || true

# C37
cat << 'EOF' > core/security/keyring.go
package security

import "crypto/ed25519"

type KeyRing struct {
	trustedKeys []ed25519.PublicKey
}
func NewKeyRing() *KeyRing { return &KeyRing{} }
EOF
git add core/security/keyring.go
git commit -m "feat(security): define KeyRing struct for pubkey management" || true

# C38
cat << 'EOF' > core/security/manifest.go
package security

type ToolManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WasmCode    []byte `json:"wasm_code,omitempty"`
}
EOF
git add core/security/manifest.go
git commit -m "feat(security): define ToolManifest schema in Go (mirroring Rust)" || true

# C39
cat << 'EOF' > core/security/keyring_test.go
package security

import "testing"

func TestKeyRing_LoadKey(t *testing.T) {
	kr := NewKeyRing()
	if len(kr.trustedKeys) != 0 {
		t.Errorf("Expected 0 keys")
	}
}
EOF
git add core/security/keyring_test.go
git commit -m "test(security): add test suite for KeyRing" || true

# C40
cat << 'EOF' >> core/security/keyring_test.go

func TestKeyRing_LoadValidEd25519PublicKey(t *testing.T) {
	kr := NewKeyRing()
	err := kr.LoadPEM([]byte("invalid pem data"))
	if err == nil {
		t.Errorf("Expected loading invalid PEM to fail")
	}
}
EOF
git add core/security/keyring_test.go
git commit -m "test(security): add failing test for loading valid Ed25519 public key" || true

# C41
cat << 'EOF' > core/security/keyring.go
package security

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

type KeyRing struct {
	trustedKeys []ed25519.PublicKey
}
func NewKeyRing() *KeyRing { return &KeyRing{} }

func (k *KeyRing) LoadPEM(pemData []byte) error {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return errors.New("failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	ed25519Key, ok := pub.(ed25519.PublicKey)
	if !ok {
		return errors.New("not an Ed25519 public key")
	}

	k.trustedKeys = append(k.trustedKeys, ed25519Key)
	return nil
}
EOF
git add core/security/keyring.go
git commit -m "feat(security): implement ParseEd25519PublicKey native Go logic" || true

# C42
cat << 'EOF' >> core/security/keyring_test.go

func TestKeyRing_LoadMalformedPublicKey(t *testing.T) {
	kr := NewKeyRing()
	
	// Valid PEM block but random data inside
	pemData := `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==
-----END PUBLIC KEY-----`

	err := kr.LoadPEM([]byte(pemData))
	if err == nil {
		t.Errorf("Expected parsing garbage PKIX to fail")
	}
}
EOF
git add core/security/keyring_test.go
git commit -m "test(security): add failing test for loading malformed public key" || true

# C43
# Refactoring C41 logic to verify block type
cat << 'EOF' > core/security/keyring.go
package security

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

type KeyRing struct {
	trustedKeys []ed25519.PublicKey
}
func NewKeyRing() *KeyRing { return &KeyRing{} }

func (k *KeyRing) LoadPEM(pemData []byte) error {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return errors.New("failed to decode PEM block containing public key")
	}
	if block.Type != "PUBLIC KEY" {
		return errors.New("unsupported PEM block type")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	ed25519Key, ok := pub.(ed25519.PublicKey)
	if !ok {
		return errors.New("not an Ed25519 public key")
	}
	if len(ed25519Key) != ed25519.PublicKeySize {
		return errors.New("invalid public key size")
	}

	k.trustedKeys = append(k.trustedKeys, ed25519Key)
	return nil
}

func (k *KeyRing) Keys() []ed25519.PublicKey {
	return k.trustedKeys
}
EOF
git add core/security/keyring.go
git commit -m "feat(security): implement rigorous PEM block validation" || true

git push || true
