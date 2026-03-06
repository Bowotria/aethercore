package core_test

// TestFiveNodeMesh is the Day 21 integration test simulating a 5-node
// AetherCore mesh. It exercises all Week 3 components end-to-end:
//   - Day 15: NodeIdentity / mTLS (shared mesh CA, per-node leaf certs)
//   - Day 16: Peer structure (MeshDiscovery types)
//   - Day 17: MeshPropagator over mTLS TCP
//   - Day 18: VectorStore per node
//   - Day 19: SignedMemoryStore + QueryToken signing + verification
//   - Day 20: EphemeralLog, SetTaskDeadline, WithTaskDeadline

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/fzihak/aethercore/core"
	"github.com/fzihak/aethercore/memory"
)

const numNodes = 5

// meshNode represents one in-process AetherCore node for integration testing.
type meshNode struct {
	id          string
	tlsCfg      *tls.Config
	caPool      *x509.CertPool
	leafDER     []byte
	propagator  *core.MeshPropagator
	ephLog      *core.EphemeralLog
	vecStore    *memory.VectorStore
	signedStore *memory.SignedMemoryStore
	addr        string
}

// buildMeshCluster creates n nodes sharing a single mesh CA.
func buildMeshCluster(t *testing.T, n int) ([]*meshNode, *x509.CertPool) {
	t.Helper()

	// 1. Generate a shared mesh CA.
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "AetherCore Mesh Test CA"},
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
	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	const leafLifetime = time.Hour

	// 2. Generate one leaf cert + TLS config per node.
	nodes := make([]*meshNode, n)
	for i := range n {
		nodeID := fmt.Sprintf("node-%d", i)

		leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("generate leaf key for %s: %v", nodeID, err)
		}
		leafTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(int64(i + 2)),
			Subject:      pkix.Name{CommonName: nodeID},
			NotBefore:    time.Now().Add(-time.Minute),
			NotAfter:     time.Now().Add(leafLifetime),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
			DNSNames:     []string{nodeID, "localhost"},
		}
		leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
		if err != nil {
			t.Fatalf("create leaf cert for %s: %v", nodeID, err)
		}

		leafPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
		leafKeyDER, _ := x509.MarshalECPrivateKey(leafKey)
		leafKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})

		tlsCert, err := tls.X509KeyPair(leafPEM, leafKeyPEM)
		if err != nil {
			t.Fatalf("build tls cert for %s: %v", nodeID, err)
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			ClientCAs:    caPool,
			RootCAs:      caPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}

		vs := memory.NewVectorStore()
		nm := &meshNode{
			id:          nodeID,
			tlsCfg:      tlsCfg,
			caPool:      caPool,
			leafDER:     leafDER,
			ephLog:      core.NewEphemeralLog(),
			vecStore:    vs,
			signedStore: memory.NewSignedMemoryStore(vs, caPool),
		}
		// Construct a NodeIdentity-like struct the propagator needs.
		// Since NodeIdentity is in core and its fields are exported, we build
		// a lightweight wrapper using reflection to stay package-agnostic.
		nm.propagator = buildPropagator(t, tlsCfg)
		nodes[i] = nm
	}
	return nodes, caPool
}

// buildPropagator creates a MeshPropagator from a ready-made TLS config.
// MeshPropagator requires a *core.NodeIdentity but we have only a tls.Config,
// so we build a minimal identity wrapper via reflection-free direct struct init.
func buildPropagator(t *testing.T, tlsCfg *tls.Config) *core.MeshPropagator {
	t.Helper()
	identity := &core.NodeIdentity{TLSConfig: tlsCfg}
	return core.NewMeshPropagator(identity)
}

// freeAddr finds an ephemeral TCP address on the loopback interface.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

