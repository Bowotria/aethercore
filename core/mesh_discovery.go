package core

// Inviolable Rule: Layer 0 strictly uses Go stdlib ONLY.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

const (
	// meshDiscoveryPort is the UDP port used for local broadcast peer discovery.
	// All AetherCore nodes on the same LAN segment listen on this port.
	meshDiscoveryPort = 7946

	// meshBroadcastInterval is how often a node announces itself to the LAN.
	meshBroadcastInterval = 5 * time.Second

	// meshPeerTimeout is how long before a peer is considered stale and removed.
	meshPeerTimeout = 15 * time.Second

	// maxBeaconSize is the maximum UDP payload we accept (prevents amplification).
	maxBeaconSize = 512
)

// ErrMeshAlreadyRunning is returned when Start() is called on a running Mesh.
var ErrMeshAlreadyRunning = errors.New("mesh: peer discovery already running")

// PeerBeacon is the JSON payload broadcast over UDP by each node.
// Kept minimal to fit within a single UDP datagram.
type PeerBeacon struct {
	NodeID    string `json:"node_id"`
	GRPCAddr  string `json:"grpc_addr"` // host:port for task propagation
	Version   string `json:"version"`
	Timestamp int64  `json:"ts"` // Unix seconds
}

// Peer represents a discovered remote AetherCore node.
type Peer struct {
	NodeID   string
	GRPCAddr string
	Version  string
	LastSeen time.Time
}

// MeshDiscovery runs Layer 3 UDP broadcast peer discovery on the local network.
// It both announces this node to peers AND listens for peer announcements.
type MeshDiscovery struct {
	beacon   PeerBeacon
	peers    map[string]*Peer // keyed by NodeID
	mu       sync.RWMutex
	quit     chan struct{}
	stopOnce sync.Once
	running  bool
}

// NewMeshDiscovery creates a MeshDiscovery instance.
// beacon is the description this node broadcasts to the LAN.
func NewMeshDiscovery(nodeID, grpcAddr, version string) *MeshDiscovery {
	return &MeshDiscovery{
		beacon: PeerBeacon{
			NodeID:   nodeID,
			GRPCAddr: grpcAddr,
			Version:  version,
		},
		peers: make(map[string]*Peer),
		quit:  make(chan struct{}),
	}
}

// Start launches the background announce + listen goroutines.
// Cancel ctx or call Stop() to shut down cleanly.
func (m *MeshDiscovery) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return ErrMeshAlreadyRunning
	}
	m.running = true
	m.mu.Unlock()

	// 1. Open the UDP listener on all interfaces for incoming beacons.
	listenAddr := fmt.Sprintf("0.0.0.0:%d", meshDiscoveryPort)
	conn, err := net.ListenPacket("udp4", listenAddr)
	if err != nil {
		return fmt.Errorf("mesh: bind udp listener on %s: %w", listenAddr, err)
	}

	log := WithComponent("mesh_discovery")
	log.Info("mesh_discovery_started",
		slog.String("node_id", m.beacon.NodeID),
		slog.String("grpc_addr", m.beacon.GRPCAddr),
		slog.String("listen", listenAddr),
	)

	// 2. Goroutine: listen for incoming peer beacons.
	go m.listenLoop(conn, log)

	// 3. Goroutine: broadcast our own beacon and evict stale peers.
	go m.announceLoop(ctx, log)

	return nil
}

// Stop gracefully shuts down both goroutines.
func (m *MeshDiscovery) Stop() {
	m.stopOnce.Do(func() {
		close(m.quit)
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
		WithComponent("mesh_discovery").Info("mesh_discovery_stopped")
	})
}

// Peers returns a snapshot of all currently known live peers (excluding self).
func (m *MeshDiscovery) Peers() []Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Peer, 0, len(m.peers))
	for _, p := range m.peers {
		out = append(out, *p)
	}
	return out
}

