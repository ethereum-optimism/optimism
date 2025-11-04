package virtual_node

import (
	"context"
	"errors"
	"math"
	"sync"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

// defaultInnerNodeFactory is the default factory that creates a real op-node
func defaultInnerNodeFactory(ctx context.Context, cfg *opnodecfg.Config, log gethlog.Logger, appVersion string, m *opmetrics.Metrics, initOverload *rollupNode.InitializationOverrides) (innerNode, error) {
	var overrides rollupNode.InitializationOverrides
	if initOverload != nil {
		overrides = *initOverload
	}
	return rollupNode.NewWithOverride(ctx, cfg, log, appVersion, m, overrides)
}

var (
	ErrVirtualNodeConfigNil      = errors.New("virtual node config is nil")
	ErrVirtualNodeAlreadyRunning = errors.New("virtual node already running")
	ErrVirtualNodeNotRunning     = errors.New("virtual node not running")
	ErrVirtualNodeCantStart      = errors.New("virtual node cannot be started in this state")
)

type VirtualNode interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error)
	// L1AtSafeHead returns the earliest L1 block at which the given L2 block became safe.
	L1AtSafeHead(ctx context.Context, target eth.BlockID) (eth.BlockID, error)
	CurrentL1(ctx context.Context) (eth.BlockRef, error)
}

type innerNode interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SafeDB() rollupNode.SafeDBReader
	SyncStatus() *eth.SyncStatus
}

type innerNodeFactory func(ctx context.Context, cfg *opnodecfg.Config, log gethlog.Logger, appVersion string, m *opmetrics.Metrics, initOverload *rollupNode.InitializationOverrides) (innerNode, error)

type VNState int

const (
	VNStateNotStarted VNState = iota
	VNStateRunning
	VNStateStopped
)

type simpleVirtualNode struct {
	log        gethlog.Logger
	vnID       string
	appVersion string

	inner            innerNode                           // Inner node instance
	cfg              *opnodecfg.Config                   // op-node config for the virtual node
	initOverload     *rollupNode.InitializationOverrides // Shared resources which are overridden by the supernode
	innerNodeFactory innerNodeFactory                    // Factory function to create inner node (overloadable for testing)

	mu     sync.Mutex         // Protects state transitions
	state  VNState            // Current lifecycle state
	cancel context.CancelFunc // Cancels the running context
}

func generateVirtualNodeID() string {
	return uuid.New().String()[:4]
}

func NewVirtualNode(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) *simpleVirtualNode {
	vnID := generateVirtualNodeID()
	l := log.New("chain_id", cfg.Rollup.L2ChainID.String(), "vn_id", vnID)
	return &simpleVirtualNode{
		vnID:             vnID,
		cfg:              cfg,
		log:              l,
		initOverload:     initOverload,
		appVersion:       appVersion,
		innerNodeFactory: defaultInnerNodeFactory,
		state:            VNStateNotStarted,
	}
}

func (v *simpleVirtualNode) Start(ctx context.Context) error {
	// Accquire lock while setting up inner node
	v.mu.Lock()
	if v.state != VNStateNotStarted {
		v.mu.Unlock()
		v.log.Debug("virtual node not in a valid state to start", "state", v.state)
		return ErrVirtualNodeCantStart
	}
	if v.cfg == nil {
		v.mu.Unlock()
		return ErrVirtualNodeConfigNil
	}

	runCtx, cancel := context.WithCancel(ctx)
	v.cancel = cancel

	// Capture inner node errors via cancel callback
	var cancelErr error
	v.cfg.Cancel = func(err error) {
		cancelErr = err
		cancel() // Cancel the run context when inner node fails
	}

	// Create and start the inner node
	m := opmetrics.NewMetrics("supernode")
	n, err := v.innerNodeFactory(runCtx, v.cfg, v.log, v.appVersion, m, v.initOverload)
	if err != nil {
		v.state = VNStateStopped
		v.mu.Unlock()
		return err
	}
	v.inner = n
	// Release the lock once the inner node is created
	v.state = VNStateRunning
	v.mu.Unlock()
	// Don't hold the lock while running or waiting for inner node to stop

	// Run inner node in goroutine
	// and await any signal to exit (Stop(), parent ctx, or inner error)
	var innerErr error = nil
	go func() {
		innerErr = v.inner.Start(runCtx)
	}()
	<-runCtx.Done()

	// Clean up with lock to end of function
	v.mu.Lock()
	defer v.mu.Unlock()
	v.state = VNStateStopped
	v.cancel = nil

	// Stop the inner node if it's still running
	if v.inner != nil {
		stopCtx := context.Background()
		if err := v.inner.Stop(stopCtx); err != nil {
			v.log.Error("error stopping inner node", "err", err)
		}
	}

	// Return inner error if that's what caused the cancellation, otherwise context error
	if cancelErr != nil {
		v.log.Warn("virtual node stopped due to inner cancel error", "err", cancelErr)
		return cancelErr
	}
	if innerErr != nil {
		v.log.Warn("virtual node stopped due to inner error", "err", innerErr)
		return innerErr
	}
	if ctx.Err() != nil {
		v.log.Warn("virtual node stopped due to context cancellation", "err", ctx.Err())
		return ctx.Err()
	}
	v.log.Info("virtual node stopped")
	return nil
}

func (v *simpleVirtualNode) Stop(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.state != VNStateRunning {
		return nil // Already stopped or not started
	}

	// Cancel the run context to trigger shutdown
	if v.cancel != nil {
		v.cancel()
	}

	return nil
}

// State returns the current state of the virtual node (for testing and monitoring)
func (v *simpleVirtualNode) State() VNState {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.state
}

