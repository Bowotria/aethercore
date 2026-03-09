package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

type ChainManager struct {
	mu     sync.RWMutex
	blocks []Block
}

func (c *ChainManager) calculateHash(b *Block) string {
	eventBytes, err := json.Marshal(b.Event)
	if err != nil {
		// This should never happen with our Event schema
		return "invalid-json-hash"
	}
	record := fmt.Sprintf("%d:%s:%s:%s", b.Index, b.Timestamp.UTC().Format(time.RFC3339Nano), b.PreviousHash, string(eventBytes))
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

func NewChainManager() *ChainManager {
	c := &ChainManager{}
	genesis := &Block{
		Index:        0,
		Timestamp:    time.Now(),
		Event:        Event{Type: "SYSTEM_INIT", Actor: "kernel"},
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}
	genesis.Hash = c.calculateHash(genesis)
	c.blocks = []Block{*genesis}
	return c
}

// Append cryptographically links a new event to the end of the chain.
func (c *ChainManager) Append(event *Event) *Block {
	c.mu.Lock()
	defer c.mu.Unlock()

	prevBlock := c.blocks[len(c.blocks)-1]

	newBlock := &Block{
		Index:        prevBlock.Index + 1,
		Timestamp:    time.Now(),
		Event:        *event,
		PreviousHash: prevBlock.Hash,
	}
	newBlock.Hash = c.calculateHash(newBlock)
	c.blocks = append(c.blocks, *newBlock)
	return newBlock
}

func (c *ChainManager) LogEvent(ctx context.Context, event *Event) error {
	c.Append(event)
	return nil
}

// VerifyChain performs a full O(N) sweep to ensure absolutely zero historical tampering of the chain data.
func (c *ChainManager) VerifyChain() (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := 1; i < len(c.blocks); i++ {
		prev := c.blocks[i-1]
		curr := c.blocks[i]

		if curr.PreviousHash != prev.Hash {
			return false, errors.New("broken cryptographic link detected")
		}

		if c.calculateHash(&curr) != curr.Hash {
			return false, errors.New("block payload tampering detected")
		}
	}
	return true, nil
}
