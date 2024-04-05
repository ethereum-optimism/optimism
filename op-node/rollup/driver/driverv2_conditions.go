package driver

import (
	"context"
	"time"
)

// Design note: we can make the check/do conditions public, such that we can manually operate a driver,
// and no longer have to duplicate the behavior in the op-e2e action tests.

func (d *DriverV2) checkGenerateAttributes() bool {
	// TODO: check if derivation is exhausted at current L1 block, and if we don't currently have attributes buffered
	return false
}

func (d *DriverV2) doGenerateAttributes() {
	// TODO: pull attributes from derivation, buffer for processing, signal attributes related conditions
}

func (d *DriverV2) checkUnsafeBlockSyncTrigger() bool {
	d.payloadsLock.RLock()
	defer d.payloadsLock.RUnlock()

	// TODO check sync mode, we may not want to trigger sync, even if we can
	maxBlock := d.payloads.Max()
	if maxBlock == nil {
		return false
	}
	// trigger sync if the latest block in the buffer is ahead of the tip by more than 1 block.
	// Note: d.unsafeHead is locked and safe to use.
	// TODO: maybe with separate Lock/Unlock for the condition and effect parts we can make this safe explicitly?
	return d.unsafeHead.Number+1 < uint64(maxBlock.ExecutionPayload.BlockNumber)
}

func (d *DriverV2) doUnsafeBlockSyncTrigger() {
	d.payloadsLock.Lock()
	defer d.payloadsLock.Unlock()
	maxBlock := d.payloads.Max()
	if maxBlock == nil {
		return
	}
	// TODO use timeout constants, same timeout as previously, etc.
	ctx, cancel := context.WithTimeout(d.lifetimeCtx, time.Second*10)
	defer cancel()
	err := d.engineController.InsertUnsafePayload(ctx, maxBlock, d.unsafeHead.L2BlockRef)
	if err != nil {
		d.log.Error("failed to trigger sync with unsafe block", "block", maxBlock, "head", d.unsafeHead.L2BlockRef, "err", err)
	}
	d.onNewUnsafeBlock()
}

func (d *DriverV2) checkUnsafeBlockProcessing() bool {
	d.payloadsLock.RLock()
	defer d.payloadsLock.RUnlock()

	next := d.payloads.Peek()
	// if older: to be dropped by the effect
	// if equal: to be processed by the effect
	return next != nil && uint64(next.ExecutionPayload.BlockNumber) <= d.unsafeHead.Number+1
}

func (d *DriverV2) doUnsafeBlockProcessing() {
	d.payloadsLock.Lock()
	defer d.payloadsLock.Unlock()
	next := d.payloads.Pop()
	if next == nil {
		return
	}
	if uint64(next.ExecutionPayload.BlockNumber) < d.unsafeHead.Number+1 {
		// drop the old payload
		d.log.Debug("already processed unsafe block past this height", "block", next, "head", d.unsafeHead.L2BlockRef)
		return
	}
	// TODO use timeout constants, same timeout as previously, etc.
	ctx, cancel := context.WithTimeout(d.lifetimeCtx, time.Second*10)
	defer cancel()
	err := d.engineController.InsertUnsafePayload(ctx, next, d.unsafeHead.L2BlockRef)
	if err != nil {
		d.log.Error("failed to process next unsafe block", "block", next, "head", d.unsafeHead.L2BlockRef, "err", err)
	}
	d.onNewUnsafeBlock()
}

func (d *DriverV2) checkSequencerAction() bool {
	if !d.cfg.SequencerEnabled || d.cfg.SequencerStopped {
		return false
	}
	next := d.sequencer.PlanNextSequencerAction()
	if next < time.Millisecond*10 {
		return true
	}
	// we can wait, just schedule a signal to check back later.
	time.AfterFunc(next, d.sequencerAction.Signal)
	return false
}

func (d *DriverV2) doSequencerAction() {
	// TODO use timeout constants, same timeout as previously, etc.
	ctx, cancel := context.WithTimeout(d.lifetimeCtx, time.Second*10)
	defer cancel()
	_, err := d.sequencer.RunNextSequencerAction(ctx, d.asyncGossiper, d.sequencerConductor)
	if err != nil {
		d.log.Error("failed to sequencer action", "head", d.unsafeHead.L2BlockRef, "err", err)
	}
	d.sequencerAction.Signal() // schedule the next sequencer action
}

// onNewUnsafeBlock is a helper function to signal all unsafe-block related jobs,
// if the unsafe-head changed due to a non-sequencer action.
func (d *DriverV2) onNewUnsafeBlock() {
	d.unsafeHead.L2BlockRef = d.engineController.UnsafeL2Head()
	d.unsafeBlockProcessor.Signal()
	d.sequencerAction.Signal()
	d.unsafeBlockSyncTrigger.Signal()
}
