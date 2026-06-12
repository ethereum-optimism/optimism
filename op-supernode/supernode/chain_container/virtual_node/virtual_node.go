package virtual_node

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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
	ErrVirtualNodeStartGated     = errors.New("virtual node start gated (chain paused/stopping)")
)

type VirtualNode interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error)
	// L1AtSafeHead returns the earliest L1 block at which the given L2 block became safe.
	L1AtSafeHead(ctx context.Context, target eth.BlockID) (eth.BlockID, error)
	// FirstSafeHeadEntry returns the lowest recorded (L1, L2 safe head) pair from SafeDB.
	// Returns safedb.ErrNotFound when SafeDB has no entries yet.
	FirstSafeHeadEntry(ctx context.Context) (eth.BlockID, eth.BlockID, error)
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	State() VNState
	// SetBeforeStart installs a predicate consulted under the VN's lock,
	// atomically with the NotStarted->Running transition. If it returns false,
	// Start aborts with ErrVirtualNodeStartGated without running the inner node.
	// The chain container uses this to close the freeze race: a VN created but
	// not yet started cannot slip past a concurrent PauseAndStopVN.
	SetBeforeStart(fn func() bool)
}

type innerNode interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SafeDB() rollupNode.SafeDBReader
	SyncStatus() *eth.SyncStatus
}

type innerNodeFactory func(ctx context.Context, cfg *opnodecfg.Config, log gethlog.Logger, appVersion string, m *opmetrics.Metrics, initOverload *rollupNode.InitializationOverrides) (innerNode, error)

type VNState int32

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

	mu          sync.Mutex   // Coordinates Start/Stop transitions and protects cancel + beforeStart.
	state       atomic.Int32 // Current lifecycle state; stores VNState values.
	cancel      context.CancelFunc
	beforeStart func() bool // Optional gate checked under mu before NotStarted->Running.
}

func generateVirtualNodeID() string {
	return uuid.New().String()[:4]
}

func NewVirtualNode(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) *simpleVirtualNode {
	vnID := generateVirtualNodeID()
	l := log.New("chain_id", cfg.Rollup.L2ChainID.String(), "vn_id", vnID)
	vn := &simpleVirtualNode{
		vnID:             vnID,
		cfg:              cfg,
		log:              l,
		initOverload:     initOverload,
		appVersion:       appVersion,
		innerNodeFactory: defaultInnerNodeFactory,
	}
	vn.setState(VNStateNotStarted)
	return vn
}

