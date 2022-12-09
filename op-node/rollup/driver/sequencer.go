package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type Downloader interface {
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type SequencerMetrics interface {
	RecordSequencingError()

	CountSequencedTxs(count int)

	RecordSequencerBuildingDiffTime(duration time.Duration)
	RecordSequencerSealingTime(duration time.Duration)
}

type L1OriginSelectorIface interface {
	FindL1Origin(ctx context.Context, l1Head eth.L1BlockRef, l2Head eth.L2BlockRef) (eth.L1BlockRef, error)
}

type EngineState interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
	Origin() eth.L1BlockRef

	SetUnsafeHead(head eth.L2BlockRef)
}

// Sequencer implements the sequencing interface of the driver: it starts and completes block building jobs.
type Sequencer struct {
	log    log.Logger
	config *rollup.Config

	l2          derive.Engine
	engineState EngineState

	attrBuilder      derive.AttributesBuilder
	l1OriginSelector L1OriginSelectorIface

	buildingOnto      eth.L2BlockRef
	buildingID        eth.PayloadID
	buildingStartTime time.Time

	nextAction time.Time

	metrics SequencerMetrics
}

func NewSequencer(log log.Logger, cfg *rollup.Config, l2 derive.Engine, engineState EngineState, attributesBuilder derive.AttributesBuilder, l1OriginSelector L1OriginSelectorIface, metrics SequencerMetrics) *Sequencer {
	return &Sequencer{
		log:              log,
		config:           cfg,
		l2:               l2,
		metrics:          metrics,
		engineState:      engineState,
		attrBuilder:      attributesBuilder,
		l1OriginSelector: l1OriginSelector,
	}
}

// StartBuildingBlock initiates a block building job on top of the given L2 head, safe and finalized blocks, and using the provided l1Origin.
func (d *Sequencer) StartBuildingBlock(ctx context.Context, l1Head eth.L1BlockRef) error {
	l2Head := d.engineState.UnsafeL2Head()

	// Figure out which L1 origin block we're going to be building on top of.
	l1Origin, err := d.l1OriginSelector.FindL1Origin(ctx, l1Head, l2Head)
	if err != nil {
		d.log.Error("Error finding next L1 Origin", "err", err)
		return err
	}

	if !(l2Head.L1Origin.Hash == l1Origin.ParentHash || l2Head.L1Origin.Hash == l1Origin.Hash) {
		return fmt.Errorf("cannot build new L2 block with L1 origin %s (parent L1 %s) on current L2 head %s with L1 origin %s", l1Origin, l1Origin.ParentHash, l2Head, l2Head.L1Origin)
	}

	d.log.Info("creating new block", "parent", l2Head, "l1Origin", l1Origin)
	if d.buildingID != (eth.PayloadID{}) { // This may happen when we decide to build a different block in response to a reorg. Or when previous block building failed.
		d.log.Warn("did not finish previous block building, starting new building now", "prev_onto", d.buildingOnto, "prev_payload_id", d.buildingID, "new_onto", l2Head)
	}
	d.buildingStartTime = time.Now()

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

	// And construct our fork choice state. This is our current fork choice state and will be
	// updated as a result of executing the block based on the attributes described above.
	fc := eth.ForkchoiceState{
		HeadBlockHash:      l2Head.Hash,
		SafeBlockHash:      d.engineState.SafeL2Head().Hash,
		FinalizedBlockHash: d.engineState.Finalized().Hash,
	}
	// Start a payload building process.
	id, errTyp, err := derive.StartPayload(ctx, d.l2, fc, attrs)
	if err != nil {
		return fmt.Errorf("failed to start building on top of L2 chain %s, error (%d): %w", l2Head, errTyp, err)
	}
	d.buildingOnto = l2Head
	d.buildingID = id
	return nil
}

