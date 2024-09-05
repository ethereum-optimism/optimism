package interop

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/event"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const checkBlockTimeout = time.Second * 10

type InteropBackend interface {
	CheckBlock(ctx context.Context,
		chainID types.ChainID, blockHash common.Hash, blockNumber uint64) (types.SafetyLevel, error)
}

type L2Source interface {
	L2BlockRefByNumber(context.Context, uint64) (eth.L2BlockRef, error)
}

// InteropDeriver watches for update events (either real changes to block safety,
// or updates published upon request), checks if there is some local data to cross-verify,
// and then checks with the interop-backend, to try to promote to cross-verified safety.
type InteropDeriver struct {
	log log.Logger
	cfg *rollup.Config

	// we cache the chainID,
	// to not continuously convert from the type in the rollup-config to this type.
	chainID types.ChainID

	driverCtx context.Context

	// L2 blockhash -> derived from L1 block ref.
	// Added to when a block is local-safe.
	// Removed from when it is promoted to cross-safe.
	derivedFrom map[common.Hash]eth.L1BlockRef

	backend InteropBackend
	l2      L2Source

	emitter event.Emitter

	mu sync.Mutex
}

var _ event.Deriver = (*InteropDeriver)(nil)
var _ event.AttachEmitter = (*InteropDeriver)(nil)

func NewInteropDeriver(log log.Logger, cfg *rollup.Config,
	driverCtx context.Context, backend InteropBackend, l2 L2Source) *InteropDeriver {
	return &InteropDeriver{
		log:         log,
		cfg:         cfg,
		chainID:     types.ChainIDFromBig(cfg.L2ChainID),
		driverCtx:   driverCtx,
		derivedFrom: make(map[common.Hash]eth.L1BlockRef),
		backend:     backend,
		l2:          l2,
	}
}

func (d *InteropDeriver) AttachEmitter(em event.Emitter) {
	d.emitter = em
}

func (d *InteropDeriver) OnEvent(ev event.Event) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch x := ev.(type) {
	case engine.UnsafeUpdateEvent:
		d.emitter.Emit(engine.RequestCrossUnsafeEvent{})
	case engine.CrossUnsafeUpdateEvent:
		if x.CrossUnsafe.Number >= x.LocalUnsafe.Number {
			break // nothing left to promote
		}
		// Pre-interop the engine itself handles promotion to cross-unsafe.
		// Check if the next block (still unsafe) can be promoted to cross-unsafe.
		if !d.cfg.IsInterop(d.cfg.TimestampForBlock(x.CrossUnsafe.Number + 1)) {
			return false
		}
		ctx, cancel := context.WithTimeout(d.driverCtx, checkBlockTimeout)
		defer cancel()
		candidate, err := d.l2.L2BlockRefByNumber(ctx, x.CrossUnsafe.Number+1)
		if err != nil {
			d.log.Warn("Failed to fetch next cross-unsafe candidate", "err", err)
			break
		}
		blockSafety, err := d.backend.CheckBlock(ctx, d.chainID, candidate.Hash, candidate.Number)
		if err != nil {
			d.log.Warn("Failed to check interop safety of unsafe block", "err", err)
			break
		}
		switch blockSafety {
		case types.CrossUnsafe, types.CrossSafe, types.CrossFinalized:
			// Hold off on promoting higher than cross-unsafe,
			// this will happen once we verify it to be local-safe first.
			d.emitter.Emit(engine.PromoteCrossUnsafeEvent{Ref: candidate})
		}
	case engine.LocalSafeUpdateEvent:
		d.derivedFrom[x.Ref.Hash] = x.DerivedFrom
		d.emitter.Emit(engine.RequestCrossSafeEvent{})
	case engine.CrossSafeUpdateEvent:
		if x.CrossSafe.Number >= x.LocalSafe.Number {
			break // nothing left to promote
		}
		// Pre-interop the engine itself handles promotion to cross-safe.
		// Check if the next block (not yet cross-safe) can be promoted to cross-safe.
		if !d.cfg.IsInterop(d.cfg.TimestampForBlock(x.CrossSafe.Number + 1)) {
			return false
		}
		ctx, cancel := context.WithTimeout(d.driverCtx, checkBlockTimeout)
		defer cancel()
		candidate, err := d.l2.L2BlockRefByNumber(ctx, x.CrossSafe.Number+1)
		if err != nil {
			d.log.Warn("Failed to fetch next cross-safe candidate", "err", err)
			break
		}
		blockSafety, err := d.backend.CheckBlock(ctx, d.chainID, candidate.Hash, candidate.Number)
		if err != nil {
			d.log.Warn("Failed to check interop safety of local-safe block", "err", err)
			break
		}
		derivedFrom, ok := d.derivedFrom[candidate.Hash]
		if !ok {
			break
		}
		switch blockSafety {
		case types.CrossSafe:
			// TODO(#11673): once we have interop reorg support, we need to clean stale blocks also.
			delete(d.derivedFrom, candidate.Hash)
			d.emitter.Emit(engine.PromoteSafeEvent{
				Ref:         candidate,
				DerivedFrom: derivedFrom,
			})
		case types.Finalized:
			// TODO(#11673): once we have interop reorg support, we need to clean stale blocks also.
			delete(d.derivedFrom, candidate.Hash)
			d.emitter.Emit(engine.PromoteSafeEvent{
				Ref:         candidate,
				DerivedFrom: derivedFrom,
			})
			d.emitter.Emit(engine.PromoteFinalizedEvent{
				Ref: candidate,
			})
		}
	// no reorg support yet; the safe L2 head will finalize eventually, no exceptions
	default:
		return false
	}
	return true
}
