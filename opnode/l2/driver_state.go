package l2

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum/go-ethereum/log"
)

// StateMachine provides control over the driver state, when given control over the Driver actions.
type StateMachine interface {
	// RequestUpdate tries to update the state-machine with the driver head information.
	// If the state-machine changed, considering the engine L1 and L2 head, it will return true. False otherwise.
	RequestUpdate(ctx context.Context, log log.Logger, driver Driver) (l2Updated bool)
	// RequestSync tries to sync the provided driver towards the sync target of the state-machine.
	// If the L2 syncs a step, but is not finished yet, it will return true. False otherwise.
	RequestSync(ctx context.Context, log log.Logger, driver Driver) (l2Updated bool)
	// NotifyL1Head updates the state-machine with the L1 signal,
	// and attempts to sync the driver if the update extends the previous head.
	// Returns true if the driver successfully derived and synced the L2 block to match L1. False otherwise.
	NotifyL1Head(ctx context.Context, log log.Logger, l1HeadSig eth.HeadSignal, driver Driver) (l2Updated bool)
}

type EngineDriverState struct {
	// Locks the L1 and L2 head changes, to keep a consistent view of the engine
	headLock sync.RWMutex

	// l1Head tracks the L1 block corresponding to the l2Head
	l1Head eth.BlockID

	// l2Head tracks the head-block of the engine
	l2Head eth.BlockID

	// l2Finalized tracks the block the engine can safely regard as irreversible
	// (a week for disputes, or maybe shorter if we see L1 finalize and take the derived L2 chain up till there)
	l2Finalized eth.BlockID

	// The L1 block we are syncing towards, may be ahead of l1Head
	l1Target eth.BlockID

	// Genesis starting point
	Genesis Genesis
}

// L1Head returns the block-id (hash and number) of the last L1 block that was derived into the L2 block
func (e *EngineDriverState) L1Head() eth.BlockID {
	e.headLock.RLock()
	defer e.headLock.RUnlock()
	return e.l1Head
}

// L2Head returns the block-id (hash and number) of the L2 chain head
func (e *EngineDriverState) L2Head() eth.BlockID {
	e.headLock.RLock()
	defer e.headLock.RUnlock()
	return e.l2Head
}

func (e *EngineDriverState) Head() (l1Head eth.BlockID, l2Head eth.BlockID) {
	e.headLock.RLock()
	defer e.headLock.RUnlock()
	return e.l1Head, e.l2Head
}

func (e *EngineDriverState) UpdateHead(l1Head eth.BlockID, l2Head eth.BlockID) {
	e.headLock.Lock()
	defer e.headLock.Unlock()
	e.l1Head = l1Head
	e.l2Head = l2Head
}

func (e *EngineDriverState) RequestUpdate(ctx context.Context, log log.Logger, driver Driver) (l2Updated bool) {
	refL1, refL2, err := driver.requestEngineHead(ctx)
	if err != nil {
		log.Error("failed to request engine head", "err", err)
		return false
	}
	e.headLock.Lock()
	defer e.headLock.Unlock()

	e.l1Head = refL1
	e.l2Head = refL2
	return e.l1Head != refL1 || e.l2Head != refL2
}

func (e *EngineDriverState) RequestSync(ctx context.Context, log log.Logger, driver Driver) (l2Updated bool) {
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
		e.UpdateHead(nextRefL1, l2ID) // l2ID is derived from the nextRefL1
	}
	return e.l1Head != e.l1Target
}

func (e *EngineDriverState) NotifyL1Head(ctx context.Context, log log.Logger, l1HeadSig eth.HeadSignal, driver Driver) (l2Updated bool) {
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
			e.UpdateHead(l1HeadSig.Self, l2ID)
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
