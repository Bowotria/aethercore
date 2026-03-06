package core

import (
	"crypto/tls"
	"crypto/x509"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOrCreateIdentity_GeneratesOnFirstBoot(t *testing.T) {
	// Use a temp dir so we don't pollute the real cert store.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("APPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	identity, err := LoadOrCreateIdentity("test-node-001")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity failed: %v", err)
	}
	if identity == nil {
		t.Fatal("expected non-nil identity")
	}
	if len(identity.CACertPEM) == 0 {
		t.Fatal("expected non-empty CA cert PEM")
	}
	if identity.TLSConfig == nil {
		t.Fatal("expected non-nil TLS config")
	}
	if identity.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Fatal("expected TLS 1.3 minimum")
	}
	if identity.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatal("expected RequireAndVerifyClientCert")
	}
}

func TestLoadOrCreateIdentity_IdempotentOnReload(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("APPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// First call — generate.
	id1, err := LoadOrCreateIdentity("node-reload-test")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call — must load from disk, not regenerate.
	id2, err := LoadOrCreateIdentity("node-reload-test")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	// CA cert bytes must be identical (same CA, not regenerated).
	if string(id1.CACertPEM) != string(id2.CACertPEM) {
		t.Fatal("CA cert changed between calls — identity was unexpectedly regenerated")
	}
}

func TestLoadOrCreateIdentity_CertFilesHaveCorrectPermissions(t *testing.T) {
	if os.Getenv("CI") != "" && strings.Contains(os.Getenv("RUNNER_OS"), "Windows") {
		t.Skip("file permission check not applicable on Windows CI")
	}
	if _, err := os.Stat(`C:\`); err == nil {
		t.Skip("file permission check not applicable on Windows")
	}

	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("APPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	_, err := LoadOrCreateIdentity("perm-test-node")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity failed: %v", err)
	}

	// Find the cert dir under tmp and verify each file has mode 0600.
	err = filepath.WalkDir(tmp, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Errorf("file %s has mode %o, want 0600", path, mode)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}
}

func TestLoadOrCreateIdentity_LeafCertSignedByCA(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("APPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	identity, err := LoadOrCreateIdentity("chain-verify-node")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity failed: %v", err)
	}

	// Extract the leaf cert from the TLS config.
	if len(identity.TLSConfig.Certificates) == 0 {
		t.Fatal("no certificates in TLS config")
	}
	leafDER := identity.TLSConfig.Certificates[0].Certificate[0]
	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("parse leaf cert: %v", err)
	}

	// Verify leaf is signed by CA.
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(identity.CACertPEM)

	opts := x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if _, err := leafCert.Verify(opts); err != nil {
		t.Fatalf("leaf cert not signed by CA: %v", err)
	}
}
