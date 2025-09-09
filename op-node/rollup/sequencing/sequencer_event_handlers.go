package sequencing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

func (d *Sequencer) OnEvent(ctx context.Context, ev event.Event) bool {
	d.l.Lock()
	defer d.l.Unlock()

	preTime := d.nextAction
	preOk := d.nextActionOK
	defer func() {
		if d.nextActionOK != preOk || d.nextAction != preTime {
			d.log.Debug("Sequencer action schedule changed",
				"time", d.nextAction, "wait", d.nextAction.Sub(d.timeNow()), "ok", d.nextActionOK, "event", ev)
		}
	}()

	switch x := ev.(type) {
	case engine.InteropInvalidateBlockEvent:
		d.emitter.Emit(ctx, engine.BuildStartEvent{Attributes: x.Attributes})
	case BuildStartedEvent:
		d.onBuildStarted(ctx, x)
	case BuildSealEvent:
		d.onBuildSeal(ctx, x)
	case BuildSealedEvent:
		d.onBuildSealed(ctx, x)
	case BuildCancelEvent:
		d.onBuildCancel(ctx, x)
	case BuildInvalidEvent:
		d.onBuildInvalid(ctx, x)
	case engine.InvalidPayloadAttributesEvent:
		d.onInvalidPayloadAttributes(x)
	case engine.PayloadSealInvalidEvent:
		d.onPayloadSealInvalid(x)
	case engine.PayloadSealExpiredErrorEvent:
		d.onPayloadSealExpiredError(x)
	case engine.PayloadInvalidEvent:
		d.onPayloadInvalid(x)
	case engine.PayloadSuccessEvent:
		d.onPayloadSuccess(x)
	case SequencerActionEvent:
		d.onSequencerAction(x)
	case rollup.EngineTemporaryErrorEvent:
		d.onEngineTemporaryError(x)
	case rollup.ResetEvent:
		d.onReset(x)
	case engine.EngineResetConfirmedEvent:
		d.onEngineResetConfirmedEvent(x)
	case engine.ForkchoiceUpdateEvent:
		d.onForkchoiceUpdate(x)
	default:
		return false
	}
	return true
}

func (d *Sequencer) onBuildStarted(ctx context.Context, x BuildStartedEvent) {
	// If a (pending) safe block, immediately seal the block
	if x.DerivedFrom != (eth.L1BlockRef{}) {
		d.emitter.Emit(ctx, BuildSealEvent{
			Info:         x.Info,
			BuildStarted: x.BuildStarted,
			Concluding:   x.Concluding,
			DerivedFrom:  x.DerivedFrom,
		})

		// If we are adding new blocks onto the tip of the chain, derived from L1,
		// then don't try to build on top of it immediately, as sequencer.
		d.log.Warn("Detected new block-building from L1 derivation, avoiding sequencing for now.",
			"build_job", x.Info.ID, "build_timestamp", x.Info.Timestamp,
			"parent", x.Parent, "derived_from", x.DerivedFrom)
		d.nextActionOK = false
		return
	}
	if d.latest.Onto != x.Parent {
		d.log.Warn("Canceling stale block-building job that was just started, as target to build onto has changed",
			"stale", x.Parent, "new", d.latest.Onto, "job_id", x.Info.ID, "job_timestamp", x.Info.Timestamp)
		d.emitter.Emit(d.ctx, BuildCancelEvent{
			Info:  x.Info,
			Force: true,
		})
		d.handleInvalid()
		return
	}
	// if not a derived block, then it is work of the sequencer
	d.log.Debug("Sequencer started building new block",
		"payloadID", x.Info.ID, "parent", x.Parent, "parent_time", x.Parent.Time)
	d.latest.Info = x.Info
	d.latest.Started = x.BuildStarted

	d.nextActionOK = d.active.Load()

	// schedule sealing
	now := d.timeNow()
	payloadTime := time.Unix(int64(x.Parent.Time+d.rollupCfg.BlockTime), 0)
	remainingTime := payloadTime.Sub(now)
	if remainingTime < sealingDuration {
		d.nextAction = now // if there's not enough time for sealing, don't wait.
	} else {
		// finish with margin of sealing duration before payloadTime
		d.nextAction = payloadTime.Add(-sealingDuration)
	}
}

