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

func TestMemoryEngine_Recall(t *testing.T) {
	storage := NewZestDBStorage()
	engine := NewMemoryEngine(storage, 5)

	ctx := context.Background()
	_ = engine.Record(ctx, llm.Message{Role: "user", Content: "AetherCore is a security sandbox."})
	_ = engine.Record(ctx, llm.Message{Role: "assistant", Content: "Understood."})

	messages, err := engine.Recall(ctx, "security")
	if err != nil {
		t.Fatalf("failed to recall: %v", err)
	}

	// Should have 2 short-term + some long-term (if matched)
	if len(messages) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(messages))
	}

	found := false
	for _, m := range messages {
		if m.Role == "system" && (len(m.Content) > 15 && m.Content[:15] == "[Memory Recall]") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected long-term memory recall but not found")
	}
}
