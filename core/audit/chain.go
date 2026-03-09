package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type ChainManager struct {
	mu     sync.RWMutex
	blocks []Block
}

func (c *ChainManager) calculateHash(b Block) string {
	eventBytes, _ := json.Marshal(b.Event)
	record := fmt.Sprintf("%d:%s:%s:%s", b.Index, b.Timestamp.UTC().Format(time.RFC3339Nano), b.PreviousHash, string(eventBytes))
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

func NewChainManager() *ChainManager {
	c := &ChainManager{}
	genesis := Block{
		Index:        0,
		Timestamp:    time.Now(),
		Event:        AuditEvent{Type: "SYSTEM_INIT"},
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}
	genesis.Hash = c.calculateHash(genesis)
	c.blocks = []Block{genesis}
	return c
}
