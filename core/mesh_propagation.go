package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// PropagatedTask carries an LLM context from a source node to an idle edge node.
type PropagatedTask struct {
	TaskID         string   `json:"task_id"`
	SourceID       string   `json:"source_id"`
	Prompt         string   `json:"prompt"`
	Tools          []string `json:"tools"`
	Model          string   `json:"model"`
	Timestamp      int64    `json:"timestamp"`
	DeadlineUnixNs int64    `json:"deadline_unix_ns,omitempty"` // 0 means no deadline
}

// PropagatedResult is the response returned by the edge node after executing
// a propagated task.
type PropagatedResult struct {
	TaskID     string `json:"task_id"`
	NodeID     string `json:"node_id"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

// TaskHandler processes an incoming propagated task and returns a result.
type TaskHandler func(task *PropagatedTask) *PropagatedResult

// MeshPropagator enables mTLS-secured task passing between AetherCore nodes.
// It can act as both a server (accepting inbound tasks) and a client
// (dispatching tasks to discovered peers).
type MeshPropagator struct {
	identity *NodeIdentity
	handler  TaskHandler
	listener net.Listener
	mu       sync.Mutex
	quit     chan struct{}
}

// NewMeshPropagator creates a propagator anchored to the given node identity.
// The identity provides the mTLS TLSConfig used for all connections.
func NewMeshPropagator(identity *NodeIdentity) *MeshPropagator {
	return &MeshPropagator{
		identity: identity,
		quit:     make(chan struct{}),
	}
}

// SetHandler registers the callback invoked when this node receives a task
// from a remote peer. Must be set before calling Listen.
func (p *MeshPropagator) SetHandler(fn TaskHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handler = fn
}

// Listen starts the mTLS server on addr (e.g., ":9100") and accepts incoming
// task propagation requests until ctx is cancelled or Stop is called.
func (p *MeshPropagator) Listen(ctx context.Context, addr string) error {
	tlsCfg := p.identity.TLSConfig.Clone()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("mesh propagator: bind %s: %w", addr, err)
	}
	// Wrap with mTLS.
	tlsLn := newTLSListener(ln, tlsCfg)

	p.mu.Lock()
	p.listener = tlsLn
	p.mu.Unlock()

	log := WithComponent("mesh.propagator")
	log.Info("mesh_propagator_listening", "addr", addr)

	go func() {
		select {
		case <-ctx.Done():
			_ = tlsLn.Close()
		case <-p.quit:
			_ = tlsLn.Close()
		}
	}()

	for {
		conn, err := tlsLn.Accept()
		if err != nil {
			select {
			case <-p.quit:
				return nil
			case <-ctx.Done():
				return nil
			default:
				log.Error("mesh_propagator_accept_error", "error", err)
				continue
			}
		}
		go p.handleConn(conn, log)
	}
}

// Stop shuts down the server listener gracefully.
func (p *MeshPropagator) Stop() {
	close(p.quit)
}

// PropagateTask sends a task to a remote peer over a fresh mTLS connection.
// The peer's GRPCAddr field is used as the dial target (host:port).
// Returns the result produced by the remote node.
func (p *MeshPropagator) PropagateTask(ctx context.Context, peer Peer, task *PropagatedTask) (*PropagatedResult, error) {
	log := WithComponent("mesh.propagator")

	tlsCfg := p.identity.TLSConfig.Clone()
	// Use a dialer that respects context cancellation.
	dialer := &net.Dialer{}
	rawConn, err := dialer.DialContext(ctx, "tcp", peer.GRPCAddr)
	if err != nil {
		return nil, fmt.Errorf("mesh propagator: dial %s: %w", peer.GRPCAddr, err)
	}

	tlsConn := newTLSClientConn(rawConn, tlsCfg, hostFromAddr(peer.GRPCAddr))
	defer func() { _ = tlsConn.Close() }()

	// Set deadline from context.
	if deadline, ok := ctx.Deadline(); ok {
		if err := tlsConn.SetDeadline(deadline); err != nil {
			log.Error("mesh_propagator_set_deadline_error", "error", err)
		}
	}

	encoder := json.NewEncoder(tlsConn)
	decoder := json.NewDecoder(tlsConn)

	if err := encoder.Encode(task); err != nil {
		return nil, fmt.Errorf("mesh propagator: send task: %w", err)
	}

	var result PropagatedResult
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("mesh propagator: read result: %w", err)
	}

	log.Info("mesh_task_propagated",
		"task_id", task.TaskID,
		"peer", peer.NodeID,
		"duration_ms", result.DurationMS,
	)
	return &result, nil
}

// handleConn reads one PropagatedTask from the connection, dispatches to the
// registered handler, and writes back the PropagatedResult.
func (p *MeshPropagator) handleConn(conn net.Conn, log *slog.Logger) {
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var task PropagatedTask
	if err := decoder.Decode(&task); err != nil {
		log.Error("mesh_propagator_decode_error", "error", err)
		return
	}

	p.mu.Lock()
	handler := p.handler
	p.mu.Unlock()

	var result *PropagatedResult
	if handler == nil {
		result = &PropagatedResult{
			TaskID: task.TaskID,
			Error:  "no handler registered",
		}
	} else {
		start := time.Now()
		result = handler(&task)
		if result == nil {
			result = &PropagatedResult{TaskID: task.TaskID}
		}
		result.DurationMS = time.Since(start).Milliseconds()
	}

	if err := encoder.Encode(result); err != nil {
		log.Error("mesh_propagator_encode_error", "error", err)
	}
}

// hostFromAddr extracts the hostname from "host:port" for TLS SNI.
func hostFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
