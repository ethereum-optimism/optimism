package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type recoverySource interface {
	RecoveryBlock(context.Context, uint64, eth.BlockID) (eth.L2BlockRef, error)
}

// followRecovery supplies canonical deposit-only attributes to the ordinary
// attributes handler. Only the operator executes them: projection block hashes
// are never used as private engine forkchoice hashes.
// All methods run on the driver's event loop. RPC deadlines must not escape
// through emitted events: those events are consumed after the method returns.
type followRecovery struct {
	source     recoverySource
	l2         L2Chain
	builder    derive.AttributesBuilder
	engine     *engine.EngineController
	pause      func(bool)
	emitter    event.Emitter
	status     *sources.FollowStatus
	enabled    bool
	mapped     eth.L2BlockRef
	projection eth.L2BlockRef
	inflight   eth.L2BlockRef
}

func (f *followRecovery) AttachEmitter(em event.Emitter) { f.emitter = em }

func (f *followRecovery) update(ctx context.Context, status *sources.FollowStatus) (err error) {
	defer func() {
		if err != nil {
			f.pause(true)
		}
	}()
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	plan := status.Recovery
	if plan == nil {
		// Ordinary follow endpoints have no replacement protocol.
		if !f.enabled {
			f.engine.FollowSource(status.SafeL2, status.LocalSafeL2, status.FinalizedL2)
			return nil
		}
		f.pause(true)
		return fmt.Errorf("recovery-capable follow source omitted its recovery snapshot")
	}
	f.enabled = true
	if plan.Anchor != status.LocalSafeL2 || plan.Target.Number < plan.Anchor.Number || plan.Safe.Number > plan.Target.Number || plan.Finalized.Number > plan.Safe.Number || status.CurrentL1 == (eth.L1BlockRef{}) {
		f.pause(true)
		return fmt.Errorf("inconsistent private recovery snapshot")
	}
	if f.projection.Number > plan.Target.Number || f.inflight.Number > plan.Target.Number {
		f.pause(true)
		f.status, f.inflight = nil, eth.L2BlockRef{}
	}
	prefixChanged := f.status != nil && !sameRecoveryPrefix(f.status.Recovery.Prefix, plan.Prefix)
	if f.status == nil || f.status.Recovery.Anchor != plan.Anchor || prefixChanged {
		anchor := plan.Anchor
		// A locally finalized replacement remains an authenticated private anchor
		// across restarts, even if no later operator claim has been published.
		finalized := f.engine.FinalizedHead()
		if finalized.Number > anchor.Number {
			anchor = finalized
		}
		if plan.Prefix != nil && plan.Prefix.Last.Number > anchor.Number {
			var err error
			anchor, err = f.prefixAnchor(rpcCtx, anchor, plan.Prefix)
			if err != nil {
				f.pause(true)
				return err
			}
		}
		if anchor.Number > plan.Target.Number {
			return fmt.Errorf("projection recovery frontier is behind private finality")
		}
		local, lookupErr := f.l2.L2BlockRefByNumber(rpcCtx, anchor.Number)
		canonical := lookupErr == nil && local == anchor
		if !canonical {
			f.pause(true)
			available, err := f.l2.L2BlockRefByHash(rpcCtx, anchor.Hash)
			if err != nil {
				return fmt.Errorf("reading private recovery anchor: %w", err)
			}
			if available != anchor {
				return fmt.Errorf("private recovery anchor does not match its commitment")
			}
		}
		// Authenticate the safety labels on the selected private branch before
		// changing forkchoice. A known noncanonical commitment is recoverable.
		privateAt := func(number uint64) (eth.L2BlockRef, error) {
			if canonical {
				return f.l2.L2BlockRefByNumber(rpcCtx, number)
			}
			return f.ancestorAt(rpcCtx, anchor, number)
		}
		retained, err := privateAt(finalized.Number)
		if err != nil {
			return err
		}
		if retained != finalized {
			return fmt.Errorf("private recovery anchor contradicts finalized ancestry")
		}
		if status.FinalizedL2.Number > finalized.Number {
			finalized = status.FinalizedL2
		}
		crossSafe := status.SafeL2
		if crossSafe.Number < finalized.Number {
			crossSafe = finalized
		}
		for _, expected := range []eth.L2BlockRef{finalized, crossSafe} {
			actual, err := privateAt(expected.Number)
			if err != nil {
				return err
			}
			if actual != expected {
				return fmt.Errorf("private recovery safety label is not on the authenticated branch")
			}
		}

		reset := !canonical || f.status == nil ||
			prefixChanged ||
			plan.Prefix != nil ||
			anchor.Number < f.mapped.Number ||
			(anchor.Number == f.mapped.Number && anchor.Hash != f.mapped.Hash) ||
			f.inflight != (eth.L2BlockRef{})
		if reset {
			f.pause(true)
			unsafe := anchor
			if canonical && f.mapped == (eth.L2BlockRef{}) && plan.Prefix == nil {
				// Bootstrap the pending-safe cursor, retaining the existing unsafe
				// chain for the normal attributes handler to consolidate or replace.
				unsafe = f.engine.UnsafeL2Head()
			}
			f.engine.ForceReset(ctx, unsafe, anchor, crossSafe, finalized)
		} else {
			// A new claim usually confirms an existing private ancestor. Ordinary
			// advancement must not interrupt an unrelated sequencer build.
			f.engine.FollowSource(crossSafe, anchor, finalized)
			f.engine.TryUpdatePendingSafe(ctx, anchor, true, status.CurrentL1)
		}
		f.mapped, f.projection, f.inflight = anchor, eth.L2BlockRef{}, eth.L2BlockRef{}
	}
	// An already executed suffix must still belong to this canonical snapshot.
	if f.projection != (eth.L2BlockRef{}) {
		ref, err := f.source.RecoveryBlock(rpcCtx, f.projection.Number, plan.Target.ID())
		if err != nil {
			f.pause(true)
			return err
		}
		if ref != f.projection {
			f.pause(true)
			f.status, f.inflight = nil, eth.L2BlockRef{}
			return fmt.Errorf("canonical projection replacement changed; re-anchoring")
		}
	}
	f.status = status
	if err := f.followHeads(ctx); err != nil {
		f.pause(true)
		return err
	}
	if f.mapped.Number >= plan.Target.Number {
		f.pause(!f.canResume())
		return nil
	}
	f.pause(true)
	f.engine.RequestPendingSafeUpdate(ctx)
	return nil
}