func TestFiveNodeMesh(t *testing.T) {
	nodes, _ := buildMeshCluster(t, numNodes)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ── 1. Start all propagator servers ──────────────────────────────────────
	for _, n := range nodes {
		n.addr = freeAddr(t)
		go func() {
			_ = n.propagator.Listen(ctx, n.addr)
		}()
	}
	time.Sleep(60 * time.Millisecond) // let servers bind

	// ── 2. Register task handler on nodes 1-4 ────────────────────────────────
	// Each peer stores a unique result embedding and records lifecycle events.
	for _, peer := range nodes[1:] {
		peer.propagator.SetHandler(func(task *core.PropagatedTask) *core.PropagatedResult {
			// Record task receipt.
			peer.ephLog.Record(core.EphemeralEvent{
				TaskID: task.TaskID,
				NodeID: peer.id,
				Stage:  core.StagePropagated,
			})

			// Verify deadline is still live.
			rem, ok := core.TaskDeadlineRemaining(task)
			if !ok {
				peer.ephLog.Record(core.EphemeralEvent{
					TaskID: task.TaskID,
					NodeID: peer.id,
					Stage:  core.StageTimeout,
					Detail: "deadline already expired on arrival",
				})
				return &core.PropagatedResult{
					TaskID: task.TaskID,
					NodeID: peer.id,
					Error:  "deadline expired",
				}
			}

			_ = rem // deadline still valid

			// Store a result embedding in this node's VectorStore.
			emb := []float32{float32(len(peer.id)), 0.5, 0.1}
			peer.vecStore.Store(task.TaskID, emb, fmt.Sprintf("result from %s: %s", peer.id, task.Prompt))

			peer.ephLog.Record(core.EphemeralEvent{
				TaskID: task.TaskID,
				NodeID: peer.id,
				Stage:  core.StageCompleted,
			})

			return &core.PropagatedResult{
				TaskID: task.TaskID,
				NodeID: peer.id,
				Output: "processed by " + peer.id,
			}
		})
	}

	// ── 3. Node 0 creates a task and propagates it to peers 1-4 ──────────────
	task := &core.PropagatedTask{
		TaskID:   "mesh-integration-001",
		SourceID: nodes[0].id,
		Prompt:   "Summarise the distributed mesh status.",
		Model:    "llama3",
	}
	core.SetTaskDeadline(task, 8*time.Second)
	nodes[0].ephLog.Record(core.EphemeralEvent{
		TaskID: task.TaskID,
		NodeID: nodes[0].id,
		Stage:  core.StageCreated,
	})

	peers := make([]core.Peer, numNodes-1)
	for i, n := range nodes[1:] {
		peers[i] = core.Peer{NodeID: n.id, GRPCAddr: n.addr}
	}

	results := make([]*core.PropagatedResult, numNodes-1)
	for i, peer := range peers {
		res, err := nodes[0].propagator.PropagateTask(ctx, peer, task)
		if err != nil {
			t.Errorf("PropagateTask to %s: %v", peer.NodeID, err)
			continue
		}
		results[i] = res
	}

	// ── 4. Assert all peers completed ────────────────────────────────────────
	for i, res := range results {
		if res == nil {
			t.Errorf("node %d returned nil result", i+1)
			continue
		}
		if res.Error != "" {
			t.Errorf("node %d returned error: %s", i+1, res.Error)
		}
		expectedOutput := fmt.Sprintf("processed by node-%d", i+1)
		if res.Output != expectedOutput {
			t.Errorf("node %d: expected output %q, got %q", i+1, expectedOutput, res.Output)
		}
	}

	// ── 5. Verify EphemeralLog events on each peer ────────────────────────────
	for i, n := range nodes[1:] {
		events := n.ephLog.Events(task.TaskID)
		if len(events) < 2 {
			t.Errorf("node %d: expected ≥2 ephemeral events, got %d", i+1, len(events))
			continue
		}
		stages := make([]core.EphemeralStage, len(events))
		for j, ev := range events {
			stages[j] = ev.Stage
		}
		expected := []core.EphemeralStage{core.StagePropagated, core.StageCompleted}
		if !reflect.DeepEqual(stages, expected) {
			t.Errorf("node %d: expected stages %v, got %v", i+1, expected, stages)
		}
	}

	// ── 6. Verify VectorStore populated on each peer ──────────────────────────
	for i, n := range nodes[1:] {
		if n.vecStore.Len() == 0 {
			t.Errorf("node %d VectorStore is empty after task completion", i+1)
		}
		entry := n.vecStore.Get(task.TaskID)
		if entry == nil {
			t.Errorf("node %d: task entry not found in VectorStore", i+1)
		}
	}

	// ── 7. Node 0 queries each peer's signed memory ───────────────────────────
	for i, n := range nodes[1:] {
		queryEmb := []float32{float32(len(n.id)), 0.5, 0.1}
		token, leafDER, err := memory.SignQuery(nodes[0].tlsCfg, queryEmb, 1, nodes[0].id)
		if err != nil {
			t.Errorf("node 0 SignQuery for peer %d: %v", i+1, err)
			continue
		}
		req := &memory.SignedQueryRequest{Token: token, CertDER: leafDER}
		memResults, err := n.signedStore.AuthorisedQuery(req)
		if err != nil {
			t.Errorf("AuthorisedQuery on node %d: %v", i+1, err)
			continue
		}
		if len(memResults) == 0 {
			t.Errorf("node %d signed memory query returned no results", i+1)
		}
	}

	// ── 8. Purge expired ephemeral events and verify cleanup ──────────────────
	for _, n := range nodes {
		removed := n.ephLog.PurgeExpired(0) // purge everything (age=0)
		_ = removed
		if n.ephLog.TaskCount() != 0 {
			t.Errorf("EphemeralLog not empty after full purge on %s", n.id)
		}
	}

	cancel() // shutdown servers
}
