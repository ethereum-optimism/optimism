package l2

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"

	"github.com/ethereum/go-ethereum/log"
)

// Driver exposes the driver functionality that maintains the external execution-engine.
type Driver interface {
	// requestEngineHead retrieves the L2 head reference of the engine, as well as the L1 reference it was derived from.
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

type EngineDriver struct {
	Log log.Logger
	// API bindings to execution engine
	RPC     DriverAPI
	DL      Downloader
	SyncRef SyncReference

	// The current driving force, to shutdown before closing the engine.
	driveSub ethereum.Subscription
	// There may only be 1 driver at a time
	driveLock sync.Mutex

	EngineDriverState
}

func (e *EngineDriver) Drive(ctx context.Context, l1Heads <-chan eth.HeadSignal) ethereum.Subscription {
	e.driveLock.Lock()
	defer e.driveLock.Unlock()
	if e.driveSub != nil {
		return e.driveSub
	}

	e.driveSub = event.NewSubscription(NewDriverLoop(ctx, &e.EngineDriverState, e.Log, l1Heads, e))
	return e.driveSub
}

func (e *EngineDriver) requestEngineHead(ctx context.Context) (refL1 eth.BlockID, refL2 eth.BlockID, err error) {
	refL1, refL2, _, err = e.SyncRef.RefByL2Num(ctx, nil, &e.Genesis)
	return
}

func (e *EngineDriver) findSyncStart(ctx context.Context) (nextRefL1 eth.BlockID, refL2 eth.BlockID, err error) {
	return FindSyncStart(ctx, e.SyncRef, &e.Genesis)
}

func (e *EngineDriver) driverStep(ctx context.Context, nextRefL1 eth.BlockID, refL2 eth.BlockID, finalized eth.BlockID) (l2ID eth.BlockID, err error) {
	return DriverStep(ctx, e.Log, e.RPC, e.DL, nextRefL1, refL2, finalized.Hash)
}

func (e *EngineDriver) Close() {
	e.driveSub.Unsubscribe()
}
