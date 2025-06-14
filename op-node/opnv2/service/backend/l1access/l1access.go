package l1access

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/safemath"
)

type L1Source interface {
	L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error)
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)
}

const reqTimeout = time.Second * 10

type Config struct {
	ConfDepth uint64

	Subscribe bool

	// TODO poll interval (or via underlying RPC http polling?)
}

// L1Accessor is a read-only L1 accessor
type L1Accessor struct {
	ctx context.Context

	emitter event.Emitter

	log log.Logger

	finalitySub ethereum.Subscription
	latestSub   ethereum.Subscription

	// tip is the L1 chain tip. Used to block access to requests more recent than
	// the confirmation depth, and to detect reorgs.
	// Or zero if no block was seen yet.
	tip eth.BlockRef

	// confirmed is the last seen L1 block, with tip.Number-ConfDepth == confirmed.Number.
	// Or the finalized block, if finalizing recent blocks.
	// Or zero if no block is currently confirmed.
	confirmed eth.BlockRef

	// finalized is the last seen finalized L1 signal.
	// Or zero if no signal has been received yet.
	finalized eth.BlockRef

	cfg *Config

	client L1Source
}

var _ event.AttachEmitter = (*L1Accessor)(nil)
var _ event.Deriver = (*L1Accessor)(nil)

func NewL1Access(rootCtx context.Context, logger log.Logger,
	client L1Source, cfg *Config) *L1Accessor {
	return &L1Accessor{
		log:    logger,
		ctx:    rootCtx,
		client: client,
		cfg:    cfg,
	}
}

func (a *L1Accessor) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case LatestL1RequestEvent:
		a.onLatestL1Request(ctx, x)
	case ConfirmedL1RequestEvent:
		a.onConfirmedL1Request(ctx, x)
	case FinalizedL1RequestEvent:
		a.onFinalizedL1Request(ctx, x)
	case ByNumberL1RequestEvent:
		a.onByNumberL1Request(ctx, x)
	default:
		return false
	}
	return true
}

func (a *L1Accessor) AttachEmitter(em event.Emitter) {
	a.emitter = em
}

func (a *L1Accessor) SubscribeFinalityHandler() {
	// TODO: tests run faster finality, L1 has slow finality (every 6.4 minute, but not guaranteed),
	//  and "finalized" does not have an eth_subscribe.
	//  So customized polling, at a time just after the expected finality, would be ideal.
	// Note that gap-slots can make the finalized EL timestamp not align with expected finality interval.
	a.finalitySub = eth.PollBlockChanges(
		a.log,
		a.client,
		a.updateFinalized,
		eth.Finalized,
		3*time.Second,
		reqTimeout)
}

func (a *L1Accessor) UnsubscribeFinalityHandler() {
	if a.finalitySub != nil {
		a.finalitySub.Unsubscribe()
	}
}

func (a *L1Accessor) SubscribeLatestHandler() {
	// TODO: current options are all sub-optimal:
	//  - the (unapplied) PollingClient wrapper around the untyped RPC does not contribute to the L1 block-header cache.
	//  - the WatchHeadChanges will exit on fail, and needs to be wrapped with a retry-subscriber, and also does not contribute to the cache.
	//  - the PollBlockChanges does contribute to the cache, but is polling rather than subscribing.
	// So for now we go with polling, but we should make a subscriber, attached to EthClient (for L1/L2 reuse, and caching).

	a.latestSub = eth.PollBlockChanges(
		a.log,
		a.client,
		a.updateLatest,
		eth.Unsafe,
		3*time.Second,
		reqTimeout)
}

func (a *L1Accessor) UnsubscribeLatestHandler() {
	if a.latestSub != nil {
		a.latestSub.Unsubscribe()
	}
}

func (a *L1Accessor) SetConfDepth(depth uint64) {
	a.cfg.ConfDepth = depth
}

func (a *L1Accessor) onLatestL1Request(ctx context.Context, ev LatestL1RequestEvent) {
	if err := a.PullLatest(ctx); err != nil {
		a.emitTempErr(ctx, err)
	}
}

func (a *L1Accessor) onConfirmedL1Request(ctx context.Context, ev ConfirmedL1RequestEvent) {
	a.updateConfirmed(ctx)
}

func (a *L1Accessor) onFinalizedL1Request(ctx context.Context, ev FinalizedL1RequestEvent) {
	if err := a.PullFinalized(ctx); err != nil {
		a.emitTempErr(ctx, err)
	}
}

func (a *L1Accessor) emitTempErr(ctx context.Context, err error) {
	a.emitter.Emit(ctx, TemporaryL1AccessErrorEvent{
		LatestL1:    a.tip,
		ConfirmedL1: a.confirmed,
		FinalizedL1: a.finalized,
		Err:         err,
	})
}

