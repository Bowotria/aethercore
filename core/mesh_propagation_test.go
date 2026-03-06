package core

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestNewMeshPropagator_Init verifies the struct initialises without error.
func TestNewMeshPropagator_Init(t *testing.T) {
	identity, err := LoadOrCreateIdentity("test-prop-node")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	p := NewMeshPropagator(identity)
	if p == nil {
		t.Fatal("expected non-nil MeshPropagator")
	}
}

// TestMeshPropagator_SetAndInvokeHandler verifies handler registration.
func TestMeshPropagator_SetAndInvokeHandler(t *testing.T) {
	identity, err := LoadOrCreateIdentity("test-handler-node")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}
	p := NewMeshPropagator(identity)

	called := make(chan struct{}, 1)
	p.SetHandler(func(task *PropagatedTask) *PropagatedResult {
		called <- struct{}{}
		return &PropagatedResult{
			TaskID: task.TaskID,
			Output: "pong",
		}
	})

	// Call the handler directly (simulating what handleConn does).
	p.mu.Lock()
	h := p.handler
	p.mu.Unlock()

	if h == nil {
		t.Fatal("handler should be set")
	}

	result := h(&PropagatedTask{TaskID: "t1", Prompt: "hello"})
	select {
	case <-called:
	default:
		t.Error("handler was not invoked")
	}
	if result.Output != "pong" {
		t.Errorf("expected 'pong', got '%s'", result.Output)
	}
}

// TestPropagatedTask_SerializedCorrectly verifies struct field names.
func TestPropagatedTask_SerializedCorrectly(t *testing.T) {
	task := &PropagatedTask{
		TaskID:    "task-42",
		SourceID:  "node-A",
		Prompt:    "Summarise the document.",
		Tools:     []string{"read_file", "summarise"},
		Model:     "llama3",
		Timestamp: 1234567890,
	}
	if task.TaskID != "task-42" {
		t.Error("TaskID mismatch")
	}
	if task.SourceID != "node-A" {
		t.Errorf("expected SourceID 'node-A', got '%s'", task.SourceID)
	}
	if task.Prompt != "Summarise the document." {
		t.Errorf("expected Prompt set, got '%s'", task.Prompt)
	}
	if task.Model != "llama3" {
		t.Errorf("expected Model 'llama3', got '%s'", task.Model)
	}
	if task.Timestamp != 1234567890 {
		t.Errorf("expected Timestamp 1234567890, got %d", task.Timestamp)
	}
	if len(task.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(task.Tools))
	}
}

// TestMeshPropagator_ListenAndPropagate runs a full loopback mTLS roundtrip.
// Server and client share the same NodeIdentity (single-node test CA), so the
// leaf cert's SAN covers "localhost" and "127.0.0.1".
func TestMeshPropagator_ListenAndPropagate(t *testing.T) {
	identity, err := LoadOrCreateIdentity("proptest-loopback")
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}

	// Pick a random free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	server := NewMeshPropagator(identity)
	server.SetHandler(func(task *PropagatedTask) *PropagatedResult {
		return &PropagatedResult{
			TaskID: task.TaskID,
			NodeID: "proptest-loopback",
			Output: "echo:" + task.Prompt,
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Listen(ctx, addr)
	}()

	// Give the server a moment to bind.
	time.Sleep(50 * time.Millisecond)

	client := NewMeshPropagator(identity)
	peer := Peer{
		NodeID:   "proptest-loopback",
		GRPCAddr: addr,
	}
	task := &PropagatedTask{
		TaskID:   "roundtrip-1",
		SourceID: "proptest-loopback",
		Prompt:   "hello mesh",
	}

	result, err := client.PropagateTask(ctx, peer, task)
	cancel() // trigger server shutdown

	if err != nil {
		t.Fatalf("PropagateTask: %v", err)
	}
	if result.TaskID != "roundtrip-1" {
		t.Errorf("expected task_id 'roundtrip-1', got '%s'", result.TaskID)
	}
	if result.Output != "echo:hello mesh" {
		t.Errorf("expected 'echo:hello mesh', got '%s'", result.Output)
	}
}