func (eq *Sequencer) onBuildSeal(ctx context.Context, ev BuildSealEvent) {
	rpcCtx, cancel := context.WithTimeout(eq.ctx, buildSealTimeout)
	defer cancel()

	sealingStart := time.Now()
	envelope, err := eq.engine.GetPayload(rpcCtx, ev.Info)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) && eth.ErrorCode(rpcErr.ErrorCode()) == eth.UnknownPayload {
			eq.log.Warn("Cannot seal block, payload ID is unknown",
				"payloadID", ev.Info.ID, "payload_time", ev.Info.Timestamp,
				"started_time", ev.BuildStarted)
		}
		// Although the engine will very likely not be able to continue from here with the same building job,
		// we still call it "temporary", since the exact same payload-attributes have not been invalidated in-consensus.
		// So the user (attributes-handler or sequencer) should be able to re-attempt the exact
		// same attributes with a new block-building job from here to recover from this error.
		// We name it "expired", as this generally identifies a timeout, unknown job, or otherwise invalidated work.
		eq.emitter.Emit(ctx, engine.PayloadSealExpiredErrorEvent{
			Info:        ev.Info,
			Err:         fmt.Errorf("failed to seal execution payload (ID: %s): %w", ev.Info.ID, err),
			Concluding:  ev.Concluding,
			DerivedFrom: ev.DerivedFrom,
		})
		return
	}

	if err := sanityCheckPayload(envelope.ExecutionPayload); err != nil {
		eq.emitter.Emit(ctx, engine.PayloadSealInvalidEvent{
			Info: ev.Info,
			Err: fmt.Errorf("failed sanity-check of execution payload contents (ID: %s, blockhash: %s): %w",
				ev.Info.ID, envelope.ExecutionPayload.BlockHash, err),
			Concluding:  ev.Concluding,
			DerivedFrom: ev.DerivedFrom,
		})
		return
	}

	ref, err := derive.PayloadToBlockRef(eq.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		eq.emitter.Emit(ctx, engine.PayloadSealInvalidEvent{
			Info:        ev.Info,
			Err:         fmt.Errorf("failed to decode L2 block ref from payload: %w", err),
			Concluding:  ev.Concluding,
			DerivedFrom: ev.DerivedFrom,
		})
		return
	}

	now := time.Now()
	sealTime := now.Sub(sealingStart)
	buildTime := now.Sub(ev.BuildStarted)
	eq.metrics.RecordSequencerSealingTime(sealTime)
	eq.metrics.RecordSequencerBuildingDiffTime(buildTime - time.Duration(eq.rollupCfg.BlockTime)*time.Second)

	txnCount := len(envelope.ExecutionPayload.Transactions)
	depositCount, _ := lastDeposit(envelope.ExecutionPayload.Transactions)
	eq.metrics.CountSequencedTxsInBlock(txnCount, depositCount)

	eq.log.Debug("Built new L2 block", "l2_unsafe", ref, "l1_origin", ref.L1Origin,
		"txs", txnCount, "deposits", depositCount, "time", ref.Time, "seal_time", sealTime, "build_time", buildTime)

	eq.emitter.Emit(ctx, BuildSealedEvent{
		Concluding:   ev.Concluding,
		DerivedFrom:  ev.DerivedFrom,
		BuildStarted: ev.BuildStarted,
		Info:         ev.Info,
		Envelope:     envelope,
		Ref:          ref,
	})
}

func (eq *Sequencer) onBuildCancel(ctx context.Context, ev BuildCancelEvent) {
	rpcCtx, cancel := context.WithTimeout(eq.ctx, buildCancelTimeout)
	defer cancel()
	// the building job gets wrapped up as soon as the payload is retrieved, there's no explicit cancel in the Engine API
	eq.log.Warn("cancelling old block building job", "info", ev.Info)
	_, err := eq.engine.GetPayload(rpcCtx, ev.Info)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) && eth.ErrorCode(rpcErr.ErrorCode()) == eth.UnknownPayload {
			eq.log.Warn("tried cancelling unknown block building job", "info", ev.Info, "err", err)
			return // if unknown, then it did not need to be cancelled anymore.
		}
		eq.log.Error("failed to cancel block building job", "info", ev.Info, "err", err)
		if !ev.Force {
			eq.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{Err: err})
		}
	}
}

