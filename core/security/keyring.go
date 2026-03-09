package security

import "crypto/ed25519"

type KeyRing struct {
	trustedKeys []ed25519.PublicKey
}
func NewKeyRing() *KeyRing { return &KeyRing{} }
