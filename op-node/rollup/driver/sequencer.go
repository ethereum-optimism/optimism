package driver

import (
	"context"
	"errors"
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
	RecordSequencerBuildingDiffTime(duration time.Duration)
	RecordSequencerSealingTime(duration time.Duration)
}

type EngineState interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
	Origin() eth.L1BlockRef
}

// Sequencer implements the sequencing interface of the driver: it starts and completes block building jobs.
type Sequencer struct {
	log    log.Logger
	config *rollup.Config

	l1          Downloader
	l2          derive.Engine
	engineState EngineState

	buildingOnto      eth.L2BlockRef
	buildingID        eth.PayloadID
	buildingStartTime time.Time

	metrics SequencerMetrics
}

func NewSequencer(log log.Logger, cfg *rollup.Config, l1 Downloader, l2 derive.Engine, engineState EngineState, metrics SequencerMetrics) *Sequencer {
	return &Sequencer{
		log:         log,
		config:      cfg,
		l1:          l1,
		l2:          l2,
		metrics:     metrics,
		engineState: engineState,
	}
}

// StartBuildingBlock initiates a block building job on top of the given L2 head, safe and finalized blocks, and using the provided l1Origin.
func (d *Sequencer) StartBuildingBlock(ctx context.Context, l1Origin eth.L1BlockRef) error {
	l2Head := d.engineState.UnsafeL2Head()
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

	attrs, err := derive.PreparePayloadAttributes(fetchCtx, d.config, d.l1, d.l2, l2Head, l2Head.Time+d.config.BlockTime, l1Origin.ID())
	if err != nil {
		return err
	}

	// If our next L2 block timestamp is beyond the Sequencer drift threshold, then we must produce
	// empty blocks (other than the L1 info deposit and any user deposits). We handle this by
	// setting NoTxPool to true, which will cause the Sequencer to not include any transactions
	// from the transaction pool.
	attrs.NoTxPool = uint64(attrs.Timestamp) >= l1Origin.Time+d.config.MaxSequencerDrift

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
	d.log.Debug("sequenced block", "seal_time", sealTime, "build_time", buildTime)
	d.buildingID = eth.PayloadID{}
	return payload, nil
}

// PlanNextSequencerAction returns a desired delay till the next action, and if we should seal the block:
// - true whenever we need to complete a block
// - false whenever we need to start a block
func (d *Sequencer) PlanNextSequencerAction(sequenceErr error) (delay time.Duration, seal bool, onto eth.BlockID) {
	blockTime := time.Duration(d.config.BlockTime) * time.Second
	head := d.engineState.UnsafeL2Head()

	// based on the build error, delay and start over again
	if sequenceErr != nil {
		if errors.Is(sequenceErr, UninitializedL1StateErr) {
			// temporary errors are not so bad, just retry in 500ms
			return 500 * time.Millisecond, false, head.ID()
		} else {
			// we just hit an unknown type of error, delay a re-attempt by as much as a block
			return blockTime, false, head.ID()
		}
	}

	payloadTime := time.Unix(int64(head.Time+d.config.BlockTime), 0)
	remainingTime := time.Until(payloadTime)

	// If we started building a block already, and if that work is still consistent,
	// then we would like to finish it by sealing the block.
	if d.buildingID != (eth.PayloadID{}) && d.buildingOnto.Hash == head.Hash {
		// if we started building already, then we will schedule the sealing.
		if remainingTime < sealingDuration {
			return 0, true, head.ID() // if there's not enough time for sealing, don't wait.
		} else {
			// finish with margin of sealing duration before payloadTime
			return remainingTime - sealingDuration, true, head.ID()
		}
	} else {
		// if we did not yet start building, then we will schedule the start.
		if remainingTime > blockTime {
			// if we have too much time, then wait before starting the build
			return remainingTime - blockTime, false, head.ID()
		} else {
			// otherwise start instantly
			return 0, false, head.ID()
		}
	}
}
