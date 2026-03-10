package memory

import (
	"context"
	"testing"

	"github.com/fzihak/aethercore/core/llm"
)

func TestMemoryEngine_Record(t *testing.T) {
	storage := NewZestDBStorage()
	engine := NewMemoryEngine(storage, 5)

	msg := llm.Message{Role: "user", Content: "hello world"}
	err := engine.Record(context.Background(), msg)
	if err != nil {
		t.Fatalf("failed to record memory: %v", err)
	}

	if len(engine.shortTermMem) != 1 {
		t.Errorf("expected 1 msg in short-term, got %d", len(engine.shortTermMem))
	}
}
