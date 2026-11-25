package dsl

import (
	"bytes"
	"io"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// maxCraftedFrameSize keeps a crafted channel in one calldata frame, well under the L1
// transaction size limit.
const maxCraftedFrameSize = 120_000

// PostBatchesOpts configures L2Network.PostBatches.
type PostBatchesOpts struct {
	// BatchType is derive.SingularBatchType (the default) or derive.SpanBatchType.
	BatchType int
}

// WithSpanBatch makes PostBatches encode the blocks as one span batch.
func WithSpanBatch(opts *PostBatchesOpts) {
	opts.BatchType = derive.SpanBatchType
}

// PostBatches stands in for a batcher that posts crafted data. It encodes blocks, in order, as one
// zlib channel with op-node's batch encoding, and sends the channel in a single frame as calldata
// to the batch inbox, signed with the batcher key. The blocks need not be ones any node built: a
// test can change their transactions or parent, so that the verifiers derive blocks the sequencer
// never built. Stop the honest batcher first, or its frames interleave with these.
//
// It waits for the batch transaction to be included and returns the L1 block that includes it.
func (n *L2Network) PostBatches(blocks []*eth.ExecutionPayload, opts ...func(*PostBatchesOpts)) eth.L1BlockRef {
	cfg := applyOpts(PostBatchesOpts{BatchType: derive.SingularBatchType}, opts...)
	n.require.NotEmpty(blocks, "a crafted channel needs at least one block")
	rollupCfg := n.inner.RollupConfig()
	chainSpec := rollup.NewChainSpec(rollupCfg)

	var channel derive.ChannelOut
	switch cfg.BatchType {
	case derive.SingularBatchType:
		comp, err := compressor.NewNonCompressor(compressor.Config{
			TargetOutputSize: maxCraftedFrameSize,
			CompressionAlgo:  derive.Zlib,
		})
		n.require.NoError(err)
		channel, err = derive.NewSingularChannelOut(comp, chainSpec)
		n.require.NoError(err)
	case derive.SpanBatchType:
		var err error
		channel, err = derive.NewSpanChannelOut(maxCraftedFrameSize, derive.Zlib, chainSpec)
		n.require.NoError(err)
	default:
		n.require.FailNowf("unknown batch type", "batch type %d", cfg.BatchType)
	}

	for _, block := range blocks {
		_, err := channel.AddBlock(rollupCfg, block)
		n.require.NoError(err, "block %d must fit in the crafted channel", block.BlockNumber)
	}
	n.require.NoError(channel.Close())
	data := bytes.NewBuffer([]byte{params.DerivationVersion0})
	_, err := channel.OutputFrame(data, maxCraftedFrameSize)
	n.require.ErrorIs(err, io.EOF, "the crafted channel must fit in one frame")

	first, last := blocks[0].BlockNumber, blocks[len(blocks)-1].BlockNumber
	n.log.Info("Posting crafted batches", "chain", n.ChainID(), "batchType", cfg.BatchType,
		"firstBlock", first, "lastBlock", last, "channel", channel.ID(), "bytes", data.Len())
	batcherKey := n.inner.Keys().Secret(devkeys.BatcherRole.Key(rollupCfg.L2ChainID))
	batcher := NewEOA(NewKey(n.t, batcherKey), n.PrimaryL1EL())
	tx := batcher.Transact(
		batcher.Plan(),
		txplan.WithTo(&rollupCfg.BatchInboxAddress),
		txplan.WithData(data.Bytes()),
	)
	inclusion, err := tx.IncludedBlock.Eval(n.ctx)
	n.require.NoError(err, "the crafted batch transaction must be included")
	n.log.Info("Crafted batches included", "chain", n.ChainID(), "firstBlock", first, "lastBlock", last, "l1", inclusion)
	return inclusion
}
