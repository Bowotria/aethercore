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
