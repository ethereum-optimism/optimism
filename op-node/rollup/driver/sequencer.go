package driver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Downloader interface {
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type L1OriginSelectorIface interface {
	FindL1Origin(ctx context.Context, l2Head eth.L2BlockRef) (eth.L1BlockRef, error)
}

type SequencerMetrics interface {
	RecordSequencerInconsistentL1Origin(from eth.BlockID, to eth.BlockID)
	RecordSequencerReset()
}

// Sequencer implements the sequencing interface of the driver: it starts and completes block building jobs.
type Sequencer struct {
	log    log.Logger
	config *rollup.Config

	engine derive.ResettableEngineControl

	attrBuilder      derive.AttributesBuilder
	l1OriginSelector L1OriginSelectorIface

	metrics SequencerMetrics

	// timeNow enables sequencer testing to mock the time
	timeNow func() time.Time

	nextAction time.Time
}

func NewSequencer(log log.Logger, cfg *rollup.Config, engine derive.ResettableEngineControl, attributesBuilder derive.AttributesBuilder, l1OriginSelector L1OriginSelectorIface, metrics SequencerMetrics) *Sequencer {
	return &Sequencer{
		log:              log,
		config:           cfg,
		engine:           engine,
		timeNow:          time.Now,
		attrBuilder:      attributesBuilder,
		l1OriginSelector: l1OriginSelector,
		metrics:          metrics,
	}
}

// StartBuildingBlock initiates a block building job on top of the given L2 head, safe and finalized blocks, and using the provided l1Origin.
func (d *Sequencer) StartBuildingBlock(ctx context.Context) error {
	l2Head := d.engine.UnsafeL2Head()

	// Figure out which L1 origin block we're going to be building on top of.
	l1Origin, err := d.l1OriginSelector.FindL1Origin(ctx, l2Head)
	if err != nil {
		d.log.Error("Error finding next L1 Origin", "err", err)
		return err
	}

	if !(l2Head.L1Origin.Hash == l1Origin.ParentHash || l2Head.L1Origin.Hash == l1Origin.Hash) {
		d.metrics.RecordSequencerInconsistentL1Origin(l2Head.L1Origin, l1Origin.ID())
		return derive.NewResetError(fmt.Errorf("cannot build new L2 block with L1 origin %s (parent L1 %s) on current L2 head %s with L1 origin %s", l1Origin, l1Origin.ParentHash, l2Head, l2Head.L1Origin))
	}

	d.log.Info("creating new block", "parent", l2Head, "l1Origin", l1Origin)

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	attrs, err := d.attrBuilder.PreparePayloadAttributes(fetchCtx, l2Head, l1Origin.ID())
	if err != nil {
		return err
	}

	// If our next L2 block timestamp is beyond the Sequencer drift threshold, then we must produce
	// empty blocks (other than the L1 info deposit and any user deposits). We handle this by
	// setting NoTxPool to true, which will cause the Sequencer to not include any transactions
	// from the transaction pool.
	attrs.NoTxPool = uint64(attrs.Timestamp) > l1Origin.Time+d.config.MaxSequencerDrift

	d.log.Debug("prepared attributes for new block",
		"num", l2Head.Number+1, "time", uint64(attrs.Timestamp),
		"origin", l1Origin, "origin_time", l1Origin.Time, "noTxPool", attrs.NoTxPool)

	// Start a payload building process.
	errTyp, err := d.engine.StartPayload(ctx, l2Head, attrs, false)
	if err != nil {
		return fmt.Errorf("failed to start building on top of L2 chain %s, error (%d): %w", l2Head, errTyp, err)
	}
	return nil
}

// CompleteBuildingBlock takes the current block that is being built, and asks the engine to complete the building, seal the block, and persist it as canonical.
// Warning: the safe and finalized L2 blocks as viewed during the initiation of the block building are reused for completion of the block building.
// The Execution engine should not change the safe and finalized blocks between start and completion of block building.
func (d *Sequencer) CompleteBuildingBlock(ctx context.Context) (*eth.ExecutionPayload, error) {
	payload, errTyp, err := d.engine.ConfirmPayload(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to complete building block: error (%d): %w", errTyp, err)
	}
	return payload, nil
}

// CancelBuildingBlock cancels the current open block building job.
// This sequencer only maintains one block building job at a time.
func (d *Sequencer) CancelBuildingBlock(ctx context.Context) {
	// force-cancel, we can always continue block building, and any error is logged by the engine state
	_ = d.engine.CancelPayload(ctx, true)
}

// PlanNextSequencerAction returns a desired delay till the RunNextSequencerAction call.
func (d *Sequencer) PlanNextSequencerAction() time.Duration {
	// If the engine is busy building safe blocks (and thus changing the head that we would sync on top of),
	// then give it time to sync up.
	if onto, _, safe := d.engine.BuildingPayload(); safe {
		d.log.Warn("delaying sequencing to not interrupt safe-head changes", "onto", onto, "onto_time", onto.Time)
		// approximates the worst-case time it takes to build a block, to reattempt sequencing after.
		return time.Second * time.Duration(d.config.BlockTime)
	}

	head := d.engine.UnsafeL2Head()
	now := d.timeNow()

	buildingOnto, buildingID, _ := d.engine.BuildingPayload()

	// We may have to wait till the next sequencing action, e.g. upon an error.
	// If the head changed we need to respond and will not delay the sequencing.
	if delay := d.nextAction.Sub(now); delay > 0 && buildingOnto.Hash == head.Hash {
		return delay
	}

	blockTime := time.Duration(d.config.BlockTime) * time.Second
	payloadTime := time.Unix(int64(head.Time+d.config.BlockTime), 0)
	remainingTime := payloadTime.Sub(now)

	// If we started building a block already, and if that work is still consistent,
	// then we would like to finish it by sealing the block.
	if buildingID != (eth.PayloadID{}) && buildingOnto.Hash == head.Hash {
		// if we started building already, then we will schedule the sealing.
		if remainingTime < sealingDuration {
			return 0 // if there's not enough time for sealing, don't wait.
		} else {
			// finish with margin of sealing duration before payloadTime
			return remainingTime - sealingDuration
		}
	} else {
		// if we did not yet start building, then we will schedule the start.
		if remainingTime > blockTime {
			// if we have too much time, then wait before starting the build
			return remainingTime - blockTime
		} else {
			// otherwise start instantly
			return 0
		}
	}
}

// BuildingOnto returns the L2 head reference that the latest block is or was being built on top of.
func (d *Sequencer) BuildingOnto() eth.L2BlockRef {
	ref, _, _ := d.engine.BuildingPayload()
	return ref
}

// RunNextSequencerAction starts new block building work, or seals existing work,
// and is best timed by first awaiting the delay returned by PlanNextSequencerAction.
// If a new block is successfully sealed, it will be returned for publishing, nil otherwise.
//
// Only critical errors are bubbled up, other errors are handled internally.
// Internally starting or sealing of a block may fail with a derivation-like error:
//   - If it is a critical error, the error is bubbled up to the caller.
//   - If it is a reset error, the ResettableEngineControl used to build blocks is requested to reset, and a backoff applies.
//     No attempt is made at completing the block building.
//   - If it is a temporary error, a backoff is applied to reattempt building later.
//   - If it is any other error, a backoff is applied and building is cancelled.
//
// Upon L1 reorgs that are deep enough to affect the L1 origin selection, a reset-error may occur,
// to direct the engine to follow the new L1 chain before continuing to sequence blocks.
// It is up to the EngineControl implementation to handle conflicting build jobs of the derivation
// process (as verifier) and sequencing process.
// Generally it is expected that the latest call interrupts any ongoing work,
// and the derivation process does not interrupt in the happy case,
// since it can consolidate previously sequenced blocks by comparing sequenced inputs with derived inputs.
// If the derivation pipeline does force a conflicting block, then an ongoing sequencer task might still finish,
// but the derivation can continue to reset until the chain is correct.
// If the engine is currently building safe blocks, then that building is not interrupted, and sequencing is delayed.
func (d *Sequencer) RunNextSequencerAction(ctx context.Context) (*eth.ExecutionPayload, error) {
	if onto, buildingID, safe := d.engine.BuildingPayload(); buildingID != (eth.PayloadID{}) {
		if safe {
			d.log.Warn("avoiding sequencing to not interrupt safe-head changes", "onto", onto, "onto_time", onto.Time)
			// approximates the worst-case time it takes to build a block, to reattempt sequencing after.
			d.nextAction = d.timeNow().Add(time.Second * time.Duration(d.config.BlockTime))
			return nil, nil
		}
		payload, err := d.CompleteBuildingBlock(ctx)
		if err != nil {
			if errors.Is(err, derive.ErrCritical) {
				return nil, err // bubble up critical errors.
			} else if errors.Is(err, derive.ErrReset) {
				d.log.Error("sequencer failed to seal new block, requiring derivation reset", "err", err)
				d.metrics.RecordSequencerReset()
				d.nextAction = d.timeNow().Add(time.Second * time.Duration(d.config.BlockTime)) // hold off from sequencing for a full block
				d.CancelBuildingBlock(ctx)
				d.engine.Reset()
			} else if errors.Is(err, derive.ErrTemporary) {
				d.log.Error("sequencer failed temporarily to seal new block", "err", err)
				d.nextAction = d.timeNow().Add(time.Second)
				// We don't explicitly cancel block building jobs upon temporary errors: we may still finish the block.
				// Any unfinished block building work eventually times out, and will be cleaned up that way.
			} else {
				d.log.Error("sequencer failed to seal block with unclassified error", "err", err)
				d.nextAction = d.timeNow().Add(time.Second)
				d.CancelBuildingBlock(ctx)
			}
			return nil, nil
		} else {
			d.log.Info("sequencer successfully built a new block", "block", payload.ID(), "time", uint64(payload.Timestamp), "txs", len(payload.Transactions))
			return payload, nil
		}
	} else {
		err := d.StartBuildingBlock(ctx)
		if err != nil {
			if errors.Is(err, derive.ErrCritical) {
				return nil, err
			} else if errors.Is(err, derive.ErrReset) {
				d.log.Error("sequencer failed to seal new block, requiring derivation reset", "err", err)
				d.metrics.RecordSequencerReset()
				d.nextAction = d.timeNow().Add(time.Second * time.Duration(d.config.BlockTime)) // hold off from sequencing for a full block
				d.engine.Reset()
			} else if errors.Is(err, derive.ErrTemporary) {
				d.log.Error("sequencer temporarily failed to start building new block", "err", err)
				d.nextAction = d.timeNow().Add(time.Second)
			} else {
				d.log.Error("sequencer failed to start building new block with unclassified error", "err", err)
				d.nextAction = d.timeNow().Add(time.Second)
			}
		} else {
			parent, buildingID, _ := d.engine.BuildingPayload() // we should have a new payload ID now that we're building a block
			d.log.Info("sequencer started building new block", "payload_id", buildingID, "l2_parent_block", parent, "l2_parent_block_time", parent.Time)
		}
		return nil, nil
	}
}