// CompleteBuildingBlock takes the current block that is being built, and asks the engine to complete the building, seal the block, and persist it as canonical.
// Warning: the safe and finalized L2 blocks as viewed during the initiation of the block building are reused for completion of the block building.
// The Execution engine should not change the safe and finalized blocks between start and completion of block building.
func (d *Sequencer) CompleteBuildingBlock(ctx context.Context) (*eth.ExecutionPayload, error) {
	if d.buildingID == (eth.PayloadID{}) {
		return nil, fmt.Errorf("cannot complete payload building: not currently building a payload")
	}
	sealingStart := time.Now()

	l2Head := d.engineState.UnsafeL2Head()
	if d.buildingOnto.Hash != l2Head.Hash {
		return nil, fmt.Errorf("engine reorged from %s to %s while building block", d.buildingOnto, l2Head)
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      l2Head.Hash,
		SafeBlockHash:      d.engineState.SafeL2Head().Hash,
		FinalizedBlockHash: d.engineState.Finalized().Hash,
	}

	// Actually execute the block and add it to the head of the chain.
	payload, errTyp, err := derive.ConfirmPayload(ctx, d.log, d.l2, fc, d.buildingID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to complete building on top of L2 chain %s, id: %s, error (%d): %w", d.buildingOnto, d.buildingID, errTyp, err)
	}
	now := time.Now()
	sealTime := now.Sub(sealingStart)
	buildTime := now.Sub(d.buildingStartTime)
	d.metrics.RecordSequencerSealingTime(sealTime)
	d.metrics.RecordSequencerBuildingDiffTime(buildTime - time.Duration(d.config.BlockTime)*time.Second)
	d.metrics.CountSequencedTxs(len(payload.Transactions))
	d.buildingID = eth.PayloadID{}

	// Generate an L2 block ref from the payload.
	newUnsafeL2Head, err := derive.PayloadToBlockRef(payload, &d.config.Genesis)
	if err != nil {
		return nil, fmt.Errorf("sequenced payload %s cannot be transformed into valid L2 block reference: %w", payload.ID(), err)
	}
	// Update our L2 head block based on the new unsafe block we just generated.
	d.engineState.SetUnsafeHead(newUnsafeL2Head)
	d.log.Info("Sequenced new L2 block", "l2_unsafe", newUnsafeL2Head, "l1_origin", newUnsafeL2Head.L1Origin,
		"txs", len(payload.Transactions), "time", newUnsafeL2Head.Time, "seal_time", sealTime, "build_time", buildTime)
	return payload, nil
}

// CancelBuildingBlock cancels the current open block building job.
// This sequencer only maintains one block building job at a time.
func (d *Sequencer) CancelBuildingBlock(ctx context.Context) {
	d.log.Error("cancelling old block sealing job", "payload", d.buildingID)
	_, err := d.l2.GetPayload(ctx, d.buildingID)
	d.log.Error("failed to cancel block building job", "payload", d.buildingID, "err", err)
}

// PlanNextSequencerAction returns a desired delay till the RunNextSequencerAction call.
func (d *Sequencer) PlanNextSequencerAction() time.Duration {
	head := d.engineState.UnsafeL2Head()
	now := time.Now()

	// We may have to wait till the next sequencing action, e.g. upon an error.
	// If the head changed we need to respond and will not delay the sequencing.
	if delay := d.nextAction.Sub(now); delay > 0 && d.buildingOnto.Hash == head.Hash {
		return delay
	}

	blockTime := time.Duration(d.config.BlockTime) * time.Second
	payloadTime := time.Unix(int64(head.Time+d.config.BlockTime), 0)
	remainingTime := payloadTime.Sub(now)

	// If we started building a block already, and if that work is still consistent,
	// then we would like to finish it by sealing the block.
	if d.buildingID != (eth.PayloadID{}) && d.buildingOnto.Hash == head.Hash {
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
	return d.buildingOnto
}

// RunNextSequencerAction starts new block building work, or seals existing work,
// and is best timed by first awaiting the delay returned by PlanNextSequencerAction.
// If a new block is successfully sealed, it will be returned for publishing, nil otherwise.
func (d *Sequencer) RunNextSequencerAction(ctx context.Context, l1Head eth.L1BlockRef) *eth.ExecutionPayload {
	if d.buildingID != (eth.PayloadID{}) {
		payload, err := d.CompleteBuildingBlock(ctx)
		if err != nil {
			d.log.Error("sequencer failed to seal new block", "err", err)
			d.metrics.RecordSequencingError()
			d.nextAction = time.Now().Add(time.Second)
			if d.buildingID != (eth.PayloadID{}) { // don't keep stale block building jobs around, try to cancel them
				d.CancelBuildingBlock(ctx)
				// We always reset the building ID, and do not try to cancel repeatedly.
				// If it's not just already stopped, then the building job will expire within 12 seconds.
				d.buildingID = eth.PayloadID{}
			}
			return nil
		} else {
			d.log.Info("sequencer successfully built a new block", "block", payload.ID(), "time", uint64(payload.Timestamp), "txs", len(payload.Transactions))
			return payload
		}
	} else {
		err := d.StartBuildingBlock(ctx, l1Head)
		if err != nil {
			d.log.Error("sequencer failed to start building new block", "err", err)
			d.metrics.RecordSequencingError()
			d.nextAction = time.Now().Add(time.Second)
		} else {
			d.log.Info("sequencer start building new block", "payload_id", d.buildingID)
		}
		return nil
	}
}