func (eq *Sequencer) onBuildInvalid(ctx context.Context, ev BuildInvalidEvent) {
	eq.log.Warn("could not process payload attributes", "err", ev.Err)

	// Deposit transaction execution errors are suppressed in the execution engine, but if the
	// block is somehow invalid, there is nothing we can do to recover & we should exit.
	if ev.Attributes.Attributes.IsDepositsOnly() {
		eq.log.Error("deposit only block was invalid", "parent", ev.Attributes.Parent, "err", ev.Err)
		eq.emitter.Emit(ctx, rollup.CriticalErrorEvent{
			Err: fmt.Errorf("failed to process block with only deposit transactions: %w", ev.Err),
		})
		return
	}

	if ev.Attributes.IsDerived() && eq.rollupCfg.IsHolocene(ev.Attributes.DerivedFrom.Time) {
		parent := ev.Attributes.Parent.ID()
		derivedFrom := ev.Attributes.DerivedFrom
		eq.log.Warn("Holocene active, requesting deposits-only attributes", "parent", parent, "derived_from", derivedFrom)
		// request deposits-only version
		eq.emitter.Emit(ctx, derive.DepositsOnlyPayloadAttributesRequestEvent{
			Parent:      parent,
			DerivedFrom: derivedFrom,
		})
		return
	}

	// Revert the pending safe head to the safe head.
	eq.eng.SetPendingSafeL2Head(eq.eng.SafeL2Head())
	// suppress the error b/c we want to retry with the next batch from the batch queue
	// If there is no valid batch the node will eventually force a deposit only block. If
	// the deposit only block fails, this will return the critical error above.

	// Try to restore to previous known unsafe chain.
	eq.eng.SetBackupUnsafeL2Head(eq.eng.BackupUnsafeL2Head(), true)

	// drop the payload without inserting it into the engine

	// Signal that we deemed the attributes as unfit
	eq.emitter.Emit(ctx, engine.InvalidPayloadAttributesEvent(ev))
}

func (d *Sequencer) onInvalidPayloadAttributes(x engine.InvalidPayloadAttributesEvent) {
	if x.Attributes.DerivedFrom != (eth.L1BlockRef{}) {
		return // not our payload, should be ignored.
	}
	d.log.Error("Cannot sequence invalid payload attributes",
		"attributes_parent", x.Attributes.Parent,
		"timestamp", x.Attributes.Attributes.Timestamp, "err", x.Err)

	d.handleInvalid()
}

func (d *Sequencer) onBuildSealed(ctx context.Context, x BuildSealedEvent) {
	// If a (pending) safe block, immediately process the block
	if x.DerivedFrom != (eth.L1BlockRef{}) {
		d.emitter.Emit(ctx, engine.PayloadProcessEvent{
			Concluding:   x.Concluding,
			DerivedFrom:  x.DerivedFrom,
			Envelope:     x.Envelope,
			Ref:          x.Ref,
			BuildStarted: x.BuildStarted,
		})
	}

	if d.latest.Info != x.Info {
		return // not our payload, should be ignored.
	}
	d.log.Info("Sequencer sealed block", "payloadID", x.Info.ID,
		"block", x.Envelope.ExecutionPayload.ID(),
		"parent", x.Envelope.ExecutionPayload.ParentID(),
		"txs", len(x.Envelope.ExecutionPayload.Transactions),
		"time", uint64(x.Envelope.ExecutionPayload.Timestamp))

	// generous timeout, the conductor is important
	ctx, cancel := context.WithTimeout(d.ctx, time.Second*30)
	defer cancel()
	if err := d.conductor.CommitUnsafePayload(ctx, x.Envelope); err != nil {
		d.emitter.Emit(d.ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to commit unsafe payload to conductor: %w", err),
		})
		return
	}

	// begin gossiping as soon as possible
	// asyncGossip.Clear() will be called later if an non-temporary error is found,
	// or if the payload is successfully inserted
	d.asyncGossip.Gossip(x.Envelope)
	// Now after having gossiped the block, try to put it in our own canonical chain
	d.emitter.Emit(d.ctx, engine.PayloadProcessEvent{
		Concluding:   x.Concluding,
		DerivedFrom:  x.DerivedFrom,
		BuildStarted: x.BuildStarted,
		Envelope:     x.Envelope,
		Ref:          x.Ref,
	})
	d.latest.Ref = x.Ref
	d.latestSealed = x.Ref
}

