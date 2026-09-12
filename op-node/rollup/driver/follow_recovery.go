package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"

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

type recoveryBuild struct {
	public eth.L2BlockRef
	parent common.Hash
}

// followRecovery resolves an attested private parent, then feeds the canonical
// replay interval into the ordinary attributes handler. EngineController owns
// private execution progress; this adapter only tracks its public correspondence.
// All methods run on the driver event loop. RPC deadlines never escape in events.
type followRecovery struct {
	source         recoverySource
	l2             L2Chain
	builder        derive.AttributesBuilder
	engine         *engine.EngineController
	pause          func(bool)
	emitter        event.Emitter
	enabled        bool
	status         *sources.FollowStatus
	applied        eth.L2BlockRef
	appliedPrivate eth.L2BlockRef
	build          *recoveryBuild
	journal        *recoveryJournal
}

func (f *followRecovery) AttachEmitter(em event.Emitter) { f.emitter = em }

func (f *followRecovery) update(ctx context.Context, status *sources.FollowStatus) (err error) {
	defer func() {
		if err != nil {
			f.pause(true)
		}
	}()
	plan := status.Recovery
	if plan == nil {
		if f.enabled {
			return fmt.Errorf("recovery source omitted its snapshot")
		}
		f.engine.FollowSource(status.SafeL2, status.LocalSafeL2, status.FinalizedL2)
		return nil
	}
	f.enabled = true
	if plan.Anchor != status.LocalSafeL2 || plan.Target.Number < plan.Anchor.Number ||
		plan.Safe.Number > plan.Target.Number || plan.Finalized.Number > plan.Safe.Number ||
		status.CurrentL1 == (eth.L1BlockRef{}) {
		return fmt.Errorf("inconsistent private recovery snapshot")
	}
	if p := plan.Prefix; p != nil && (p.Last.Number <= plan.Anchor.Number || p.Last.Number > plan.Target.Number || p.Last.Number > p.Parent.Number) {
		return fmt.Errorf("invalid surviving prefix bounds")
	}
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	changed := f.status == nil || f.status.Recovery.Anchor != plan.Anchor || !sameRecoveryPrefix(f.status.Recovery.Prefix, plan.Prefix)
	changed = changed || f.applied.Number > plan.Target.Number || f.build != nil && f.build.public.Number > plan.Target.Number
	if !changed && f.applied != (eth.L2BlockRef{}) {
		ref, err := f.source.RecoveryBlock(rpcCtx, f.applied.Number, plan.Target.ID())
		if err != nil {
			return err
		}
		changed = ref != f.applied
	}
	if !changed && f.build != nil {
		ref, err := f.source.RecoveryBlock(rpcCtx, f.build.public.Number, plan.Target.ID())
		if err != nil {
			return err
		}
		changed = ref != f.build.public
	}
	if changed {
		if err := f.adopt(ctx, rpcCtx, status); err != nil {
			return err
		}
	}
	f.status = status
	if err := f.saveProgress(); err != nil {
		return err
	}
	if err := f.followHeads(ctx); err != nil {
		return err
	}
	f.pause(!f.canResume(f.engine.LocalSafeHead().Number))
	f.engine.RequestPendingSafeUpdate(ctx)
	return nil
}

