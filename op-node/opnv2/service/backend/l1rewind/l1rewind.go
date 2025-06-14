package l1rewind

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type L1Source interface {
	L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error)
}

type DB interface {
	Finalized(eth.ChainID) (types.DerivedBlockSealPair, error)

	LocalSafe(eth.ChainID) (types.DerivedBlockSealPair, error)
	CrossSafe(eth.ChainID) (types.DerivedBlockSealPair, error)

	LocalSafeDerivedAt(chainID eth.ChainID, source eth.BlockID) (types.BlockSeal, error)
	CrossSourceToLastDerived(chainID eth.ChainID, source eth.BlockID) (derived types.BlockSeal, err error)

	RewindLocalSafeSource(eth.ChainID, eth.BlockID) error
	RewindCrossSafeSource(eth.ChainID, eth.BlockID) error
}

type L1Rewinder struct {
	log log.Logger

	// The L2 chain that this rewinder will apply to
	chainID eth.ChainID

	db DB
	l1 L1Source

	emitter event.Emitter
}

func NewL1Rewinder(logger log.Logger, chainID eth.ChainID, db DB, l1 L1Source) *L1Rewinder {
	return &L1Rewinder{
		log:     logger,
		chainID: chainID,
		db:      db,
		l1:      l1,
	}
}

func (r *L1Rewinder) AttachEmitter(em event.Emitter) {
	r.emitter = em
}

func (r *L1Rewinder) OnEvent(ctx context.Context, ev event.Event) bool {
	// TODO filter event context by ChainID

	// TODO on L1 rewind check, run the rewind, and emit an event when done
	return false
}

type L1RewindCheckEvent struct {
	ChainID eth.ChainID
}

func (ev L1RewindCheckEvent) String() string {
	return "l1-rewind-check"
}

type L1RewindCheckCompletedEvent struct {
	ChainID eth.ChainID
	// zero if we rewind to a point older than DB start
	LocalSafe types.DerivedBlockSealPair
	// zero if we rewind to a point older than DB start
	CrossSafe types.DerivedBlockSealPair
}

func (ev L1RewindCheckCompletedEvent) String() string {
	return "l1-rewind-check-completed"
}

// TODO: L1 rewinds.
//
// The old "rewinder" has the following problems:
// - does both L1 and L2
// - does no binary search (reasonable for small finalized-head range, but not always the case)
// - considers first block as "finalized"
// - does not rewind out the first block in the DB, even if needed.
//
// What it does right:
// - consider finality, to reduce search range
// - use finality based on the local DB, not the remote chain,
//   since the local DB may be out of sync in terms of block-number, but on the wrong chain.
//
// We need a single binary-search that fixes this.
//
// This needs to happen for each L1 chain.
// This needs to happen for local-safe and cross-safe.
//
// When rewinding, we need the controller state of the affected DBs to be clearly expressed as busy,
// until the rewind succeeds/fails.
// So the fan-out for each chain may be better to apply with multiple L1Rewinder instances, so it can run in parallel.
