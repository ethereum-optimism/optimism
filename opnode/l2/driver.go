package l2

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"

	"github.com/ethereum/go-ethereum/log"
)

type EngineDriver struct {
	Log log.Logger
	// API bindings to execution engine
	RPC DriverAPI

	SyncRef SyncReference

	// The current driving force, to shutdown before closing the engine.
	driveSub ethereum.Subscription
	// There may only be 1 driver at a time
	driveLock sync.Mutex

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
func (e *EngineDriver) L1Head() eth.BlockID {
	e.headLock.RLock()
	defer e.headLock.RUnlock()
	return e.l1Head
}

// L2Head returns the block-id (hash and number) of the L2 chain head
func (e *EngineDriver) L2Head() eth.BlockID {
	e.headLock.RLock()
	defer e.headLock.RUnlock()
	return e.l2Head
}

func (e *EngineDriver) Head() (l1Head eth.BlockID, l2Head eth.BlockID) {
	e.headLock.RLock()
	defer e.headLock.RUnlock()
	return e.l1Head, e.l2Head
}

func (e *EngineDriver) UpdateHead(l1Head eth.BlockID, l2Head eth.BlockID) {
	e.headLock.Lock()
	defer e.headLock.Unlock()
	e.l1Head = l1Head
	e.l2Head = l2Head
}

func (e *EngineDriver) RequestHeadUpdate(ctx context.Context) error {
	e.headLock.Lock()
	defer e.headLock.Unlock()
	refL1, refL2, _, err := e.SyncRef.RefByL2Num(ctx, nil, &e.Genesis)
	if err != nil {
		return err
	}
	e.l1Head = refL1
	e.l2Head = refL2
	return nil
}

func (e *EngineDriver) Drive(ctx context.Context, dl Downloader, l1Heads <-chan eth.HeadSignal) ethereum.Subscription {
	e.driveLock.Lock()
	defer e.driveLock.Unlock()
	if e.driveSub != nil {
		return e.driveSub
	}
	e.driveSub = event.NewSubscription(func(quit <-chan struct{}) error {
		// keep making many sync steps if we can make sync progress
		hot := time.Millisecond * 30
		// check on sync regularly, but prioritize sync triggers with head updates etc.
		cold := time.Second * 8
		// at least try every minute to sync, even if things are going well
		max := time.Minute

		syncTicker := time.NewTicker(cold)
		defer syncTicker.Stop()

		// backoff sync attempts if we are not making progress
		backoff := cold
		syncQuickly := func() {
			syncTicker.Reset(hot)
			backoff = cold
		}
		// exponential backoff, add 10% each step, up to max.
		syncBackoff := func() {
			backoff += backoff / 10
			if backoff > max {
				backoff = max
			}
			syncTicker.Reset(backoff)
		}

		l2HeadPollTicker := time.NewTicker(time.Second * 14)
		defer l2HeadPollTicker.Stop()

		onL2Update := func() {
			// When we updated L2, we want to continue sync quickly
			syncQuickly()
			// And we want to slow down requesting the L2 engine for its head (we just changed it ourselves)
			// Request head if we don't successfully change it in the next 14 seconds.
			l2HeadPollTicker.Reset(time.Second * 14)
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-quit:
				return nil
			case <-l2HeadPollTicker.C:
				ctx, cancel := context.WithTimeout(ctx, time.Second*4)
				if err := e.RequestHeadUpdate(ctx); err != nil {
					e.Log.Error("failed to fetch L2 head info", "err", err)
				}
				cancel()
				continue
			case l1HeadSig := <-l1Heads:
				if e.l1Head == l1HeadSig.Self {
					e.Log.Debug("Received L1 head signal, already synced to it, ignoring event", "l1_head", e.l1Head)
					continue
				}
				if e.l1Head == l1HeadSig.Parent {
					// Simple extend, a linear life is easy
					if l2ID, err := DriverStep(ctx, e.Log, e.RPC, dl, l1HeadSig.Self, e.l2Head, e.l2Finalized.Hash); err != nil {
						e.Log.Error("Failed to extend L2 chain with new L1 block", "l1", l1HeadSig.Self, "l2", e.l2Head, "err", err)
						// Retry sync later
						e.l1Target = l1HeadSig.Self
						onL2Update()
						continue
					} else {
						e.UpdateHead(l1HeadSig.Self, l2ID)
						continue
					}
				}
				if e.l1Head.Number < l1HeadSig.Parent.Number {
					e.Log.Debug("Received new L1 head, engine is out of sync, cannot immediately process", "l1", l1HeadSig.Self, "l2", e.l2Head)
				} else {
					e.Log.Warn("Received a L1 reorg, syncing new alternative chain", "l1", l1HeadSig.Self, "l2", e.l2Head)
				}

				e.l1Target = l1HeadSig.Self
				syncQuickly()
				continue
			case <-syncTicker.C:
				// If already synced, or in case of failure, we slow down
				syncBackoff()
				if e.l1Head == e.l1Target {
					e.Log.Debug("Engine is fully synced", "l1_head", e.l1Head, "l2_head", e.l2Head)
					// TODO: even though we are fully synced, it may be worth attempting anyway,
					// in case the e.l1Head is not updating (failed/broken L1 head subscription)
					continue
				}
				e.Log.Debug("finding next sync step, engine syncing", "l2", e.l2Head, "l1", e.l1Head)
				nextRefL1, refL2, err := FindSyncStart(ctx, e.SyncRef, &e.Genesis)
				if err != nil {
					e.Log.Error("Failed to find sync starting point", "err", err)
					continue
				}
				if nextRefL1 == e.l1Head {
					e.Log.Debug("Engine is already synced, aborting sync", "l1_head", e.l1Head, "l2_head", e.l2Head)
					continue
				}
				if l2ID, err := DriverStep(ctx, e.Log, e.RPC, dl, nextRefL1, refL2, e.l2Finalized.Hash); err != nil {
					e.Log.Error("Failed to sync L2 chain with new L1 block", "l1", nextRefL1, "onto_l2", refL2, "err", err)
					continue
				} else {
					e.UpdateHead(nextRefL1, l2ID) // l2ID is derived from the nextRefL1
				}
				// Successfully stepped toward target. Continue quickly if we are not there yet
				if e.l1Head != e.l1Target {
					onL2Update()
				}
			}
		}
	})
	return e.driveSub
}

func (e *EngineDriver) Close() {
	e.RPC.Close()
	e.driveSub.Unsubscribe()
}
