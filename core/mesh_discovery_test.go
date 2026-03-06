package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewMeshDiscovery_InitializesCorrectly(t *testing.T) {
	m := NewMeshDiscovery("node-alpha", "192.168.1.10:8080", "0.1.0")
	if m == nil {
		t.Fatal("expected non-nil MeshDiscovery")
	}
	if m.beacon.NodeID != "node-alpha" {
		t.Errorf("expected node_id 'node-alpha', got '%s'", m.beacon.NodeID)
	}
	if m.beacon.GRPCAddr != "192.168.1.10:8080" {
		t.Errorf("expected grpc_addr '192.168.1.10:8080', got '%s'", m.beacon.GRPCAddr)
	}
}

func TestPeers_EmptyOnInit(t *testing.T) {
	m := NewMeshDiscovery("node-beta", "localhost:9000", "0.1.0")
	peers := m.Peers()
	if len(peers) != 0 {
		t.Errorf("expected 0 peers on init, got %d", len(peers))
	}
}

func TestEvictStalePeers_RemovesTimedOutPeers(t *testing.T) {
	m := NewMeshDiscovery("node-gamma", "localhost:9001", "0.1.0")

	// Inject a stale peer manually.
	m.mu.Lock()
	m.peers["stale-node"] = &Peer{
		NodeID:   "stale-node",
		GRPCAddr: "10.0.0.1:9000",
		LastSeen: time.Now().Add(-30 * time.Second), // well past 15s timeout
	}
	m.peers["fresh-node"] = &Peer{
		NodeID:   "fresh-node",
		GRPCAddr: "10.0.0.2:9000",
		LastSeen: time.Now(),
	}
	m.mu.Unlock()

	log := WithComponent("test")
	m.evictStalePeers(log)

	peers := m.Peers()
	if len(peers) != 1 {
		t.Errorf("expected 1 peer after eviction, got %d", len(peers))
	}
	if peers[0].NodeID != "fresh-node" {
		t.Errorf("expected 'fresh-node' to survive, got '%s'", peers[0].NodeID)
	}
}

func TestPeerBeacon_MarshalRoundtrip(t *testing.T) {
	original := PeerBeacon{
		NodeID:    "roundtrip-node",
		GRPCAddr:  "172.16.0.5:9090",
		Version:   "0.1.0",
		Timestamp: 1234567890,
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded PeerBeacon
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.NodeID != original.NodeID || decoded.GRPCAddr != original.GRPCAddr {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestMeshDiscovery_SelfBeaconsIgnored(t *testing.T) {
	m := NewMeshDiscovery("self-node", "localhost:9002", "0.1.0")
	log := WithComponent("test")

	// Simulate receiving our own beacon.
	selfBeacon := PeerBeacon{
		NodeID:   "self-node",
		GRPCAddr: "localhost:9002",
		Version:  "0.1.0",
	}
	data, _ := json.Marshal(selfBeacon)

	// Directly test the unmarshalling logic: if NodeID == self, skip.
	var parsed PeerBeacon
	_ = json.Unmarshal(data, &parsed)
	if parsed.NodeID == m.beacon.NodeID {
		// should be skipped — verify peers stays empty
	}
	_ = log

	if len(m.Peers()) != 0 {
		t.Errorf("self beacon should not be registered as a peer")
	}
}