func (d *Sequencer) onPayloadSealInvalid(x engine.PayloadSealInvalidEvent) {
	if d.latest.Info != x.Info {
		return // not our payload, should be ignored.
	}
	d.log.Error("Sequencer could not seal block",
		"payloadID", x.Info.ID, "timestamp", x.Info.Timestamp, "err", x.Err)
	d.handleInvalid()
}

func (d *Sequencer) onPayloadSealExpiredError(x engine.PayloadSealExpiredErrorEvent) {
	if d.latest.Info != x.Info {
		return // not our payload, should be ignored.
	}
	d.log.Error("Sequencer temporarily could not seal block",
		"payloadID", x.Info.ID, "timestamp", x.Info.Timestamp, "err", x.Err)
	// Restart building, this way we get a block we should be able to seal
	// (smaller, since we adapt build time).
	d.handleInvalid()
}

func (d *Sequencer) onPayloadInvalid(x engine.PayloadInvalidEvent) {
	if d.latest.Ref.Hash != x.Envelope.ExecutionPayload.BlockHash {
		return // not a payload from the sequencer
	}
	d.log.Error("Sequencer could not insert payload",
		"block", x.Envelope.ExecutionPayload.ID(), "err", x.Err)
	d.handleInvalid()
}

func (d *Sequencer) onPayloadSuccess(x engine.PayloadSuccessEvent) {
	// d.latest as building state may already be empty,
	// if the forkchoice update (that dropped the stale building job) was received before the payload-success.
	if d.latest.Ref != (eth.L2BlockRef{}) && d.latest.Ref.Hash != x.Envelope.ExecutionPayload.BlockHash {
		// Not a payload that was built by this sequencer. We can ignore it, and continue upon forkchoice update.
		return
	}
	d.latest = BuildingState{}
	d.log.Info("Sequencer inserted block",
		"block", x.Ref, "parent", x.Envelope.ExecutionPayload.ParentID())
	// The payload was already published upon sealing.
	// Now that we have processed it ourselves we don't need it anymore.
	d.asyncGossip.Clear()
}

func (d *Sequencer) onSequencerAction(ev SequencerActionEvent) {
	d.log.Debug("Sequencer action")
	payload := d.asyncGossip.Get()
	if payload != nil {
		if d.latest.Info.ID == (eth.PayloadID{}) {
			d.log.Warn("Found reusable payload from async gossiper, and no block was being built. Reusing payload.",
				"hash", payload.ExecutionPayload.BlockHash,
				"number", uint64(payload.ExecutionPayload.BlockNumber),
				"parent", payload.ExecutionPayload.ParentHash)
		}
		ref, err := d.toBlockRef(d.rollupCfg, payload.ExecutionPayload)
		if err != nil {
			d.log.Error("Payload from async-gossip buffer could not be turned into block-ref", "err", err)
			d.asyncGossip.Clear() // bad payload
			return
		}
		d.log.Info("Resuming sequencing with previously async-gossip confirmed payload",
			"payload", payload.ExecutionPayload.ID())
		// Payload is known, we must have resumed sequencer-actions after a temporary error,
		// meaning that we have seen BuildSealedEvent already.
		// We can retry processing to make it canonical.
		d.emitter.Emit(d.ctx, engine.PayloadProcessEvent{
			Concluding:  false,
			DerivedFrom: eth.L1BlockRef{},
			Envelope:    payload,
			Ref:         ref,
		})
		d.latest.Ref = ref
	} else {
		if d.latest.Info != (eth.PayloadInfo{}) {
			// We should not repeat the seal request.
			d.nextActionOK = false
			// No known payload for block building job,
			// we have to retrieve it first.
			d.emitter.Emit(d.ctx, BuildSealEvent{
				Info:         d.latest.Info,
				BuildStarted: d.latest.Started,
				Concluding:   false,
				DerivedFrom:  eth.L1BlockRef{},
			})
		} else if d.latest == (BuildingState{}) {
			// If we have not started building anything, start building.
			d.startBuildingBlock()
		}
	}
}

