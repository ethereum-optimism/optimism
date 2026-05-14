package virtual_node

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

// VIRTUAL_NODE_CHAIN_ID_LABEL is the name of the label used to differentiate
// metrics registered by virtual nodes.
const VIRTUAL_NODE_CHAIN_ID_LABEL = "virtual_node_chain_id"

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
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
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
	additionalLabels := map[string]string{VIRTUAL_NODE_CHAIN_ID_LABEL: v.cfg.Rollup.L2ChainID.String()}
	m := opmetrics.NewMetrics("supernode", additionalLabels)
	n, err := v.innerNodeFactory(runCtx, v.cfg, v.log, v.appVersion, m, v.initOverload)
	if err != nil {
		v.state = VNStateStopped
		v.mu.Unlock()
		return err
	}
	v.inner = n
	v.state = VNStateRunning
	v.mu.Unlock()

	// Run inner node in goroutine
	// and await any signal to exit (Stop(), parent ctx, or inner error)
	var innerErr error = nil
	go func() {
		innerErr = n.Start(runCtx)
	}()
	<-runCtx.Done()

	// Update state under lock, but do NOT hold the lock during inner.Stop().
	// inner.Stop() drains the op-node event system, which may call back into
	// this VirtualNode (e.g. SyncStatus via EngineController.FinalizedHead).
	// SyncStatus needs v.mu, so holding it here would deadlock.
	v.mu.Lock()
	v.state = VNStateStopped
	v.cancel = nil
	v.mu.Unlock()

	// Stop the inner node outside the lock. Use n which is the local reference
	// to the inner node created at the top of this function.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()
	if err := n.Stop(stopCtx); err != nil {
		v.log.Error("error stopping inner node", "err", err)
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

// ErrL1AtSafeHeadNotFound: transient — SafeDB hasn't observed the answer yet
// (target ahead of latest, or DB empty at startup). Retry.
var ErrL1AtSafeHeadNotFound = errors.New("l1 at safe head not found")

// ErrL1AtSafeHeadUnavailable: permanent on this node — the crossing happened
// before SafeDB started recording (snap/CL-sync bootstrap), or the walkback
// reached the genesis bound. Retrying won't help; operator must intervene.
var ErrL1AtSafeHeadUnavailable = errors.New("l1 at safe head history unavailable")

// L1AtSafeHead finds the earliest L1 block at which the provided L2 block became local safe,
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

	// Special case: genesis L2 block is trivially safe at genesis L1
	// Note: We use L1 block 0 (not cfg.Genesis.L1) because contracts may have been deployed
	// earlier than cfg.Genesis.L1, allowing dispute games with L1 heads prior to cfg.Genesis.L1
	if target == v.cfg.Rollup.Genesis.L2 {
		// Return L1 block 0 (L1 genesis)
		l1Genesis := eth.BlockID{Number: 0} // Hash not necessary
		return l1Genesis, nil
	}

	// Get the latest entry to start the walkback
	latestL1, latestL2, err := db.SafeHeadAtL1(ctx, math.MaxUint64-1)
	if err != nil {
		// Empty DB on startup is transient; anything else is a real failure.
		if errors.Is(err, safedb.ErrNotFound) {
			v.log.Debug("L1AtSafeHead: SafeDB empty, no entries yet",
				"target_l2_num", target.Number, "target_l2_hash", target.Hash)
			return eth.BlockID{}, ErrL1AtSafeHeadNotFound
		}
		v.log.Debug("L1AtSafeHead: latest lookup failed", "err", err)
		return eth.BlockID{}, err
	}
	v.log.Debug("L1AtSafeHead: latest bounds", "latest_l1", latestL1.Number, "latest_l2_num", latestL2.Number, "latest_l2_hash", latestL2.Hash)
	if latestL2.Number < target.Number {
		v.log.Debug("L1AtSafeHead: target beyond latest", "latest_l2", latestL2.Number, "target", target.Number)
		return eth.BlockID{}, ErrL1AtSafeHeadNotFound
	}
	v.log.Debug("L1AtSafeHead: target within latest", "latest_l2", latestL2.Number, "target", target.Number)
	// Walk back until the cursor would drop below the target. cursor tracks
	// the earliest entry we've successfully resolved; on failure it is the
	// first (earliest) recorded SafeDB entry, which is the most useful piece
	// of diagnostic context for the operator.
	cursor := latestL1
	cursorL2 := latestL2
	genesisL1 := v.cfg.Rollup.Genesis.L1.Number
	steps := 0
	for {
		steps++
		if cursor.Number <= 0 || cursor.Number <= genesisL1 {
			// Walkback crossed the genesis bound without ever dropping below
			// target: the crossing is older than anything we have. Permanent.
			v.log.Warn("L1AtSafeHead: reached genesis bound without crossing target",
				"target_l2_num", target.Number, "target_l2_hash", target.Hash,
				"earliest_l1", cursor.Number, "earliest_l2", cursorL2.Number,
				"genesis_l1", genesisL1)
			return eth.BlockID{}, ErrL1AtSafeHeadUnavailable
		}
		prev := cursor.Number - 1
		l1Prev, l2Prev, err := db.SafeHeadAtL1(ctx, prev)
		if err != nil {
			// Probed below the earliest SafeDB entry: snap/CL-sync bootstrap
			// gap. If the earliest entry is the exact target, it is still a
			// valid lower bound because SafeDB recorded that L2 at cursor L1.
			// Otherwise the target predates available history.
			// cursor is the earliest entry in the DB (nothing exists at
			// or below cursor.Number - 1, which is what we just probed).
			if errors.Is(err, safedb.ErrNotFound) {
				if cursorL2 == target {
					v.log.Debug("L1AtSafeHead: target matches earliest SafeDB entry",
						"target_l2_num", target.Number, "target_l2_hash", target.Hash,
						"earliest_l1", cursor.Number)
					return cursor, nil
				}
				v.log.Warn("L1AtSafeHead: walkback ran past earliest SafeDB entry",
					"target_l2_num", target.Number, "target_l2_hash", target.Hash,
					"earliest_l1", cursor.Number, "earliest_l2", cursorL2.Number,
					"probe_l1", prev, "genesis_l1", genesisL1)
				return eth.BlockID{}, ErrL1AtSafeHeadUnavailable
			}
			v.log.Error("L1AtSafeHead: walkback lookup failed, stopping",
				"target_l2_num", target.Number, "target_l2_hash", target.Hash,
				"earliest_l1", cursor.Number, "earliest_l2", cursorL2.Number,
				"probe_l1", prev, "err", err)
			return eth.BlockID{}, err
		}
		if l2Prev.Number >= target.Number {
			// Still meets or exceeds target; continue walking back
			cursor = l1Prev
			cursorL2 = l2Prev
			continue
		}
		// Dropped below target; current cursor is the first that meets/exceeds
		break
	}
	v.log.Debug("L1AtSafeHead: result", "l1", cursor, "steps", steps)
	return cursor, nil
}

func (v *simpleVirtualNode) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	v.mu.Lock()
	inner := v.inner
	v.mu.Unlock()
	if inner == nil {
		return nil, ErrVirtualNodeNotRunning
	}
	st := inner.SyncStatus()
	cpy := *st
	return &cpy, nil
}
