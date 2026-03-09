package audit

import (
	"os"
	"path/filepath"
	"sync"
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
	defer appender.Close()

	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("file not created")
	}

	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected strict 0600 permissions, got %v", info.Mode().Perm())
	}
}

func TestLocalAppender_ConcurrentWriteSafety(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "concurrent_audit.log")
	appender := NewLocalAppender(tmpFile)
	_ = appender.Open()
	defer appender.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := appender.AppendBlock(Block{Index: uint64(idx)})
			if err != nil {
				t.Errorf("expected successful write")
			}
		}(i)
	}
	wg.Wait()
}