// Adopt the surviving branch before producing any replacement. Normal accepted
// checkpoint advancement uses FollowSource; only an actual rewind/branch change
// (or an interrupted replay) requires the existing engine reset path.
func (f *followRecovery) adopt(ctx, rpcCtx context.Context, status *sources.FollowStatus) error {
	plan := status.Recovery
	anchor, finalized := plan.Anchor, f.engine.FinalizedHead()
	if finalized.Number > anchor.Number {
		anchor = finalized
	}
	var err error
	if plan.Prefix != nil && plan.Prefix.Last.Number > anchor.Number {
		anchor, err = f.prefixAnchor(rpcCtx, anchor, plan.Prefix)
	} else {
		var found eth.L2BlockRef
		found, err = f.l2.L2BlockRefByHash(rpcCtx, anchor.Hash)
		if err == nil && found != anchor {
			err = fmt.Errorf("private recovery anchor does not match its commitment")
		}
	}
	if err != nil {
		return err
	}
	if anchor.Number > plan.Target.Number {
		return fmt.Errorf("projection recovery frontier is behind private finality")
	}
	progress, err := f.restoreProgress(rpcCtx, plan, anchor)
	if err != nil {
		return err
	}
	if progress != nil {
		anchor = progress.Private
	}
	if status.FinalizedL2.Number > finalized.Number {
		finalized = status.FinalizedL2
	}
	safe := status.SafeL2
	if safe.Number < finalized.Number {
		safe = finalized
	}
	canonical, lookupErr := f.l2.L2BlockRefByNumber(rpcCtx, anchor.Number)
	for _, expected := range []eth.L2BlockRef{f.engine.FinalizedHead(), finalized, safe} {
		var actual eth.L2BlockRef
		var err error
		if lookupErr == nil && canonical == anchor {
			actual, err = f.l2.L2BlockRefByNumber(rpcCtx, expected.Number)
		} else {
			actual, err = f.ancestorAt(rpcCtx, anchor, expected.Number)
		}
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("private recovery branch contradicts finalized ancestry or safety labels")
		}
	}
	if plan.Prefix != nil || f.journal != nil && f.journal.path != "" {
		// Clear obsolete replay evidence before rewinding. A crash must not make
		// the next startup reuse progress belonging to a revoked branch.
		if err := f.journal.commitProgress(finalized.Number, progress); err != nil {
			return fmt.Errorf("persisting private recovery ancestry: %w", err)
		}
	}
	// A lower checkpoint revokes the suffix even when this anchor is still on
	// our canonical private branch. The plan alone cannot distinguish a temporary
	// public safety retreat from a pending claim-carrier invalidation, so retaining
	// unsafe execution here would require additional evidence from the source.
	reset := progress == nil && (lookupErr != nil || canonical != anchor || anchor.Number < f.engine.LocalSafeHead().Number || plan.Prefix != nil || f.build != nil)
	if reset || progress != nil || f.engine.PendingSafeL2Head() == (eth.L2BlockRef{}) {
		f.pause(true)
		unsafe := anchor
		// A partially recovered prefix still reserves the original range. Any
		// unsafe suffix above that partial checkpoint has not been reconciled;
		// retaining it lets a batcher publish over the remaining replay interval.
		// An in-flight build likewise cannot justify preserving an unsafe suffix.
		partial := progress != nil && (f.build != nil || plan.Prefix != nil && anchor.Number <= plan.Prefix.Parent.Number)
		if !reset && !partial {
			unsafe = f.engine.UnsafeL2Head()
		}
		f.engine.ForceReset(ctx, unsafe, anchor, safe, finalized)
	} else {
		f.engine.FollowSource(safe, anchor, finalized)
		f.engine.TryUpdatePendingSafe(ctx, anchor, true, status.CurrentL1)
	}
	f.applied, f.appliedPrivate, f.build = eth.L2BlockRef{}, eth.L2BlockRef{}, nil
	if progress != nil {
		f.applied, f.appliedPrivate = progress.Public, progress.Private
	}
	return nil
}

// restoreProgress revalidates both sides of a durable replay checkpoint. A new
// invalidation, a public reorg, or a different private canonical branch still
// takes the normal rewind path; only previously executed recovery is retained.
func (f *followRecovery) restoreProgress(ctx context.Context, plan *sources.FollowRecoveryStatus, anchor eth.L2BlockRef) (*recoveryProgress, error) {
	if f.journal == nil || f.journal.path == "" {
		return nil, nil
	}
	if err := f.journal.load(); err != nil {
		return nil, err
	}
	p := f.journal.progress
	if p == nil || p.Anchor != plan.Anchor || !sameRecoveryPrefix(p.Prefix, plan.Prefix) ||
		p.Private.Number < anchor.Number || p.Public.Number > plan.Target.Number {
		return nil, nil
	}
	public, err := f.source.RecoveryBlock(ctx, p.Public.Number, plan.Target.ID())
	if err != nil {
		return nil, err
	}
	if public != p.Public {
		return nil, nil
	}
	private, err := f.l2.L2BlockRefByNumber(ctx, p.Private.Number)
	if err != nil {
		return nil, err
	}
	if private != p.Private || f.engine.UnsafeL2Head().Number < private.Number {
		return nil, nil
	}
	unsafe, err := f.l2.L2BlockRefByNumber(ctx, f.engine.UnsafeL2Head().Number)
	if err != nil {
		return nil, err
	}
	if unsafe != f.engine.UnsafeL2Head() {
		return nil, nil
	}
	return p, nil
}

