package memory

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"testing"
	"time"
)

// buildTestIdentity creates a self-signed ECDSA P-256 certificate + TLS config
// for use in tests (does not touch the filesystem).
func buildTestIdentity(t *testing.T, nodeID string) (*tls.Config, *x509.CertPool, []byte) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "TestCA"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}
	caCert, _ := x509.ParseCertificate(caCertDER)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: nodeID},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{nodeID, "localhost"},
	}
	leafCertDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf cert: %v", err)
	}

	leafPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafCertDER})
	leafKeyDER, _ := x509.MarshalECPrivateKey(leafKey)
	leafKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})

	tlsCert, err := tls.X509KeyPair(leafPEM, leafKeyPEM)
	if err != nil {
		t.Fatalf("build tls cert: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	cfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		ClientCAs:    pool,
		RootCAs:      pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}
	return cfg, pool, leafCertDER
}

func TestSignQuery_RoundtripVerification(t *testing.T) {
	tlsCfg, caPool, _ := buildTestIdentity(t, "signer-node")

	vs := NewVectorStore()
	vs.Store("doc1", []float32{1, 0, 0}, "alpha")
	vs.Store("doc2", []float32{0, 1, 0}, "beta")

	sms := NewSignedMemoryStore(vs, caPool)

	emb := []float32{1, 0, 0}
	token, leafDER, err := SignQuery(tlsCfg, emb, 2, "signer-node")
	if err != nil {
		t.Fatalf("SignQuery: %v", err)
	}

	req := &SignedQueryRequest{Token: token, CertDER: leafDER}
	results, err := sms.AuthorisedQuery(req)
	if err != nil {
		t.Fatalf("AuthorisedQuery: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one result")
	}
	if results[0].ID != "doc1" {
		t.Errorf("expected 'doc1' as top result, got '%s'", results[0].ID)
	}
}

func TestSignedStore_ExpiredToken(t *testing.T) {
	tlsCfg, caPool, leafDER := buildTestIdentity(t, "exp-node")
	vs := NewVectorStore()
	sms := NewSignedMemoryStore(vs, caPool)

	token, _, err := SignQuery(tlsCfg, []float32{1, 0}, 1, "exp-node")
	if err != nil {
		t.Fatalf("SignQuery: %v", err)
	}

	// Back-date the token beyond the TTL.
	token.IssuedAt = time.Now().Add(-(tokenTTL + time.Second)).UnixNano()

	req := &SignedQueryRequest{Token: token, CertDER: leafDER}
	_, err = sms.AuthorisedQuery(req)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestSignedStore_TamperedSignature(t *testing.T) {
	tlsCfg, caPool, leafDER := buildTestIdentity(t, "tamper-node")
	vs := NewVectorStore()
	sms := NewSignedMemoryStore(vs, caPool)

	token, _, err := SignQuery(tlsCfg, []float32{1, 0}, 1, "tamper-node")
	if err != nil {
		t.Fatalf("SignQuery: %v", err)
	}

	// Flip a byte in the signature.
	token.Signature[0] ^= 0xFF

	req := &SignedQueryRequest{Token: token, CertDER: leafDER}
	_, err = sms.AuthorisedQuery(req)
	if err == nil {
		t.Error("expected error for tampered signature")
	}
}

func TestSignedStore_UntrustedCert(t *testing.T) {
	// Server trusts caPool1; client signs with caPool2 — different CA.
	_, caPool1, _ := buildTestIdentity(t, "server-node")
	tlsCfg2, _, leafDER2 := buildTestIdentity(t, "rogue-node")

	vs := NewVectorStore()
	sms := NewSignedMemoryStore(vs, caPool1)

	token, _, err := SignQuery(tlsCfg2, []float32{1, 0}, 1, "rogue-node")
	if err != nil {
		t.Fatalf("SignQuery: %v", err)
	}

	req := &SignedQueryRequest{Token: token, CertDER: leafDER2}
	_, err = sms.AuthorisedQuery(req)
	if err == nil {
		t.Error("expected error for untrusted CA")
	}
}

func TestTokenDigest_Deterministic(t *testing.T) {
	tok := &QueryToken{
		NodeID:    "det-node",
		IssuedAt:  1000000,
		TopK:      5,
		Embedding: []float32{0.1, 0.2, 0.3},
	}
	d1 := tokenDigest(tok)
	d2 := tokenDigest(tok)
	if d1 != d2 {
		t.Error("tokenDigest is not deterministic")
	}
}
