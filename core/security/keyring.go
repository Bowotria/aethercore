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
