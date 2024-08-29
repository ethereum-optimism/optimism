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

type InteropDeriver struct {
	log log.Logger
	cfg *rollup.Config

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

// TODO: trigger CrossUpdateRequestEvent
//  if cross-L2 updates have not shown up for a while. Or maybe trigger by op-supervisor event?

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
		ctx, cancel := context.WithTimeout(d.driverCtx, checkBlockTimeout)
		defer cancel()
		candidate, err := d.l2.L2BlockRefByNumber(ctx, x.CrossUnsafe.Number+1)
		if err != nil {
			d.log.Warn("Failed to fetch next cross-unsafe candidate", "err", err)
			break
		}
		// pre-interop the engine itself handles promotion to cross-unsafe
		if !d.cfg.IsInterop(candidate.Time) {
			return false
		}
		blockSafety, err := d.backend.CheckBlock(ctx, d.chainID, candidate.Hash, candidate.Number)
		if err != nil {
			d.log.Warn("Failed to check interop safety of unsafe block", "err", err)
			break
		}
		if blockSafety != types.Unsafe {
			d.emitter.Emit(engine.PromoteCrossUnsafeEvent{Ref: candidate})
		}
		// TODO: check if block safety == invalid: trigger chain halt (or reorg)
	case engine.LocalSafeUpdateEvent:
		d.derivedFrom[x.Ref.Hash] = x.DerivedFrom
		d.emitter.Emit(engine.RequestCrossSafeEvent{})
	case engine.CrossSafeUpdateEvent:
		if x.CrossSafe.Number >= x.LocalSafe.Number {
			break // nothing left to promote
		}
		ctx, cancel := context.WithTimeout(d.driverCtx, checkBlockTimeout)
		defer cancel()
		candidate, err := d.l2.L2BlockRefByNumber(ctx, x.CrossSafe.Number+1)
		if err != nil {
			d.log.Warn("Failed to fetch next cross-safe candidate", "err", err)
			break
		}
		// pre-interop the engine itself handles promotion to cross-safe
		if !d.cfg.IsInterop(candidate.Time) {
			return false
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
			// TODO: once we have interop reorg support, we need to clean stale blocks also.
			delete(d.derivedFrom, candidate.Hash)
			d.emitter.Emit(engine.PromoteSafeEvent{
				Ref:         candidate,
				DerivedFrom: derivedFrom,
			})
		case types.Finalized:
			// TODO: once we have interop reorg support, we need to clean stale blocks also.
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
