package driver

import (
	"context"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/log"
)

// internalDriver exposes the driver functionality that maintains the external execution-engine.
type internalDriver interface {
	// requestEngineHead retrieves the L2 he```d reference of the engine, as well as the L1 reference it was derived from.
	// An error is returned when the L2 head information could not be retrieved (timeout or connection issue)
	requestEngineHead(ctx context.Context) (refL1 eth.BlockID, refL2 eth.BlockID, err error)
	// findSyncStart statelessly finds the next L1 block to derive, and on which L2 block it applies.
	// If the engine is fully synced, then the last derived L1 block, and parent L2 block, is repeated.
	// An error is returned if the sync starting point could not be determined (due to timeouts, wrong-chain, etc.)
	findSyncStart(ctx context.Context) (nextRefL1 eth.BlockID, refL2 eth.BlockID, err error)
	// driverStep explicitly calls the engine to derive a L1 block into a L2 block, and apply it on top of the given L2 block.
	// The finalized L2 block is provided to update the engine with finality, but does not affect the derivation step itself.
	// The resulting L2 block ID is returned, or an error if the derivation fails.
	driverStep(ctx context.Context, nextRefL1 eth.BlockID, refL2 eth.BlockID, finalized eth.BlockID) (l2ID eth.BlockID, err error)
}

type state struct {
	// l1Head tracks the L1 block corresponding to the l2Head
	l1Head eth.BlockID

	// l2Head tracks the head-block of the engine
	l2Head eth.BlockID

	// l2Finalized tracks the block the engine can safely regard as irreversible
	// (a week for disputes, or maybe shorter if we see L1 finalize and take the derived L2 chain up till there)
	l2Finalized eth.BlockID

	// The L1 block we are syncing towards, may be ahead of l1Head
	l1Target eth.BlockID

	// Rollup config
	Config rollup.Config
}

func (e *state) updateHead(l1Head eth.BlockID, l2Head eth.BlockID) {
	e.l1Head = l1Head
	e.l2Head = l2Head
}

// requestUpdate tries to update the state-machine with the driver head information.
// If the state-machine changed, considering the engine L1 and L2 head, it will return true. False otherwise.
func (e *state) requestUpdate(ctx context.Context, log log.Logger, driver internalDriver) (l2Updated bool) {
	refL1, refL2, err := driver.requestEngineHead(ctx)
	if err != nil {
		log.Error("failed to request engine head", "err", err)
		return false
	}

	e.l1Head = refL1
	e.l2Head = refL2
	return e.l1Head != refL1 || e.l2Head != refL2
}

// requestSync tries to sync the provided driver towards the sync target of the state-machine.
// If the L2 syncs a step, but is not finished yet, it will return true. False otherwise.
func (e *state) requestSync(ctx context.Context, log log.Logger, driver internalDriver) (l2Updated bool) {
	if e.l1Head == e.l1Target {
		log.Debug("Engine is fully synced", "l1_head", e.l1Head, "l2_head", e.l2Head)
		// TODO: even though we are fully synced, it may be worth attempting anyway,
		// in case the e.l1Head is not updating (failed/broken L1 head subscription)
		return false
	}
	log.Debug("finding next sync step, engine syncing", "l2", e.l2Head, "l1", e.l1Head)
	nextRefL1, refL2, err := driver.findSyncStart(ctx)
	if err != nil {
		log.Error("Failed to find sync starting point", "err", err)
		return false
	}
	if nextRefL1 == e.l1Head {
		log.Debug("Engine is already synced, aborting sync", "l1_head", e.l1Head, "l2_head", e.l2Head)
		return false
	}
	if l2ID, err := driver.driverStep(ctx, nextRefL1, refL2, e.l2Finalized); err != nil {
		log.Error("Failed to sync L2 chain with new L1 block", "l1", nextRefL1, "onto_l2", refL2, "err", err)
		return false
	} else {
		e.updateHead(nextRefL1, l2ID) // l2ID is derived from the nextRefL1
	}
	return e.l1Head != e.l1Target
}

// notifyL1Head updates the state-machine with the L1 signal,
// and attempts to sync the driver if the update extends the previous head.
// Returns true if the driver successfully derived and synced the L2 block to match L1. False otherwise.
func (e *state) notifyL1Head(ctx context.Context, log log.Logger, l1HeadSig eth.HeadSignal, driver internalDriver) (l2Updated bool) {
	if e.l1Head == l1HeadSig.Self {
		log.Debug("Received L1 head signal, already synced to it, ignoring event", "l1_head", e.l1Head)
		return
	}
	if e.l1Head == l1HeadSig.Parent {
		// Simple extend, a linear life is easy
		if l2ID, err := driver.driverStep(ctx, l1HeadSig.Self, e.l2Head, e.l2Finalized); err != nil {
			log.Error("Failed to extend L2 chain with new L1 block", "l1", l1HeadSig.Self, "l2", e.l2Head, "err", err)
			// Retry sync later
			e.l1Target = l1HeadSig.Self
			return false
		} else {
			e.updateHead(l1HeadSig.Self, l2ID)
			return true
		}
	}
	if e.l1Head.Number < l1HeadSig.Parent.Number {
		log.Debug("Received new L1 head, engine is out of sync, cannot immediately process", "l1", l1HeadSig.Self, "l2", e.l2Head)
	} else {
		log.Warn("Received a L1 reorg, syncing new alternative chain", "l1", l1HeadSig.Self, "l2", e.l2Head)
	}

	e.l1Target = l1HeadSig.Self
	return false
}
