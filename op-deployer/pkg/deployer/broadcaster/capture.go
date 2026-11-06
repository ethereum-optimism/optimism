package broadcaster

import (
	"context"
	"fmt"
	"sync"
)

// CaptureBroadcaster retains broadcasts with every field so callers can
// validate and replay them.
type CaptureBroadcaster struct {
	broadcasts []Broadcast
	mu         sync.Mutex
}

func (c *CaptureBroadcaster) Hook(bcast Broadcast) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.broadcasts = append(c.broadcasts, bcast)
}

func (c *CaptureBroadcaster) Broadcast(context.Context) ([]BroadcastResult, error) {
	return nil, fmt.Errorf("capture broadcaster cannot broadcast transactions")
}

func (c *CaptureBroadcaster) Drain() []Broadcast {
	c.mu.Lock()
	defer c.mu.Unlock()

	broadcasts := c.broadcasts
	c.broadcasts = nil
	return broadcasts
}