// listenLoop reads UDP datagrams and registers discovered peers.
func (m *MeshDiscovery) listenLoop(conn net.PacketConn, log *slog.Logger) {
	defer conn.Close()
	buf := make([]byte, maxBeaconSize)

	for {
		select {
		case <-m.quit:
			return
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, addr, readErr := conn.ReadFrom(buf)
		if readErr != nil {
			if m.handleReadError(readErr, log) {
				return
			}
			continue
		}

		var beacon PeerBeacon
		if unmarshalErr := json.Unmarshal(buf[:n], &beacon); unmarshalErr != nil {
			log.Debug("mesh_beacon_parse_error",
				slog.String("from", addr.String()),
				slog.String("error", unmarshalErr.Error()),
			)
			continue
		}

		m.upsertPeer(beacon, addr.String(), log)
	}
}

// handleReadError returns true if the listen loop should exit (quit signal received).
// Timeout errors are expected during normal shutdown polling and return false.
func (m *MeshDiscovery) handleReadError(err error, log *slog.Logger) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}
	select {
	case <-m.quit:
		return true
	default:
		log.Warn("mesh_udp_read_error", slog.String("error", err.Error()))
		return false
	}
}

// upsertPeer registers or refreshes a discovered peer, ignoring self-beacons.
func (m *MeshDiscovery) upsertPeer(beacon PeerBeacon, from string, log *slog.Logger) {
	if beacon.NodeID == m.beacon.NodeID {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing, seen := m.peers[beacon.NodeID]
	if !seen {
		log.Info("mesh_peer_discovered",
			slog.String("peer_id", beacon.NodeID),
			slog.String("grpc_addr", beacon.GRPCAddr),
			slog.String("from", from),
		)
	}
	if !seen || existing.GRPCAddr != beacon.GRPCAddr {
		m.peers[beacon.NodeID] = &Peer{
			NodeID:   beacon.NodeID,
			GRPCAddr: beacon.GRPCAddr,
			Version:  beacon.Version,
			LastSeen: time.Now(),
		}
	} else {
		existing.LastSeen = time.Now()
	}
}

// announceLoop periodically broadcasts this node's beacon and evicts stale peers.
func (m *MeshDiscovery) announceLoop(ctx context.Context, log *slog.Logger) {
	ticker := time.NewTicker(meshBroadcastInterval)
	defer ticker.Stop()

	broadcastAddr := fmt.Sprintf("255.255.255.255:%d", meshDiscoveryPort)

	for {
		select {
		case <-m.quit:
			return
		case <-ctx.Done():
			m.Stop()
			return
		case <-ticker.C:
			m.broadcast(broadcastAddr, log)
			m.evictStalePeers(log)
		}
	}
}

// broadcast sends one UDP beacon to the LAN broadcast address.
func (m *MeshDiscovery) broadcast(broadcastAddr string, log *slog.Logger) {
	m.beacon.Timestamp = time.Now().Unix()
	payload, err := json.Marshal(m.beacon)
	if err != nil {
		log.Error("mesh_beacon_marshal_error", slog.String("error", err.Error()))
		return
	}

	conn, err := net.Dial("udp4", broadcastAddr)
	if err != nil {
		log.Warn("mesh_broadcast_dial_error", slog.String("error", err.Error()))
		return
	}
	defer conn.Close()

	if _, err := conn.Write(payload); err != nil {
		log.Warn("mesh_broadcast_write_error", slog.String("error", err.Error()))
		return
	}
	log.Debug("mesh_beacon_broadcast", slog.Int("peers_known", len(m.peers)))
}

// evictStalePeers removes peers that haven't been heard from recently.
func (m *MeshDiscovery) evictStalePeers(log *slog.Logger) {
	cutoff := time.Now().Add(-meshPeerTimeout)
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, p := range m.peers {
		if p.LastSeen.Before(cutoff) {
			log.Info("mesh_peer_evicted",
				slog.String("peer_id", id),
				slog.Duration("silent_for", time.Since(p.LastSeen)),
			)
			delete(m.peers, id)
		}
	}
}
