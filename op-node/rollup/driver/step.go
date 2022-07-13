package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum/go-ethereum/log"
)

type outputImpl struct {
	dl     Downloader
	l2     derive.Engine
	log    log.Logger
	Config *rollup.Config
}

func (d *outputImpl) createNewBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.L1BlockRef) (eth.L2BlockRef, *eth.ExecutionPayload, error) {
	d.log.Info("creating new block", "parent", l2Head, "l1Origin", l1Origin)

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	attrs, _, err := derive.PreparePayloadAttributes(fetchCtx, d.Config, d.dl, l2Head, l1Origin.ID())
	if err != nil {
		return l2Head, nil, err
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

	// Actually execute the block and add it to the head of the chain.
	payload, rpcErr, payloadErr := derive.InsertHeadBlock(ctx, d.log, d.l2, fc, attrs, false)
	if rpcErr != nil {
		return l2Head, nil, fmt.Errorf("failed to extend L2 chain due to RPC error: %v", rpcErr)
	}
	if payloadErr != nil {
		return l2Head, nil, fmt.Errorf("failed to extend L2 chain, cannot produce valid payload: %v", payloadErr)
	}

	// Generate an L2 block ref from the payload.
	ref, err := derive.PayloadToBlockRef(payload, &d.Config.Genesis)

	return ref, payload, err
}
