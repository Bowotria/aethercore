package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalAppender_Init(t *testing.T) {
	appender := NewLocalAppender("test.log")
	if appender == nil {
		t.Fatalf("expected non-nil appender")
	}
}

func TestLocalAppender_FileCreationStrictPerms(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "audit.log")
	appender := NewLocalAppender(tmpFile)
	err := appender.Open()
	if err != nil {
		t.Fatalf("failed to open appender: %v", err)
	}

	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("file not created")
	}

	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected strict 0600 permissions, got %v", info.Mode().Perm())
	}
}
