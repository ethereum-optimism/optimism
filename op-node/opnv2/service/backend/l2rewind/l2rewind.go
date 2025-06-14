package l2rewind

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type DB interface {
	LocalUnsafe(chainID eth.ChainID) (types.BlockSeal, error)
	LocalSafe(chainID eth.ChainID) (types.DerivedBlockSealPair, error)

	RewindLogs(chainID eth.ChainID, newHead types.BlockSeal) error

	FindSealedBlock(chainID eth.ChainID, num uint64) (types.BlockSeal, error)
	IsLocalSafe(chainID eth.ChainID, block eth.BlockID) error
}

type L2Rewinder struct {
	log log.Logger

	chainID eth.ChainID

	db DB

	emitter event.Emitter
}

func NewL2Rewinder(logger log.Logger, chainID eth.ChainID, db DB) *L2Rewinder {
	return &L2Rewinder{
		log:     logger,
		chainID: chainID,
		db:      db,
	}
}

func (r *L2Rewinder) AttachEmitter(em event.Emitter) {
	r.emitter = em
}

func (r *L2Rewinder) OnEvent(ctx context.Context, ev event.Event) bool {
	// TODO filter event context by ChainID

	// TODO on L2 rewind check, run the rewind, and emit an event when done
	return false
}

type L2RewindCheckEvent struct {
	ChainID eth.ChainID
}

func (ev L2RewindCheckEvent) String() string {
	return "l2-rewind-check"
}

type L2RewindCheckCompletedEvent struct {
	ChainID eth.ChainID

	// zero if we rewind to a point older than DB start
	LocalUnsafe types.BlockSeal
}

func (ev L2RewindCheckCompletedEvent) String() string {
	return "l2-rewind-check-completed"
}

// TODO: L2 log db rewinds
//
// A Lot like the L1 rewinder, except we are comparing the log-db against the local-safe-db
// (use the most canonical version of each block, in case of invalidations).
//
// The old "rewinder" has the following problems:
// - does both L1 and L2
// - does no binary search (reasonable for small finalized-head range, but not always the case)
// - considers first block as "finalized"
// - does not rewind out the first block in the DB, even if needed.
//    (it will be unable to get first-1, and error, and not reorg)
//
// What it does right:
// - consider finality, to reduce search range
// - use finality based on the local DB, not the remote chain,
//   since the local DB may be out of sync in terms of block-number, but on the wrong chain.
//
// We need a single binary-search that fixes this.
//
// We do this per L2, so the chains can rewind in parallel.
