package syncnode

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// managedNodeResetBackend is a shim to pass to the resetTracker to let it
// query information from the node and DB that it needs during bisection.
type managedNodeResetBackend struct {
	chainID eth.ChainID
	node    SyncControl
	backend backend
}

var _ resetBackend = (*managedNodeResetBackend)(nil)

func (m *managedNodeResetBackend) BlockIDByNumber(ctx context.Context, n uint64) (eth.BlockID, error) {
	r, err := m.node.BlockRefByNumber(ctx, n)
	return r.ID(), err
}

func (m *managedNodeResetBackend) IsLocalSafe(ctx context.Context, block eth.BlockID) error {
	return m.backend.IsLocalSafe(ctx, m.chainID, block)
}

func (m *ManagedNode) resetBackend() *managedNodeResetBackend {
	return &managedNodeResetBackend{
		chainID: m.chainID,
		node:    m.Node,
		backend: m.backend,
	}
}

func (m *ManagedNode) initiateReset(z eth.BlockID) {
	m.resetMu.Lock()
	defer m.resetMu.Unlock()
	var ctx context.Context
	ctx, m.resetCancel = context.WithCancel(m.ctx)
	defer func() { m.resetCancel = nil }()
	defer m.resetCancel()

	start, err := m.backend.ActivationBlock(ctx, m.chainID)
	if errors.Is(err, types.ErrFuture) {
		m.log.Info("no activation block yet, initiating pre-Interop reset", "err", err)
		m.emitter.Emit(superevents.ResetPreInteropRequestEvent{
			ChainID: m.chainID, Ctx: event.WrapCtx(m.ctx)})
		return
	} else if err != nil {
		m.log.Error("failed to get activation block, cancelling reset", "err", err)
		return
	}

	target, err := m.resetTracker.FindResetTarget(ctx, start.Derived.ID(), z)
	if err != nil {
		m.log.Error("failed to find reset target, cancelling reset", "err", err)
		return
	} else if target.PreInterop {
		m.log.Info("bisection results in pre-Interop reset")
		m.emitter.Emit(superevents.ResetPreInteropRequestEvent{
			ChainID: m.chainID, Ctx: event.WrapCtx(m.ctx)})
		return
	}
	m.log.Info("bisection found reset target", "target", target.Target)
	m.resetHeadsFromTarget(ctx, target.Target)
}

// resetHeadsFromTarget takes a target block and identifies the correct
// unsafe, safe, and finalized blocks to target for the reset.
// It then triggers the reset on the node.
func (t *ManagedNode) resetHeadsFromTarget(ctx context.Context, target eth.BlockID) {
	iCtx, iCancel := context.WithTimeout(ctx, internalTimeout)
	defer iCancel()

	var lUnsafe, xUnsafe, lSafe, xSafe, finalized eth.BlockID

	// We set the local unsafe block to our target (the local-safe block we determined to reset to).
	// The node checks it for consistency, but if it builds on this target,
	// it does not revert back the existing unsafe chain.
	// We do not have to pick the latest possible unsafe target here.
	lUnsafe = target

	// all other blocks are either the last consistent block, or the last block in the db, whichever is earlier
	// cross unsafe
	lastXUnsafe, err := t.backend.CrossUnsafe(iCtx, t.chainID)
	if err != nil {
		t.log.Error("failed to get last cross unsafe block. cancelling reset", "err", err)
		return
	}
	if lastXUnsafe.Number < target.Number {
		xUnsafe = lastXUnsafe
	} else {
		xUnsafe = target
	}

	// local safe
	lSafe = target

	// cross safe
	lastXSafe, err := t.backend.CrossSafe(iCtx, t.chainID)
	if err != nil {
		t.log.Error("failed to get last cross safe block. cancelling reset", "err", err)
		return
	}
	if lastXSafe.Derived.Number < target.Number {
		xSafe = lastXSafe.Derived
	} else {
		xSafe = target
		// TODO(#16026): investigate, maybe instead return an error that cross-safe changed.
		// Resetting to older block should be unneeded.
		// Note: op-node may not have the same blocks as op-supervisor has,
		// and thus needs to start from an old forkchoice state.
	}

	// finalized
	lastFinalized, err := t.backend.Finalized(iCtx, t.chainID)
	if errors.Is(err, types.ErrFuture) {
		t.log.Warn("finalized block is not yet known", "err", err)
		lastFinalized = eth.BlockID{}
	} else if err != nil {
		t.log.Error("failed to get last finalized block. cancelling reset", "err", err)
		return
	}
	if lastFinalized.Number < target.Number {
		finalized = lastFinalized
	} else {
		finalized = target
		// TODO(#16026): same story as cross-safe.
	}

	// trigger the reset
	t.log.Info("triggering reset on node",
		"localUnsafe", lUnsafe,
		"crossUnsafe", xUnsafe,
		"localSafe", lSafe,
		"crossSafe", xSafe,
		"finalized", finalized)

	nCtx, nCancel := context.WithTimeout(ctx, nodeTimeout)
	defer nCancel()
	if err := t.Node.Reset(nCtx,
		lUnsafe, xUnsafe,
		lSafe, xSafe,
		finalized); err != nil {
		t.log.Error("Failed to reset node", "err", err)
	}
}
