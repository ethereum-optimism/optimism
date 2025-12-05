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
	return rollupNode.NewWithOverride(ctx, cfg, log, appVersion, m, nil, overrides)
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

var ErrL1AtSafeHeadNotFound = errors.New("l1 at safe head not found")

// L1AtSafeHead finds the earliest L1 block at which the provided L2 block became safe,
// using the monotonicity of SafeDB (L2 safe head number is non-decreasing over L1).
func (v *simpleVirtualNode) L1AtSafeHead(ctx context.Context, target eth.BlockID) (eth.BlockID, error) {
	v.mu.Lock()
	inner := v.inner
	v.mu.Unlock()
	if inner == nil {
		return eth.BlockID{}, ErrVirtualNodeNotRunning
	}
	db := inner.SafeDB()
	if db == nil {
		return eth.BlockID{}, ErrVirtualNodeNotRunning
	}
	// Get the latest entry to start the walkback
	latestL1, latestL2, err := db.SafeHeadAtL1(ctx, math.MaxUint64-1)
	if err != nil {
		v.log.Debug("L1AtSafeHead: latest lookup failed", "err", err)
		return eth.BlockID{}, err
	}
	v.log.Debug("L1AtSafeHead: latest bounds", "latest_l1", latestL1.Number, "latest_l2_num", latestL2.Number, "latest_l2_hash", latestL2.Hash)
	if latestL2.Number < target.Number {
		v.log.Debug("L1AtSafeHead: target beyond latest", "latest_l2", latestL2.Number)
		return eth.BlockID{}, ErrL1AtSafeHeadNotFound
	}
	// Walk back until the cursor would drop below the target
	cursor := latestL1
	genesisL1 := v.cfg.Rollup.Genesis.L1.Number
	for {
		if cursor.Number <= 0 || cursor.Number <= genesisL1 {
			// if we made it all the way back to genesis, it is likely the SafeDB is not stable enough for use
			// safer to simply return an error for now.
			v.log.Warn("L1AtSafeHead: reached genesis bound", "genesis_l1", genesisL1, "earliest_l1", cursor.Number)
			return eth.BlockID{}, ErrL1AtSafeHeadNotFound
		}
		prev := cursor.Number - 1
		v.log.Debug("L1AtSafeHead: checking previous l1 block", "l1_num", prev)
		l1Prev, l2Prev, err := db.SafeHeadAtL1(ctx, prev)
		if err != nil {
			v.log.Debug("L1AtSafeHead: walkback lookup failed, stopping", "probe_l1", prev, "err", err)
			break
		}
		v.log.Debug("L1AtSafeHead: walkback result", "l1_prev", l1Prev.Number, "l2_prev_num", l2Prev.Number, "l2_prev_hash", l2Prev.Hash)
		if l2Prev.Number >= target.Number {
			// Still meets or exceeds target; continue walking back
			cursor = l1Prev
			continue
		}
		// Dropped below target; current cursor is the first that meets/exceeds
		break
	}
	v.log.Debug("L1AtSafeHead: result", "l1", cursor)
	return cursor, nil
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