func (d *Sequencer) onEngineTemporaryError(x rollup.EngineTemporaryErrorEvent) {
	if d.latest == (BuildingState{}) {
		d.log.Debug("Engine reported temporary error while building state is empty", "err", x.Err)
	}
	d.log.Error("Engine failed temporarily, backing off sequencer", "err", x.Err)
	if errors.Is(x.Err, engine.ErrEngineSyncing) { // if it is syncing we can back off by more
		d.nextAction = d.timeNow().Add(30 * time.Second)
	} else {
		d.nextAction = d.timeNow().Add(time.Second)
	}
	d.nextActionOK = d.active.Load()
	// We don't explicitly cancel block building jobs upon temporary errors: we may still finish the block (if any).
	// Any unfinished block building work eventually times out, and will be cleaned up that way.
	// Note that this only applies to temporary errors upon starting a block-building job.
	// If the engine errors upon sealing, an PayloadSealInvalidEvent will be get it to restart the attributes.

	// If we don't have an ID of a job to resume, then start over.
	// (d.latest.Onto would be set if we emitted BuildStart already)
	if d.latest.Info == (eth.PayloadInfo{}) {
		d.latest = BuildingState{}
	}
}

func (d *Sequencer) onReset(x rollup.ResetEvent) {
	d.log.Error("Sequencer encountered reset signal, aborting work", "err", x.Err)
	d.metrics.RecordSequencerReset()
	// try to cancel any ongoing payload building job
	if d.latest.Info != (eth.PayloadInfo{}) {
		d.emitter.Emit(d.ctx, BuildCancelEvent{Info: d.latest.Info})
	}
	d.latest = BuildingState{}
	// no action to perform until we get a reset-confirmation
	d.nextActionOK = false
}

func (d *Sequencer) onEngineResetConfirmedEvent(engine.EngineResetConfirmedEvent) {
	d.nextActionOK = d.active.Load()
	// Before sequencing we can wait a block,
	// assuming the execution-engine just churned through some work for the reset.
	// This will also prevent any potential reset-loop from running too hot.
	d.nextAction = d.timeNow().Add(time.Second * time.Duration(d.rollupCfg.BlockTime))
	d.log.Info("Engine reset confirmed, sequencer may continue", "next", d.nextActionOK)
}

func (d *Sequencer) onForkchoiceUpdate(x engine.ForkchoiceUpdateEvent) {
	d.log.Debug("Sequencer is processing forkchoice update", "unsafe", x.UnsafeL2Head, "latest", d.latestHead)

	if !d.active.Load() {
		d.setLatestHead(x.UnsafeL2Head)
		return
	}
	// If the safe head has fallen behind by a significant number of blocks, delay creating new blocks
	// until the safe lag is below SequencerMaxSafeLag.
	if maxSafeLag := d.maxSafeLag.Load(); maxSafeLag > 0 && x.SafeL2Head.Number+maxSafeLag <= x.UnsafeL2Head.Number {
		d.log.Warn("sequencer has fallen behind safe head by more than lag, stalling",
			"head", x.UnsafeL2Head, "safe", x.SafeL2Head, "max_lag", maxSafeLag)
		d.nextActionOK = false
	}
	// Drop stale block-building job if the chain has moved past it already.
	if d.latest != (BuildingState{}) && d.latest.Onto.Number < x.UnsafeL2Head.Number {
		d.log.Debug("Dropping stale/completed block-building job",
			"state", d.latest.Onto, "unsafe_head", x.UnsafeL2Head)
		// The cleared state will block further BuildStarted/BuildSealed responses from continuing the stale build job.
		d.latest = BuildingState{}
	}
	if x.UnsafeL2Head.Number > d.latestHead.Number {
		d.nextActionOK = true
		now := d.timeNow()
		blockTime := time.Duration(d.rollupCfg.BlockTime) * time.Second
		payloadTime := time.Unix(int64(x.UnsafeL2Head.Time+d.rollupCfg.BlockTime), 0)
		remainingTime := payloadTime.Sub(now)
		if remainingTime > blockTime {
			// if we have too much time, then wait before starting the build
			d.nextAction = payloadTime.Add(-blockTime)
		} else {
			// otherwise start instantly
			d.nextAction = now
		}
	}
	d.setLatestHead(x.UnsafeL2Head)
}
