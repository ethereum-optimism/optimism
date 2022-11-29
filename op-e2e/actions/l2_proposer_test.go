package actions

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

func TestProposer(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)

	rollupSeqCl := sequencer.RollupClient()
	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, rollupSeqCl, miner.EthClient(), seqEngine.EthClient())

	proposer := NewL2Proposer(t, log, &ProposerCfg{
		OutputOracleAddr:  sd.DeploymentsL1.L2OutputOracleProxy,
		ProposerKey:       dp.Secrets.Proposer,
		AllowNonFinalized: false,
	}, miner.EthClient(), sequencer.RollupClient())

	// L1 block
	miner.ActEmptyBlock(t)
	// L2 block
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	sequencer.ActBuildToL1Head(t)
	// submit and include in L1
	batcher.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)
	// finalize the first and second L1 blocks, including the batch
	miner.ActL1SafeNext(t)
	miner.ActL1SafeNext(t)
	miner.ActL1FinalizeNext(t)
	miner.ActL1FinalizeNext(t)
	// derive and see the L2 chain fully finalize
	sequencer.ActL2PipelineFull(t)
	sequencer.ActL1SafeSignal(t)
	sequencer.ActL1FinalizedSignal(t)
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, sequencer.SyncStatus().FinalizedL2)
	// make proposals until there is nothing left to propose
	for proposer.CanPropose(t) {
		// and propose it to L1
		proposer.ActMakeProposalTx(t)
		// include proposal on L1
		miner.ActL1StartBlock(12)(t)
		miner.ActL1IncludeTx(dp.Addresses.Proposer)(t)
		miner.ActL1EndBlock(t)
		// Check proposal was successful
		receipt, err := miner.EthClient().TransactionReceipt(t.Ctx(), proposer.LastProposalTx())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "proposal failed")
	}

	// check that L1 stored the expected output root
	outputOracleContract, err := bindings.NewL2OutputOracle(sd.DeploymentsL1.L2OutputOracleProxy, miner.EthClient())
	require.NoError(t, err)
	block := sequencer.SyncStatus().FinalizedL2
	outputOnL1, err := outputOracleContract.GetL2Output(nil, new(big.Int).SetUint64(block.Number))
	require.NoError(t, err)
	require.Less(t, block.Time, outputOnL1.Timestamp.Uint64(), "output is registered with L1 timestamp of proposal tx, past L2 block")
	outputComputed, err := sequencer.RollupClient().OutputAtBlock(t.Ctx(), block.Number)
	require.NoError(t, err)
	require.Equal(t, eth.Bytes32(outputOnL1.OutputRoot), outputComputed.OutputRoot, "output roots must match")
}