func (f *followRecovery) saveProgress() error {
	if f.applied == (eth.L2BlockRef{}) || f.journal == nil || f.journal.path == "" {
		return nil
	}
	p := &recoveryProgress{Anchor: f.status.Recovery.Anchor, Public: f.applied, Private: f.appliedPrivate}
	if prefix := f.status.Recovery.Prefix; prefix != nil {
		copy := *prefix
		p.Prefix = &copy
	}
	if f.journal.progress.equal(p) {
		return nil
	}
	if err := f.journal.commitProgress(f.engine.FinalizedHead().Number, p); err != nil {
		return fmt.Errorf("persisting private recovery progress: %w", err)
	}
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
		parent, err := f.privateHeader(ctx, ref.ParentHash)
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
	if prefix.Last.Number < base.Number || prefix.Last.Number > prefix.Parent.Number {
		return eth.L2BlockRef{}, fmt.Errorf("invalid surviving prefix bounds")
	}
	ref, err := f.privateHeader(ctx, prefix.Parent.Hash)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if ref.ID() != prefix.Parent {
		return eth.L2BlockRef{}, fmt.Errorf("private terminal parent does not match the accepted claim")
	}
	anchor, err := f.ancestorAt(ctx, ref, prefix.Last.Number)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	retained, err := f.ancestorAt(ctx, anchor, base.Number)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if retained != base {
		return eth.L2BlockRef{}, fmt.Errorf("private prefix does not descend from the retained checkpoint")
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
		if f.status == nil || f.build == nil {
			return true
		}
		expected := f.build.public
		if x.Ref.ParentHash != f.build.parent || x.Ref.Number != expected.Number ||
			x.Ref.Time != expected.Time || x.Ref.L1Origin != expected.L1Origin ||
			x.Ref.SequenceNumber != expected.SequenceNumber || x.Ref.Number > f.status.Recovery.Target.Number {
			return true
		}
		// Execution may finish after the source changed branches, including
		// before its next status poll. Validate again before persisting progress,
		// promoting safety, or allowing the sequencer to resume.
		rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		canonical, err := f.source.RecoveryBlock(rpcCtx, expected.Number, f.status.Recovery.Target.ID())
		cancel()
		if err == nil && canonical != expected {
			err = fmt.Errorf("completed recovery block is no longer canonical")
		}
		if err != nil {
			f.pause(true)
			// Retain build until adoption so a valid earlier checkpoint cannot
			// preserve this obsolete unsafe block as an ordinary private suffix.
			f.status = nil
			f.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
			return true
		}
		f.applied, f.appliedPrivate, f.build = expected, x.Ref, nil
		if err := f.saveProgress(); err != nil {
			f.pause(true)
			f.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
			return true
		}
		if err := f.followHeads(ctx); err != nil {
			f.pause(true)
			f.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
			return true
		}
		f.pause(!f.canResume(f.engine.LocalSafeHead().Number))
	case rollup.ResetEvent, engine.InvalidPayloadAttributesEvent, engine.PayloadSealInvalidEvent:
		f.pause(true)
		f.status, f.build = nil, nil
	case rollup.EngineTemporaryErrorEvent, derive.ConfirmReceivedAttributesEvent, derive.ConfirmPipelineResetEvent:
	default:
		return false
	}
	return true
}

func (f *followRecovery) canResume(number uint64) bool {
	p := f.status.Recovery
	// A surviving carrier reserves the whole original range even if the rest
	// only becomes deposit-only after sequencing-window expiry.
	return number >= p.Target.Number && (p.Prefix == nil || number > p.Prefix.Parent.Number)
}

func (f *followRecovery) followHeads(ctx context.Context) error {
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	plan, local := f.status.Recovery, f.engine.LocalSafeHead()
	safe, err := f.l2.L2BlockRefByNumber(rpcCtx, min(plan.Safe.Number, local.Number))
	if err != nil {
		return err
	}
	finalized, err := f.l2.L2BlockRefByNumber(rpcCtx, min(plan.Finalized.Number, safe.Number))
	if err != nil {
		return err
	}
	previous := f.engine.FinalizedHead()
	if finalized.Number < previous.Number || finalized.Number == previous.Number && finalized.Hash != previous.Hash {
		return fmt.Errorf("projection snapshot contradicts private finality")
	}
	f.engine.FollowSource(safe, local, finalized)
	f.engine.RequestForkchoiceUpdate(ctx)
	return nil
}

func (f *followRecovery) next(ctx context.Context, parent eth.L2BlockRef) error {
	if f.status == nil || f.build != nil {
		return nil
	}
	if parent != f.engine.LocalSafeHead() {
		return fmt.Errorf("private pending-safe does not match local-safe")
	}
	plan := f.status.Recovery
	if parent.Number >= plan.Target.Number {
		return nil
	}
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ref, err := f.source.RecoveryBlock(rpcCtx, parent.Number+1, plan.Target.ID())
	if err != nil {
		return err
	}
	attrs, err := f.builder.PreparePayloadAttributes(rpcCtx, parent, ref.L1Origin)
	if err != nil {
		return err
	}
	seq := uint64(0)
	if ref.L1Origin == parent.L1Origin {
		seq = parent.SequenceNumber + 1
	}
	if ref.Number != parent.Number+1 || uint64(attrs.Timestamp) != ref.Time || ref.SequenceNumber != seq || !attrs.NoTxPool || !attrs.IsDepositsOnly() {
		return fmt.Errorf("private replacement disagrees with canonical projection schedule")
	}
	f.build = &recoveryBuild{public: ref, parent: parent.Hash}
	f.emitter.Emit(ctx, derive.DerivedAttributesEvent{Attributes: &derive.AttributesWithParent{
		Attributes: attrs, Parent: parent, Concluding: true,
		// Window-expiry inputs became canonical at this frontier, not their L1 origin.
		DerivedFrom: f.status.CurrentL1,
	}})
	return nil
}