func (a *L1Accessor) PullFinalized(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()
	ref, err := a.client.L1BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return fmt.Errorf("failed to pull finalized block ref: %w", err)
	}
	a.updateFinalized(ctx, ref)
	return nil
}

func (a *L1Accessor) PullLatest(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()
	ref, err := a.client.L1BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to pull latest block ref: %w", err)
	}
	a.updateLatest(ctx, ref)
	return nil
}

func (a *L1Accessor) updateFinalized(ctx context.Context, ref eth.L1BlockRef) {
	if ref.Number <= a.finalized.Number {
		a.log.Debug("Ignoring stale L1 finality signal", "stale", ref, "current", a.finalized)
		a.emitter.Emit(ctx, FinalizedL1UpdateEvent{FinalizedL1: a.finalized})
		return
	}
	a.finalized = ref
	if ref.Number > a.confirmed.Number {
		a.confirmed = ref
		a.emitter.Emit(ctx, ConfirmedL1UpdateEvent{ConfirmedL1: a.confirmed})
	}
	a.emitter.Emit(ctx, FinalizedL1UpdateEvent{FinalizedL1: ref})
}

func (a *L1Accessor) updateLatest(ctx context.Context, ref eth.L1BlockRef) {
	// Stop if the block is the same or older than the tip
	if ref.Hash == a.tip.Hash {
		a.log.Debug("Latest L1 block signal is a repeat", "ref", ref)
		a.emitter.Emit(ctx, LatestL1UpdateEvent{LatestL1: a.tip})
		return
	}
	if ref.Number < a.tip.Number {
		a.log.Warn("L1 block is older than the tip", "ref", ref, "tip", a.tip)
		a.emitter.Emit(ctx, LatestL1UpdateEvent{LatestL1: a.tip})
		return
	}
	if ref.Number == a.tip.Number && ref.Hash != a.tip.Hash {
		if ref.ParentHash == a.tip.ParentHash {
			a.log.Warn("Detected L1 reorg, single block reorg",
				"ref", ref, "tip", a.tip, "parent", a.tip.ParentID())
		} else {
			a.log.Warn("Detected L1 reorg, block conflicts with existing chain", "ref", ref, "tip", a.tip)
		}
	}
	if ref.Number+1 == a.tip.Number {
		if ref.ParentHash != a.tip.Hash {
			a.log.Warn("Detected L1 reorg, next block does not continue same chain", "ref", ref, "tip", a.tip, "parent", a.tip.ParentID())
		}
	}
	if ref.Number > a.tip.Number+1 {
		a.log.Warn("Missed a L1 block signal", "ref", ref, "tip", a.tip)
	}
	a.tip = ref
	a.log.Info("Registered new L1 block", "tip", a.tip)
	a.emitter.Emit(ctx, LatestL1UpdateEvent{LatestL1: a.tip})
	a.updateConfirmed(ctx)
}

func (a *L1Accessor) updateConfirmed(ctx context.Context) {
	confNum := safemath.SaturatingSub(a.tip.Number, a.cfg.ConfDepth)
	ctx, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()
	v, err := a.client.L1BlockRefByNumber(ctx, confNum)
	if err != nil {
		v = eth.BlockRef{}
		a.log.Error("Could not retrieve confirmed L1 block", "err", err, "num", confNum)
	}
	a.confirmed = v
	a.emitter.Emit(ctx, ConfirmedL1UpdateEvent{ConfirmedL1: a.confirmed})
}

func (a *L1Accessor) onByNumberL1Request(ctx context.Context, ev ByNumberL1RequestEvent) {
	// The controller may ask for some specific L1 block, e.g. for derivation traversal.
	// We provide it, while enforcing conf-depth.
	ref, err := a.L1BlockRefByNumber(ctx, ev.Num)
	if err != nil {
		a.emitTempErr(ctx, fmt.Errorf("failed to retrieve block by number %d: %w", ev.Num, err))
		return
	}
	a.emitter.Emit(ctx, RetrievedL1BlockEvent{
		Ref: ref,
	})
}

func (a *L1Accessor) isWithinConfDepth(num uint64) bool {
	return safemath.SaturatingAdd(num, a.cfg.ConfDepth) <= a.tip.Number
}

// L1BlockRefByNumber implements the L1 source interface, with confirmation-depth applied.
// Blocks that are too recent for the confirmation depth will appear like blocks that are not available yet.
func (a *L1Accessor) L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error) {
	// block access to requests more recent than the confirmation depth
	if !a.isWithinConfDepth(number) {
		return eth.L1BlockRef{}, ethereum.NotFound
	}
	return a.client.L1BlockRefByNumber(ctx, number)
}