// SafeHeadAtL1 returns the recorded mapping of L1 block -> L2 safe head at or before the given L1 block number.
func (v *simpleVirtualNode) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	v.mu.Lock()
	inner := v.inner
	v.mu.Unlock()
	if inner == nil {
		return eth.BlockID{}, eth.BlockID{}, ErrVirtualNodeNotRunning
	}
	db := inner.SafeDB()
	if db == nil {
		return eth.BlockID{}, eth.BlockID{}, ErrVirtualNodeNotRunning
	}
	return db.SafeHeadAtL1(ctx, l1BlockNum)
}

// L1AtSafeHead finds the earliest L1 block at which the provided L2 block became safe,
// using the monotonicity of SafeDB (L2 safe head number is non-decreasing over L1).
func (v *simpleVirtualNode) L1AtSafeHead(ctx context.Context, target eth.BlockID) (eth.BlockID, error) {
	v.mu.Lock()
	inner := v.inner
	v.mu.Unlock()
	if v.log != nil {
		v.log.Debug("L1AtSafeHead: start", "target_num", target.Number, "target_hash", target.Hash)
	}
	if inner == nil {
		return eth.BlockID{}, ErrVirtualNodeNotRunning
	}
	db := inner.SafeDB()
	if db == nil {
		return eth.BlockID{}, ErrVirtualNodeNotRunning
	}
	// Get the latest entry to bound the search space
	latestL1, latestL2, err := db.SafeHeadAtL1(ctx, math.MaxUint64-1)
	if err != nil {
		if v.log != nil {
			v.log.Debug("L1AtSafeHead: latest lookup failed", "err", err)
		}
		return eth.BlockID{}, err
	}
	if v.log != nil {
		v.log.Debug("L1AtSafeHead: latest bounds", "latest_l1", latestL1.Number, "latest_l2_num", latestL2.Number, "latest_l2_hash", latestL2.Hash)
	}
	if latestL2.Number < target.Number {
		if v.log != nil {
			v.log.Debug("L1AtSafeHead: target beyond latest", "latest_l2", latestL2.Number)
		}
		return eth.BlockID{}, errors.New("target not found")
	}
	// Restrict lower bound to rollup genesis L1 (the rollup starts after this L1)
	var lo uint64 = v.cfg.Rollup.Genesis.L1.Number
	hi := latestL1.Number
	if v.log != nil {
		v.log.Debug("L1AtSafeHead: initial bounds", "lo", lo, "hi", hi)
	}
	for lo < hi {
		mid := (lo + hi) / 2
		if v.log != nil {
			v.log.Debug("L1AtSafeHead: probe", "mid", mid, "lo", lo, "hi", hi)
		}
		_, midL2, err := db.SafeHeadAtL1(ctx, mid)
		if err != nil {
			// before first entry; treat as below target
			if v.log != nil {
				v.log.Debug("L1AtSafeHead: mid lookup failed, advance lo", "mid", mid, "err", err)
			}
			lo = mid + 1
			continue
		}
		if v.log != nil {
			v.log.Debug("L1AtSafeHead: mid result", "mid", mid, "mid_l2_num", midL2.Number, "mid_l2_hash", midL2.Hash)
		}
		if midL2.Number >= target.Number {
			if v.log != nil {
				v.log.Debug("L1AtSafeHead: move hi", "from", hi, "to", mid)
			}
			hi = mid
		} else {
			if v.log != nil {
				v.log.Debug("L1AtSafeHead: move lo", "from", lo, "to", mid+1)
			}
			lo = mid + 1
		}
	}
	// Validate match at boundary
	if v.log != nil {
		v.log.Debug("L1AtSafeHead: boundary", "lo", lo)
	}
	fL1, fL2, err := db.SafeHeadAtL1(ctx, lo)
	if err != nil {
		if v.log != nil {
			v.log.Debug("L1AtSafeHead: boundary lookup failed", "lo", lo, "err", err)
		}
		return eth.BlockID{}, err
	}
	if v.log != nil {
		v.log.Debug("L1AtSafeHead: boundary result", "l1", fL1.Number, "l2_num", fL2.Number, "l2_hash", fL2.Hash)
	}
	// If the exact L2 is found, return its L1; otherwise, return the earliest L1
	// at which the safe head number is >= the target (implied availability).
	if fL2.Number == target.Number && fL2.Hash == target.Hash {
		if v.log != nil {
			v.log.Debug("L1AtSafeHead: found", "l1", fL1.Number)
		}
		return fL1, nil
	}
	if fL2.Number >= target.Number {
		if v.log != nil {
			v.log.Debug("L1AtSafeHead: implied at boundary", "implied_l1", fL1.Number)
		}
		return fL1, nil
	}
	if v.log != nil {
		v.log.Debug("L1AtSafeHead: not found (unexpected)")
	}
	return eth.BlockID{}, errors.New("target not found")
}

// CurrentL1 returns the current processed L1 block based on derivation pipeline sync status.
func (v *simpleVirtualNode) CurrentL1(ctx context.Context) (eth.BlockRef, error) {
	v.mu.Lock()
	inner := v.inner
	v.mu.Unlock()
	if inner == nil {
		return eth.BlockRef{}, ErrVirtualNodeNotRunning
	}
	st := inner.SyncStatus()
	// Map L1 block ref into generic block ref
	return eth.BlockRef{
		Hash:       st.CurrentL1.Hash,
		Number:     st.CurrentL1.Number,
		ParentHash: st.CurrentL1.ParentHash,
		Time:       st.CurrentL1.Time,
	}, nil
}
