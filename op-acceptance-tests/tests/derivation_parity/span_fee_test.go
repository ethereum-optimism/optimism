package derivation_parity

import (
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/testutils/parityreport"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestDerivationParitySpanBatchFeeAbove128Bits covers a span batch that gives a transaction a max
// fee per gas above 128 bits. A span batch stores a transaction's fields apart from its signature,
// so a batcher can change the fee without signing again.
//
// Pinned outcome:
//   - control: the span batch carries the transaction as its sender signed it. Both nodes include
//     it and record the same safe head.
//   - crafted: the span batch adds 2^128 to the transaction's max fee per gas. kona-node keeps only
//     the low 128 bits, which rebuilds the signed transaction, and includes it. op-node keeps the
//     full value, so the engine cannot decode the transaction and rejects the payload, and op-node
//     replaces the block with a deposits-only block.
//
// A fix to either node fails the crafted case: flip the kona-node assertion, and the record to
// agree.
func TestDerivationParitySpanBatchFeeAbove128Bits(gt *testing.T) {
	controlParity := &parityreport.Record{Case: "control", Agree: true}
	craftedParity := &parityreport.Record{Case: "crafted", Divergence: parityreport.DivergenceSpanBatchFee}
	recordWhenPassed(gt, controlParity)
	recordWhenPassed(gt, craftedParity)
	t := devtest.ParallelT(gt)
	sys := newParitySystem(t)
	require := t.Require()
	controlParity.Forks = latestForkAtGenesis(sys) + " and earlier forks at genesis"
	craftedParity.Forks = controlParity.Forks

	funder := dsl.NewKey(t, sys.Funder).User(sys.SequencerEL)
	controlTx := funder.Transfer(funder.Address(), eth.OneWei)
	craftedTx := funder.Transfer(funder.Address(), eth.OneWei)
	controlNum := bigs.Uint64Strict(controlTx.Included.Value().BlockNumber)
	craftedNum := bigs.Uint64Strict(craftedTx.Included.Value().BlockNumber)
	require.Less(controlNum, craftedNum, "the two transfers must be in different blocks")

	// control: the sequencer's blocks up to the first transfer, unchanged.
	controlL1 := sys.L2Network.PostBatches(sequencerBlocks(sys, 1, controlNum), dsl.WithSpanBatch)
	dsl.CheckAll(t,
		sys.OpNode.ReachedFn(safety.LocalSafe, controlNum, 30),
		sys.Kona.ReachedFn(safety.LocalSafe, controlNum, 30))
	sys.OpNode.AwaitMinL1Processed(controlL1.Number)
	sys.Kona.AwaitMinL1Processed(controlL1.Number)
	controlRecord := &eth.SafeHeadResponse{L1Block: controlL1.ID(), SafeHead: sys.SequencerEL.BlockRefByNumber(controlNum).ID()}
	require.Equal(controlRecord, sys.OpNode.SafeHeadRecordAtL1Block(controlL1.Number),
		"op-node must record the sequencer's block with the transfer as safe")
	require.Equal(controlRecord, sys.Kona.SafeHeadRecordAtL1Block(controlL1.Number),
		"kona-node must record the sequencer's block with the transfer as safe")

	// crafted: the sequencer's blocks after it, with the second transfer's fee raised by 2^128.
	// The sequencer stops first, so that no unsafe block it gossips contradicts op-node's
	// replacement.
	sys.Sequencer.StopSequencer()
	blocks := sequencerBlocks(sys, controlNum+1, craftedNum)
	sequencerCrafted := sys.SequencerEL.BlockRefByNumber(craftedNum)
	replaceTransaction(t, blocks[len(blocks)-1], craftedTx.Signed.Value(), withFeeAbove128Bits(t, craftedTx.Signed.Value()))
	craftedL1 := sys.L2Network.PostBatches(blocks, dsl.WithSpanBatch)

	dsl.CheckAll(t,
		sys.OpNode.ReachedFn(safety.LocalSafe, craftedNum, 30),
		sys.Kona.ReachedFn(safety.LocalSafe, craftedNum, 30))
	sys.OpNode.AwaitMinL1Processed(craftedL1.Number)
	sys.Kona.AwaitMinL1Processed(craftedL1.Number)
	// op-node: the engine rejects the payload, and Holocene replaces it.
	verifyDepositsOnly(t, sys.OpNodeEL, craftedNum)
	require.Equal(&eth.SafeHeadResponse{L1Block: craftedL1.ID(), SafeHead: sys.OpNodeEL.BlockRefByNumber(craftedNum).ID()},
		sys.OpNode.SafeHeadRecordAtL1Block(craftedL1.Number),
		"op-node must record the deposits-only replacement as safe")
	// kona-node: the truncated fee rebuilds the sequencer's transaction, so it keeps the sequencer's block.
	require.Equal(&eth.SafeHeadResponse{L1Block: craftedL1.ID(), SafeHead: sequencerCrafted.ID()},
		sys.Kona.SafeHeadRecordAtL1Block(craftedL1.Number),
		"kona-node must record the sequencer's block with the transfer as safe")
}

// sequencerBlocks returns copies of the sequencer's blocks from first to last, safe to modify.
func sequencerBlocks(sys *paritySystem, first, last uint64) []*eth.ExecutionPayload {
	var blocks []*eth.ExecutionPayload
	for number := first; number <= last; number++ {
		blocks = append(blocks, sequencerBlock(sys, number))
	}
	return blocks
}

// replaceTransaction swaps tx in block for replacement.
func replaceTransaction(t devtest.T, block *eth.ExecutionPayload, tx, replacement *types.Transaction) {
	i := slices.IndexFunc(block.Transactions, func(raw eth.Data) bool {
		var decoded types.Transaction
		return decoded.UnmarshalBinary(raw) == nil && decoded.Hash() == tx.Hash()
	})
	t.Require().GreaterOrEqual(i, 0, "block %d must hold transaction %s", block.BlockNumber, tx.Hash())
	raw, err := replacement.MarshalBinary()
	t.Require().NoError(err)
	block.Transactions[i] = raw
}

// withFeeAbove128Bits returns tx with 2^128 added to its max fee per gas, and the signature it had.
func withFeeAbove128Bits(t devtest.T, tx *types.Transaction) *types.Transaction {
	t.Require().Equal(uint8(types.DynamicFeeTxType), tx.Type(), "the transfer must be an EIP-1559 transaction")
	v, r, s := tx.RawSignatureValues()
	return types.NewTx(&types.DynamicFeeTx{
		ChainID:    tx.ChainId(),
		Nonce:      tx.Nonce(),
		GasTipCap:  tx.GasTipCap(),
		GasFeeCap:  new(big.Int).Add(tx.GasFeeCap(), new(big.Int).Lsh(big.NewInt(1), 128)),
		Gas:        tx.Gas(),
		To:         tx.To(),
		Value:      tx.Value(),
		Data:       tx.Data(),
		AccessList: tx.AccessList(),
		V:          v,
		R:          r,
		S:          s,
	})
}

// latestForkAtGenesis returns the newest L2 fork active at genesis.
func latestForkAtGenesis(sys *paritySystem) string {
	genesis := sys.L2Network.TimestampForBlockNum(0)
	for _, fork := range slices.Backward(forks.All) {
		if sys.L2Network.IsForkActiveAt(fork, genesis) {
			return string(fork)
		}
	}
	return string(forks.Bedrock)
}
