package sdmtest

import (
	"bytes"
	"io"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// maxBatchFrameSize keeps the hand-built channel in a single calldata frame.
const maxBatchFrameSize = 120_000

// PostTamperedSingularBatches stands in for a malicious batcher: it posts the sequencer's blocks
// [from, to] to L1 as one singular-batch channel signed with the batcher key, after tamper has
// rewritten the raw transactions of block `to`. The real batcher must be stopped.
func PostTamperedSingularBatches(t devtest.T, sys *RethSystem, from, to uint64, tamper func([]eth.Data) []eth.Data) {
	rollupCfg := sys.L2Network.Escape().RollupConfig()
	comp, err := compressor.NewNonCompressor(compressor.Config{
		TargetOutputSize: maxBatchFrameSize,
		CompressionAlgo:  derive.Zlib,
	})
	t.Require().NoError(err)
	channel, err := derive.NewSingularChannelOut(comp, rollup.NewChainSpec(rollupCfg))
	t.Require().NoError(err)

	for number := from; number <= to; number++ {
		ref := sys.L2EL.BlockRefByNumber(number)
		txs := RawBlockTransactions(t, sys.L2EL, number)
		if number == to {
			txs = tamper(txs)
		}
		_, err := channel.AddBlock(rollupCfg, &eth.ExecutionPayload{
			ParentHash:   ref.ParentHash,
			BlockHash:    ref.Hash,
			BlockNumber:  eth.Uint64Quantity(number),
			Timestamp:    eth.Uint64Quantity(ref.Time),
			Transactions: txs,
		})
		t.Require().NoError(err, "block %d must fit in the channel", number)
	}
	t.Require().NoError(channel.Close())

	data := bytes.NewBuffer([]byte{params.DerivationVersion0})
	_, err = channel.OutputFrame(data, maxBatchFrameSize)
	t.Require().ErrorIs(err, io.EOF, "the channel must fit in one frame")

	batcherKey := sys.L2Network.Escape().Keys().Secret(devkeys.BatcherRole.Key(rollupCfg.L2ChainID))
	batcher := dsl.NewEOA(dsl.NewKey(t, batcherKey), sys.L1EL)
	batcher.Transact(
		batcher.Plan(),
		txplan.WithTo(&rollupCfg.BatchInboxAddress),
		txplan.WithData(data.Bytes()),
	)
}

// RawBlockTransactions returns the EIP-2718 encoding of every transaction in a block, in order.
// Raw bytes avoid decoding the 0x7D PostExec transaction into go-ethereum types.
func RawBlockTransactions(t devtest.T, l2EL *dsl.L2ELNode, blockNum uint64) []eth.Data {
	block := GetBlockWithTxs(t, l2EL, blockNum)
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	txs := make([]eth.Data, len(block.Transactions))
	for i := range txs {
		err := rpcClient.CallContext(t.Ctx(), &txs[i], "eth_getRawTransactionByBlockNumberAndIndex",
			hexutil.Uint64(blockNum), hexutil.Uint(i))
		t.Require().NoError(err, "eth_getRawTransactionByBlockNumberAndIndex RPC failed for tx %d of block %d", i, blockNum)
		t.Require().NotEmpty(txs[i], "raw tx %d of block %d not found", i, blockNum)
	}
	return txs
}
