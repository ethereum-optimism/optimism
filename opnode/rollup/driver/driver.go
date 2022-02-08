package driver

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l1"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	rollupSync "github.com/ethereum-optimism/optimistic-specs/opnode/rollup/sync"

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
	SyncRef rollupSync.SyncReference

	// The current driving force, to shutdown before closing the engine.
	driveSub ethereum.Subscription
	// There may only be 1 driver at a time
	driveLock sync.Mutex

	EngineDriverState
}

// ENTRYPOINT
func (e *EngineDriver) Drive(ctx context.Context, l1Heads <-chan eth.HeadSignal) ethereum.Subscription {
	e.driveLock.Lock()
	defer e.driveLock.Unlock()
	if e.driveSub != nil {
		return e.driveSub
	}

	e.driveSub = event.NewSubscription(newDriverLoop(ctx, &e.EngineDriverState, e.Log, l1Heads, e))
	return e.driveSub
}

func (e *EngineDriver) requestEngineHead(ctx context.Context) (refL1 eth.BlockID, refL2 eth.BlockID, err error) {
	refL1, refL2, _, err = e.SyncRef.RefByL2Num(ctx, nil, &e.Genesis)
	return
}

func (e *EngineDriver) findSyncStart(ctx context.Context) (nextRefL1 eth.BlockID, refL2 eth.BlockID, err error) {
	return rollupSync.FindSyncStart(ctx, e.SyncRef, &e.Genesis)
}

func (e *EngineDriver) driverStep(ctx context.Context, nextRefL1 eth.BlockID, refL2 eth.BlockID, finalized eth.BlockID) (l2ID eth.BlockID, err error) {
	return DriverStep(ctx, e.Log, e.RPC, e.DL, nextRefL1, refL2, finalized.Hash)
}

func (e *EngineDriver) Close() {
	e.driveSub.Unsubscribe()
}

func (e *DriverV2) requestEngineHead(ctx context.Context) (refL1 eth.BlockID, refL2 eth.BlockID, err error) {
	refL1, refL2, _, err = e.syncRef.RefByL2Num(ctx, nil, &e.genesis)
	return
}

func (e *DriverV2) findSyncStart(ctx context.Context) (nextRefL1 eth.BlockID, refL2 eth.BlockID, err error) {
	return rollupSync.FindSyncStart(ctx, e.syncRef, &e.genesis)
}

func (e *DriverV2) driverStep(ctx context.Context, nextRefL1 eth.BlockID, refL2 eth.BlockID, finalized eth.BlockID) (l2ID eth.BlockID, err error) {
	return DriverStep(ctx, e.log, e.rpc, e.dl, nextRefL1, refL2, finalized.Hash)
}

type DriverV2 struct {
	log     log.Logger
	rpc     DriverAPI
	syncRef rollupSync.SyncReference
	dl      Downloader
	l1Heads <-chan eth.HeadSignal
	done    chan chan error
	genesis rollup.Genesis
	EngineDriverState
}

func NewDriver(l2 DriverAPI, l1 l1.Source, log log.Logger, genesis rollup.Genesis) *DriverV2 {
	return &DriverV2{
		log:               log,
		rpc:               l2,
		syncRef:           rollupSync.SyncSource{L1: l1, L2: l2},
		dl:                l1,
		done:              make(chan chan error),
		genesis:           genesis,
		EngineDriverState: EngineDriverState{Genesis: genesis},
	}
}

func (d *DriverV2) Start(ctx context.Context, l1Heads <-chan eth.HeadSignal) error {
	d.l1Heads = l1Heads
	go d.loop(ctx)
	return nil
}
func (d *DriverV2) Close() error {
	ec := make(chan error)
	d.done <- ec
	err := <-ec
	return err
}

func (d *DriverV2) loop(ctx context.Context) error {
	backoff := cold
	syncTicker := time.NewTicker(cold)
	l2HeadPoll := time.NewTicker(time.Second * 14)

	// exponential backoff, add 10% each step, up to max.
	syncBackoff := func() {
		backoff += backoff / 10
		if backoff > max {
			backoff = max
		}
		syncTicker.Reset(backoff)
	}
	syncQuickly := func() {
		syncTicker.Reset(hot)
		backoff = cold
	}
	onL2Update := func() {
		// And we want to slow down requesting the L2 engine for its head (we just changed it ourselves)
		// Request head if we don't successfully change it in the next 14 seconds.
		l2HeadPoll.Reset(time.Second * 14)
	}
	defer syncTicker.Stop()
	defer l2HeadPoll.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.done:
			return nil
		case <-l2HeadPoll.C:
			ctx, cancel := context.WithTimeout(ctx, time.Second*4)
			if d.RequestUpdate(ctx, d.log, d) {
				onL2Update()
			}
			cancel()
			continue
		case l1HeadSig := <-d.l1Heads:
			if d.NotifyL1Head(ctx, d.log, l1HeadSig, d) {
				syncQuickly()
			}
			continue
		case <-syncTicker.C:
			// If already synced, or in case of failure, we slow down
			syncBackoff()
			if d.RequestSync(ctx, d.log, d) {
				// Successfully stepped toward target. Continue quickly if we are not there yet
				syncQuickly()
				onL2Update()
			}
		}
	}

}
