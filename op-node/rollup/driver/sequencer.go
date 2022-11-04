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

// Sequencer implements the sequencing interface of the driver: it starts and completes block building jobs.
type Sequencer struct {
	log    log.Logger
	config *rollup.Config

	l1 Downloader
	l2 derive.Engine

	buildingOnto eth.ForkchoiceState
	buildingID   eth.PayloadID
}

func NewSequencer(log log.Logger, cfg *rollup.Config, l1 Downloader, l2 derive.Engine) *Sequencer {
	return &Sequencer{
		log:    log,
		config: cfg,
		l1:     l1,
		l2:     l2,
	}
}

// StartBuildingBlock initiates a block building job on top of the given L2 head, safe and finalized blocks, and using the provided l1Origin.
func (d *Sequencer) StartBuildingBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.L1BlockRef) error {
	d.log.Info("creating new block", "parent", l2Head, "l1Origin", l1Origin)
	if d.buildingID != (eth.PayloadID{}) { // This may happen when we decide to build a different block in response to a reorg. Or when previous block building failed.
		d.log.Warn("did not finish previous block building, starting new building now", "prev_onto", d.buildingOnto.HeadBlockHash, "prev_payload_id", d.buildingID, "new_onto", l2Head)
	}

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
		SafeBlockHash:      l2SafeHead.Hash,
		FinalizedBlockHash: l2Finalized.Hash,
	}
	// Start a payload building process.
	id, errTyp, err := derive.StartPayload(ctx, d.l2, fc, attrs)
	if err != nil {
		return fmt.Errorf("failed to start building on top of L2 chain %s, error (%d): %w", l2Head, errTyp, err)
	}
	d.buildingOnto = fc
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

	// Actually execute the block and add it to the head of the chain.
	payload, errTyp, err := derive.ConfirmPayload(ctx, d.log, d.l2, d.buildingOnto, d.buildingID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to complete building on top of L2 chain %s, error (%d): %w", d.buildingOnto.HeadBlockHash, errTyp, err)
	}
	d.buildingID = eth.PayloadID{}
	return payload, nil
}

// CreateNewBlock sequences a L2 block with immediate building and sealing.
func (d *Sequencer) CreateNewBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.L1BlockRef) (eth.L2BlockRef, *eth.ExecutionPayload, error) {
	if err := d.StartBuildingBlock(ctx, l2Head, l2SafeHead, l2Finalized, l1Origin); err != nil {
		return l2Head, nil, err
	}

	payloadTime := time.Unix(int64(l2Head.Time+d.config.BlockTime), 0)
	remaining := -time.Until(payloadTime)
	// TODO: allowing to breathe when remaining time is in the negative is very generous,
	//  we can reduce this if the block building timing gets better with PR 3818
	d.log.Debug("using remaining time for better block production", "remaining_time", remaining)
	time.Sleep(500 * time.Millisecond)

	payload, err := d.CompleteBuildingBlock(ctx)
	if err != nil {
		return l2Head, nil, err
	}

	// Generate an L2 block ref from the payload.
	ref, err := derive.PayloadToBlockRef(payload, &d.config.Genesis)

	return ref, payload, err
}