func (v *simpleVirtualNode) Start(ctx context.Context) error {
	// Accquire lock while setting up inner node
	v.mu.Lock()
	if state := v.State(); state != VNStateNotStarted {
		v.mu.Unlock()
		v.log.Debug("virtual node not in a valid state to start", "state", state)
		return ErrVirtualNodeCantStart
	}
	// Consult the start gate under the same lock that Stop() takes, atomically
	// with the NotStarted->Running transition below. This closes the freeze
	// race: PauseAndStopVN sets the chain's pause flag before calling Stop(), so
	// this VN either observes the pause here and aborts, or completes its
	// transition under the lock and is then reliably stopped by that Stop().
	if v.beforeStart != nil && !v.beforeStart() {
		v.setState(VNStateStopped)
		v.mu.Unlock()
		v.log.Info("virtual node start gated by chain pause/stop")
		return ErrVirtualNodeStartGated
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
		v.setState(VNStateStopped)
		v.mu.Unlock()
		return err
	}
	v.inner = n
	v.setState(VNStateRunning)
	v.mu.Unlock()

	// Run inner node in goroutine and await any signal to exit (Stop(), parent
	// ctx, or inner error). innerErr (n.Start's return) is delivered over a
	// buffered channel rather than a shared variable, so the read below
	// happens-after the goroutine. cancelErr is written by the Cancel callback
	// on the op-node event-loop goroutine (not from within n.Start); its
	// happens-before for the read below is established separately by n.Stop()
	// → eventSys.Stop() joining that event loop. Do NOT move the cancelErr read
	// above n.Stop() — the channel receive alone does not synchronize it.
	innerErrCh := make(chan error, 1)
	go func() {
		innerErrCh <- n.Start(runCtx)
	}()
	<-runCtx.Done()

	// Update state under lock, but do NOT hold the lock during inner.Stop().
	// inner.Stop() drains the op-node event system, which may call back into
	// this VirtualNode (e.g. SyncStatus via EngineController.FinalizedHead).
	// SyncStatus needs v.mu, so holding it here would deadlock.
	v.mu.Lock()
	v.setState(VNStateStopped)
	v.cancel = nil
	v.mu.Unlock()

	// Stop the inner node outside the lock. Use n which is the local reference
	// to the inner node created at the top of this function.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()
	if err := n.Stop(stopCtx); err != nil {
		v.log.Error("error stopping inner node", "err", err)
	}

	// Wait for n.Start to return before reading its result; this receive
	// synchronizes the innerErr/cancelErr reads below with the goroutine.
	innerErr := <-innerErrCh

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

	if v.State() != VNStateRunning {
		return nil // Already stopped or not started
	}

	// Keep this setState before the cancel() below; do not reorder them.
	// The per-chain RPC route gate reads this state to decide readiness, and it
	// must observe VNStateStopped ("not ready") before the run context starts
	// unwinding. If the cancel ran first, a request could still be routed to a
	// chain handler that is already tearing down.
	v.setState(VNStateStopped)

	// Cancel the run context to trigger shutdown
	if v.cancel != nil {
		v.cancel()
	}

	return nil
}

// SetBeforeStart installs the start gate. Must be called before Start; the
// chain container sets it once per VN (in the same goroutine that then calls
// Start), so the field is written-before-read without contention. Guarded by mu
// for safety against any future concurrent caller.
func (v *simpleVirtualNode) SetBeforeStart(fn func() bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.beforeStart = fn
}

// State returns the current state of the virtual node (for testing and monitoring)
func (v *simpleVirtualNode) State() VNState {
	return VNState(v.state.Load())
}

// String returns a stable identifier for logging. Without it, logging a
// VirtualNode value ("vn_id", vn) reflectively dumps the whole struct —
// reading the mutex/atomic state/fields while the run goroutine mutates them,
// a data race (and a giant log line). vnID is set once at construction and
// never mutated, so reading it here is race-free.
func (v *simpleVirtualNode) String() string {
	return "vn:" + v.vnID
}

func (v *simpleVirtualNode) setState(state VNState) {
	v.state.Store(int32(state))
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

func (v *simpleVirtualNode) FirstSafeHeadEntry(ctx context.Context) (eth.BlockID, eth.BlockID, error) {
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
	return db.FirstEntry(ctx)
}

// Re-exported from safedb for callers that still reference these via virtual_node.
var (
	ErrL1AtSafeHeadNotFound    = safedb.ErrL1AtSafeHeadNotFound
	ErrL1AtSafeHeadUnavailable = safedb.ErrL1AtSafeHeadUnavailable
)

// L1AtSafeHead returns the earliest L1 block at which the provided L2 block
// became local-safe, delegating the lookup to SafeDB.
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

	// Genesis L2 is trivially safe at L1 block 0. Use 0 rather than
	// cfg.Genesis.L1 because contracts may pre-date cfg.Genesis.L1, allowing
	// dispute games anchored to earlier L1 heads.
	if target == v.cfg.Rollup.Genesis.L2 {
		return eth.BlockID{Number: 0}, nil
	}

	l1, _, err := db.L1AtSafeHead(ctx, target.Number)
	if err != nil {
		v.log.Debug("L1AtSafeHead: lookup failed",
			"target_l2_num", target.Number, "target_l2_hash", target.Hash, "err", err)
		return eth.BlockID{}, err
	}
	v.log.Debug("L1AtSafeHead: result",
		"target_l2_num", target.Number, "target_l2_hash", target.Hash, "l1", l1)
	return l1, nil
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
