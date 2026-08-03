package broadcaster

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

// CaptureBroadcaster retains broadcasts in their script-native form so callers can
// validate and replay fields that CalldataBroadcaster intentionally discards.
type CaptureBroadcaster struct {
	broadcasts []script.Broadcast
	mu         sync.Mutex
}

func (c *CaptureBroadcaster) Hook(bcast script.Broadcast) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.broadcasts = append(c.broadcasts, bcast)
}

func (c *CaptureBroadcaster) Broadcast(context.Context) ([]BroadcastResult, error) {
	return nil, fmt.Errorf("capture broadcaster cannot broadcast transactions")
}

func (c *CaptureBroadcaster) Drain() []script.Broadcast {
	c.mu.Lock()
	defer c.mu.Unlock()

	broadcasts := c.broadcasts
	c.broadcasts = nil
	return broadcasts
}
