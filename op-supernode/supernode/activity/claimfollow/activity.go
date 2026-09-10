package claimfollow

import (
	"context"
	"sync"
	"time"

	gethlog "github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
)

// DefaultPollInterval is how often the module steps.
//
// It is the follow consumer's own cadence rather than a tuned number: the consumer polls this
// endpoint at 2x its block time, so stepping at one block time keeps the served state no more than
// a tick behind what the supernode has derived, and costs one sync-status read per tick on the
// polls that find nothing new.
const DefaultPollInterval = 2 * time.Second

// Activity runs the follow module's poll loop under the supernode's activity lifecycle.
//
// It is deliberately NOT an activity.RPCActivity. An RPCActivity's namespace is mounted on the
// supernode's ROOT handler, and this module's whole point is that it answers at a DISTINCT
// per-chain route; the API is handed to the chain container instead, via
// chain_container.WithExtraRPCRoutes.
type Activity struct {
	log      gethlog.Logger
	module   *Module
	chainID  eth.ChainID
	interval time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
}

var _ activity.RunnableActivity = (*Activity)(nil)

// NewActivity wraps a module for the supernode's activity list.
func NewActivity(lgr gethlog.Logger, chainID eth.ChainID, m *Module, interval time.Duration) *Activity {
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	return &Activity{log: lgr, module: m, chainID: chainID, interval: interval}
}

// Module exposes the state machine, so the caller can build the API that is mounted on the chain's
// own route.
func (a *Activity) Module() *Module { return a.module }

func (a *Activity) Name() string { return "claim-follow" }

// ChainID is the rendering chain the module reads.
func (a *Activity) ChainID() eth.ChainID { return a.chainID }

// Start runs the poll loop until the supernode's lifecycle context is cancelled.
func (a *Activity) Start(ctx context.Context) error {
	loopCtx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()
	defer cancel()
	a.log.Info("claim follow module started",
		"chain_id", a.chainID.String(), "registry", a.module.cfg.Registry,
		"startBlock", a.module.cfg.StartBlock, "pollInterval", a.interval)
	return a.module.Run(loopCtx, a.interval)
}

// Stop ends the poll loop.
func (a *Activity) Stop(_ context.Context) error {
	a.mu.Lock()
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

// Reset drops everything the module learned above an invalidated block, for the chain it follows.
//
// It is a defensive echo of the module's own reorg handling rather than the primary mechanism: the
// scan only ever reads at or below the chain's SAFE view, and an interop invalidation rewinds
// blocks above it, so in practice there is nothing above the cursor to drop. Rewinding anyway costs
// a rescan and cannot lower a served label — the module's refs are high-water marks — which makes
// this the cheap half of a cheap/expensive pair.
func (a *Activity) Reset(chainID eth.ChainID, _ uint64, invalidatedBlock eth.BlockRef) {
	if chainID != a.chainID || invalidatedBlock.Number == 0 {
		return
	}
	a.log.Info("claim follow module rewinding after a chain reset",
		"chain_id", chainID.String(), "invalidated", invalidatedBlock.Number)
	a.module.rewind(invalidatedBlock.Number - 1)
}
