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
	Fetch(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Transactions, eth.ReceiptsFetcher, error)
}

// Sequencer implements the sequencing interface of the driver: it starts and completes block building jobs.
type Sequencer struct {
	L1     Downloader
	L2     derive.Engine
	Log    log.Logger
	Config *rollup.Config

	buildingOnto eth.ForkchoiceState
	buildingID   eth.PayloadID
}

// StartBuildingBlock initiates a block building job on top of the given L2 head, safe and finalized blocks, and using the provided l1Origin.
func (d *Sequencer) StartBuildingBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.L1BlockRef) error {
	d.Log.Info("creating new block", "parent", l2Head, "l1Origin", l1Origin)
	if d.buildingID != (eth.PayloadID{}) { // This may happen when we decide to build a different block in response to a reorg. Or when previous block building failed.
		d.Log.Warn("did not finish previous block building, starting new building now", "prev_onto", d.buildingOnto.HeadBlockHash, "prev_payload_id", d.buildingID, "new_onto", l2Head)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	attrs, err := derive.PreparePayloadAttributes(fetchCtx, d.Config, d.L1, l2Head, l2Head.Time+d.Config.BlockTime, l1Origin.ID())
	if err != nil {
		return err
	}

	// If our next L2 block timestamp is beyond the Sequencer drift threshold, then we must produce
	// empty blocks (other than the L1 info deposit and any user deposits). We handle this by
	// setting NoTxPool to true, which will cause the Sequencer to not include any transactions
	// from the transaction pool.
	attrs.NoTxPool = uint64(attrs.Timestamp) >= l1Origin.Time+d.Config.MaxSequencerDrift

	// And construct our fork choice state. This is our current fork choice state and will be
	// updated as a result of executing the block based on the attributes described above.
	fc := eth.ForkchoiceState{
		HeadBlockHash:      l2Head.Hash,
		SafeBlockHash:      l2SafeHead.Hash,
		FinalizedBlockHash: l2Finalized.Hash,
	}
	// Start a payload building process.
	id, errTyp, err := derive.StartPayload(ctx, d.L2, fc, attrs)
	if err != nil {
		return fmt.Errorf("failed to start building on top of L2 chain %s, error (%d): %w", l2Head, errTyp, err)
	}
	d.buildingOnto = fc
	d.buildingID = id
	return nil
}

// CompleteBuildingBlock takes the current block that is being built, and asks the engine to complete the building, seal the block, and persist it as canonical.
func (d *Sequencer) CompleteBuildingBlock(ctx context.Context) (*eth.ExecutionPayload, error) {
	if d.buildingID == (eth.PayloadID{}) {
		return nil, fmt.Errorf("cannot complete payload building: not currently building a payload")
	}

	// Actually execute the block and add it to the head of the chain.
	payload, errTyp, err := derive.ConfirmPayload(ctx, d.Log, d.L2, d.buildingOnto, d.buildingID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to complete building on top of L2 chain %s, error (%d): %w", d.buildingOnto.HeadBlockHash, errTyp, err)
	}
	return payload, nil
}

// createNewBlock sequences a L2 block with immediate building and sealing.
// Deprecated: the sequencer should build the block in two steps, adjusted to maximize building time, within the block-time span, or faster when catching up.
func (d *Sequencer) createNewBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.L1BlockRef) (eth.L2BlockRef, *eth.ExecutionPayload, error) {
	if err := d.StartBuildingBlock(ctx, l2Head, l2SafeHead, l2Finalized, l1Origin); err != nil {
		return l2Head, nil, err
	}
	payload, err := d.CompleteBuildingBlock(ctx)
	if err != nil {
		return l2Head, nil, err
	}
	d.buildingID = eth.PayloadID{}

	// Generate an L2 block ref from the payload.
	ref, err := derive.PayloadToBlockRef(payload, &d.Config.Genesis)

	return ref, payload, err
}