func sameRecoveryPrefix(a, b *sources.FollowRecoveryPrefix) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// ancestorAt only follows hash-linked private headers. It never changes forkchoice.
func (f *followRecovery) ancestorAt(ctx context.Context, ref eth.L2BlockRef, number uint64) (eth.L2BlockRef, error) {
	if number > ref.Number {
		return eth.L2BlockRef{}, fmt.Errorf("private safety label is ahead of the recovery anchor")
	}
	for ref.Number > number {
		parent, err := f.l2.L2BlockRefByHash(ctx, ref.ParentHash)
		if err != nil {
			return eth.L2BlockRef{}, err
		}
		if parent.Hash != ref.ParentHash || parent.Number+1 != ref.Number {
			return eth.L2BlockRef{}, fmt.Errorf("private ancestry is inconsistent")
		}
		ref = parent
	}
	return ref, nil
}

func (f *followRecovery) prefixAnchor(ctx context.Context, base eth.L2BlockRef, prefix *sources.FollowRecoveryPrefix) (eth.L2BlockRef, error) {
	if prefix.Last.Number < base.Number || prefix.Last.Number >= prefix.Terminal.Number {
		return eth.L2BlockRef{}, fmt.Errorf("invalid surviving prefix bounds")
	}
	ref, err := f.l2.L2BlockRefByHash(ctx, prefix.Terminal.Hash)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if ref.ID() != prefix.Terminal || ref.ParentHash != prefix.TerminalParent {
		return eth.L2BlockRef{}, fmt.Errorf("private terminal does not match the accepted claim")
	}
	var anchor eth.L2BlockRef
	for ref.Number > base.Number {
		if ref.Number == prefix.Last.Number {
			anchor = ref
		}
		parent, err := f.l2.L2BlockRefByHash(ctx, ref.ParentHash)
		if err != nil {
			return eth.L2BlockRef{}, err
		}
		if parent.Hash != ref.ParentHash || parent.Number+1 != ref.Number {
			return eth.L2BlockRef{}, fmt.Errorf("private prefix ancestry is inconsistent")
		}
		ref = parent
	}
	if ref != base {
		return eth.L2BlockRef{}, fmt.Errorf("private prefix does not descend from the retained checkpoint")
	}
	if prefix.Last.Number == base.Number {
		anchor = base
	}
	if anchor.Time != prefix.Last.Time || anchor.L1Origin != prefix.Last.L1Origin || anchor.SequenceNumber != prefix.Last.SequenceNumber {
		return eth.L2BlockRef{}, fmt.Errorf("private prefix disagrees with surviving projection inputs")
	}
	return anchor, nil
}

