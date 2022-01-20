package l2

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"

	"github.com/ethereum/go-ethereum/log"
)

type Driver interface {
	requestEngineHead(ctx context.Context) (refL1 eth.BlockID, refL2 eth.BlockID, err error)
	findSyncStart(ctx context.Context) (nextRefL1 eth.BlockID, refL2 eth.BlockID, err error)
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
