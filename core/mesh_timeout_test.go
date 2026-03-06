package core

import (
	"context"
	"testing"
	"time"
)

func TestEphemeralLog_RecordAndQuery(t *testing.T) {
	log := NewEphemeralLog()

	log.Record(EphemeralEvent{TaskID: "t1", NodeID: "node-A", Stage: StageCreated})
	log.Record(EphemeralEvent{TaskID: "t1", NodeID: "node-B", Stage: StagePropagated})
	log.Record(EphemeralEvent{TaskID: "t1", NodeID: "node-B", Stage: StageCompleted})

	events := log.Events("t1")
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Stage != StageCreated {
		t.Errorf("expected StageCreated at [0], got %s", events[0].Stage)
	}
	if events[2].Stage != StageCompleted {
		t.Errorf("expected StageCompleted at [2], got %s", events[2].Stage)
	}
}

func TestEphemeralLog_EmptyQuery(t *testing.T) {
	log := NewEphemeralLog()
	events := log.Events("nonexistent")
	if events != nil {
		t.Errorf("expected nil for unknown task, got %v", events)
	}
}

func TestEphemeralLog_Purge(t *testing.T) {
	log := NewEphemeralLog()
	log.Record(EphemeralEvent{TaskID: "purge-me", NodeID: "n1", Stage: StageCreated})

	log.Purge("purge-me")
	if log.Events("purge-me") != nil {
		t.Error("events should be nil after purge")
	}
	if log.TaskCount() != 0 {
		t.Errorf("expected 0 tasks after purge, got %d", log.TaskCount())
	}
}

func TestEphemeralLog_PurgeExpired(t *testing.T) {
	log := NewEphemeralLog()

	// Old event (record then manually backdate).
	log.Record(EphemeralEvent{
		TaskID:    "old-task",
		NodeID:    "n1",
		Stage:     StageTimeout,
		Timestamp: time.Now().Add(-2 * time.Minute),
	})
	// Recent event.
	log.Record(EphemeralEvent{TaskID: "new-task", NodeID: "n1", Stage: StageCreated})

	removed := log.PurgeExpired(time.Minute)
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
	if log.Events("old-task") != nil {
		t.Error("old task should have been purged")
	}
	if log.Events("new-task") == nil {
		t.Error("new task should still exist")
	}
}

func TestWithTaskDeadline_NoDeadline(t *testing.T) {
	task := &PropagatedTask{TaskID: "no-dl", DeadlineUnixNs: 0}
	ctx, cancel := WithTaskDeadline(context.Background(), task)
	defer cancel()
	if _, ok := ctx.Deadline(); ok {
		t.Error("expected no deadline when DeadlineUnixNs is 0")
	}
}

func TestWithTaskDeadline_WithDeadline(t *testing.T) {
	task := &PropagatedTask{TaskID: "with-dl"}
	SetTaskDeadline(task, 5*time.Second)

	ctx, cancel := WithTaskDeadline(context.Background(), task)
	defer cancel()

	dl, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected a deadline")
	}
	if time.Until(dl) <= 0 || time.Until(dl) > 6*time.Second {
		t.Errorf("deadline out of expected range: %v remaining", time.Until(dl))
	}
}

func TestSetAndQueryDeadline(t *testing.T) {
	task := &PropagatedTask{TaskID: "dl-query"}
	SetTaskDeadline(task, 10*time.Second)

	rem, ok := TaskDeadlineRemaining(task)
	if !ok {
		t.Fatal("expected deadline remaining")
	}
	if rem <= 0 || rem > 10*time.Second {
		t.Errorf("remaining out of range: %v", rem)
	}
}

func TestSetTaskDeadline_ClearsOnZero(t *testing.T) {
	task := &PropagatedTask{TaskID: "clear-dl"}
	SetTaskDeadline(task, 5*time.Second)
	SetTaskDeadline(task, 0) // clear

	if task.DeadlineUnixNs != 0 {
		t.Error("deadline should be cleared")
	}
	_, ok := TaskDeadlineRemaining(task)
	if ok {
		t.Error("expected no deadline remaining after clearing")
	}
}

func TestWithTaskDeadline_PastDeadline(t *testing.T) {
	task := &PropagatedTask{
		TaskID:         "past-dl",
		DeadlineUnixNs: time.Now().Add(-time.Second).UnixNano(),
	}
	ctx, cancel := WithTaskDeadline(context.Background(), task)
	defer cancel()

	// The context should already be done (or done very soon).
	select {
	case <-ctx.Done():
		// correct — deadline already exceeded
	case <-time.After(100 * time.Millisecond):
		t.Error("expected context to be cancelled for past deadline")
	}
}
