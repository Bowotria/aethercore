package security

type ToolVerifier interface {
	Verify(manifestJSON []byte, signatureHex string) (bool, error)
}
