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

// Append cryptographically links a new event to the end of the chain.
func (c *ChainManager) Append(event AuditEvent) Block {
	c.mu.Lock()
	defer c.mu.Unlock()

	prevBlock := c.blocks[len(c.blocks)-1]

	newBlock := Block{
		Index:        prevBlock.Index + 1,
		Timestamp:    time.Now(),
		Event:        event,
		PreviousHash: prevBlock.Hash,
	}
	newBlock.Hash = c.calculateHash(newBlock)
	c.blocks = append(c.blocks, newBlock)
	return newBlock
}
