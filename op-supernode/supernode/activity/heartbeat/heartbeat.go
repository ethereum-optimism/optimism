package heartbeat

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// compile time assertions
var (
	_ activity.RunnableActivity = (*Heartbeat)(nil)
	_ activity.RPCActivity      = (*Heartbeat)(nil)
)

// Activity that emits periodic heartbeats and exposes a simple liveness RPC.
type Heartbeat struct {
	log      gethlog.Logger
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// New creates a new Heartbeat activity.
func New(log gethlog.Logger, interval time.Duration) *Heartbeat {
	return &Heartbeat{log: log, interval: interval}
}

// Start begins the periodic logging loop.
func (h *Heartbeat) Start(ctx context.Context) error {
	if h.interval <= 0 {
		h.interval = time.Second
	}
	h.ctx, h.cancel = context.WithCancel(ctx)
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-h.ctx.Done():
			return h.ctx.Err()
		case <-ticker.C:
			h.log.Info("heartbeat")
		}
	}
}

// Stop stops the heartbeat loop.
func (h *Heartbeat) Stop(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}
	return nil
}

// RPCNamespace returns the JSON-RPC namespace for this activity.
func (h *Heartbeat) RPCNamespace() string { return "heartbeat" }

// RPCService returns the service object whose exported methods are exposed in RPC.
func (h *Heartbeat) RPCService() interface{} { return (*api)(h) }

// api hosts JSON-RPC methods for the Heartbeat activity.
type api Heartbeat

// Check returns a random 4-byte for liveness.
func (a *api) Check(ctx context.Context) (hexutil.Bytes, error) {
	buf := make([]byte, 4)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}
	return hexutil.Bytes(buf), nil
}