func (f *followRecovery) OnEvent(ctx context.Context, ev event.Event) bool {
	if !f.enabled {
		return false
	}
	switch x := ev.(type) {
	case derive.PipelineStepEvent:
		if err := f.next(ctx, x.PendingSafe); err != nil {
			f.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
		}
	case engine.LocalSafeUpdateEvent:
		if f.status == nil || f.inflight == (eth.L2BlockRef{}) || x.Ref.Number != f.inflight.Number || x.Ref.ParentHash != f.mapped.Hash ||
			x.Ref.Number > f.status.Recovery.Target.Number || x.Ref.Time != f.inflight.Time ||
			x.Ref.L1Origin != f.inflight.L1Origin || x.Ref.SequenceNumber != f.inflight.SequenceNumber {
			return true
		}
		f.mapped, f.projection, f.inflight = x.Ref, f.inflight, eth.L2BlockRef{}
		if err := f.followHeads(ctx); err != nil {
			f.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
			return true
		}
		if f.mapped.Number == f.status.Recovery.Target.Number {
			f.pause(!f.canResume())
		}
	case rollup.EngineTemporaryErrorEvent:
		// The next follow poll requests pending-safe again, preserving any
		// in-flight attributes for the ordinary handler's retry path.
	case rollup.ResetEvent, engine.InvalidPayloadAttributesEvent, engine.PayloadSealInvalidEvent:
		f.pause(true)
		f.status, f.inflight = nil, eth.L2BlockRef{}
	case derive.ConfirmReceivedAttributesEvent, derive.ConfirmPipelineResetEvent:
	default:
		return false
	}
	return true
}

func (f *followRecovery) canResume() bool {
	plan := f.status.Recovery
	// The surviving carrier still reserves its old range in the registry. Wait
	// for actual canonical replacements through that range before publishing a
	// new claim; the reservation itself is never evidence of empty execution.
	return f.mapped.Number >= plan.Target.Number &&
		(plan.Prefix == nil || f.mapped.Number >= plan.Prefix.Terminal.Number)
}

// Map each projection safety frontier through the privately executed ancestry.
// Reading the private canonical chain is valid only up to the authenticated
// checkpoint plus the suffix this adapter has actually reconciled.
func (f *followRecovery) followHeads(ctx context.Context) error {
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	plan := f.status.Recovery
	cross, err := f.l2.L2BlockRefByNumber(rpcCtx, min(plan.Safe.Number, f.mapped.Number))
	if err != nil {
		return err
	}
	finalized, err := f.l2.L2BlockRefByNumber(rpcCtx, min(plan.Finalized.Number, cross.Number))
	if err != nil {
		return err
	}
	if finalized.Number < f.engine.FinalizedHead().Number ||
		(finalized.Number == f.engine.FinalizedHead().Number && finalized.Hash != f.engine.FinalizedHead().Hash) {
		return fmt.Errorf("projection safety snapshot contradicts private finality")
	}
	f.engine.FollowSource(cross, f.mapped, finalized)
	f.engine.RequestForkchoiceUpdate(ctx)
	return nil
}

func (f *followRecovery) next(ctx context.Context, parent eth.L2BlockRef) error {
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if f.status == nil || f.inflight != (eth.L2BlockRef{}) {
		return nil
	}
	if parent != f.mapped {
		return fmt.Errorf("private pending-safe does not match the recovery cursor")
	}
	if parent.Number >= f.status.Recovery.Target.Number {
		return nil
	}
	ref, err := f.source.RecoveryBlock(rpcCtx, parent.Number+1, f.status.Recovery.Target.ID())
	if err != nil {
		return err
	}
	if ref.Number != parent.Number+1 {
		return fmt.Errorf("recovery source returned an unexpected height")
	}
	attrs, err := f.builder.PreparePayloadAttributes(rpcCtx, parent, ref.L1Origin)
	if err != nil {
		return err
	}
	seq := uint64(0)
	if ref.L1Origin == parent.L1Origin {
		seq = parent.SequenceNumber + 1
	}
	if uint64(attrs.Timestamp) != ref.Time || ref.SequenceNumber != seq || !attrs.NoTxPool || !attrs.IsDepositsOnly() {
		return fmt.Errorf("private replacement attributes disagree with canonical projection schedule")
	}
	f.inflight = ref
	f.emitter.Emit(ctx, derive.DerivedAttributesEvent{Attributes: &derive.AttributesWithParent{
		Attributes: attrs, Parent: parent, Concluding: true,
		// Use the later derivation frontier, never the block's old L1 origin:
		// window-expiry fallback was not canonical at that origin yet.
		DerivedFrom: f.status.CurrentL1,
	}})
	return nil
}
