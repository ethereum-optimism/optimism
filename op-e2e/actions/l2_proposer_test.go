package actions

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/bindingspreview"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestProposerBatchType run each proposer-related test case in singular batch mode and span batch mode.
func TestProposerBatchType(t *testing.T) {
	tests := []struct {
		name string
		f    func(gt *testing.T, deltaTimeOffset *hexutil.Uint64)
	}{
		{"RunProposerTest", RunProposerTest},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name+"_SingularBatch", func(t *testing.T) {
			test.f(t, nil)
		})
	}

	deltaTimeOffset := hexutil.Uint64(0)
	for _, test := range tests {
		test := test
		t.Run(test.name+"_SpanBatch", func(t *testing.T) {
			test.f(t, &deltaTimeOffset)
		})
	}
}

func RunProposerTest(gt *testing.T, deltaTimeOffset *hexutil.Uint64) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	dp.DeployConfig.L2GenesisDeltaTimeOffset = deltaTimeOffset
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)

	rollupSeqCl := sequencer.RollupClient()
	batcher := NewL2Batcher(log, sd.RollupCfg, DefaultBatcherCfg(dp),
		rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	var proposer *L2Proposer
	if e2eutils.UseFaultProofs() {
		optimismPortal2Contract, err := bindingspreview.NewOptimismPortal2(sd.DeploymentsL1.OptimismPortalProxy, miner.EthClient())
		require.NoError(t, err)
		respectedGameType, err := optimismPortal2Contract.RespectedGameType(&bind.CallOpts{})
		require.NoError(t, err)
		proposer = NewL2Proposer(t, log, &ProposerCfg{
			DisputeGameFactoryAddr: &sd.DeploymentsL1.DisputeGameFactoryProxy,
			ProposalInterval:       6 * time.Second,
			DisputeGameType:        respectedGameType,
			ProposerKey:            dp.Secrets.Proposer,
			AllowNonFinalized:      true,
		}, miner.EthClient(), rollupSeqCl)
	} else {
		proposer = NewL2Proposer(t, log, &ProposerCfg{
			OutputOracleAddr:  &sd.DeploymentsL1.L2OutputOracleProxy,
			ProposerKey:       dp.Secrets.Proposer,
			AllowNonFinalized: false,
		}, miner.EthClient(), rollupSeqCl)
	}

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
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, sequencer.SyncStatus().FinalizedL2)
	require.True(t, proposer.CanPropose(t))

	// make proposals until there is nothing left to propose
	for proposer.CanPropose(t) {
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
	if e2eutils.UseFaultProofs() {
		optimismPortal2Contract, err := bindingspreview.NewOptimismPortal2(sd.DeploymentsL1.OptimismPortalProxy, miner.EthClient())
		require.NoError(t, err)
		respectedGameType, err := optimismPortal2Contract.RespectedGameType(&bind.CallOpts{})
		require.NoError(t, err)
		disputeGameFactoryContract, err := bindings.NewDisputeGameFactory(sd.DeploymentsL1.DisputeGameFactoryProxy, miner.EthClient())
		require.NoError(t, err)
		gameCount, err := disputeGameFactoryContract.GameCount(&bind.CallOpts{})
		require.NoError(t, err)
		require.Greater(t, gameCount.Uint64(), uint64(0), "game count must be greater than 0")
		latestGames, err := disputeGameFactoryContract.FindLatestGames(&bind.CallOpts{}, respectedGameType, new(big.Int).Sub(gameCount, common.Big1), common.Big1)
		require.NoError(t, err)
		require.Greater(t, len(latestGames), 0, "latest games must be greater than 0")
		latestGame := latestGames[0]
		gameBlockNumber := new(big.Int)
		gameBlockNumber.SetBytes(latestGame.ExtraData[0:32])
		block, err := seqEngine.EthClient().BlockByNumber(t.Ctx(), gameBlockNumber)
		require.NoError(t, err)
		require.Less(t, block.Time(), latestGame.Timestamp, "output is registered with L1 timestamp of proposal tx, past L2 block")
		outputComputed, err := sequencer.RollupClient().OutputAtBlock(t.Ctx(), gameBlockNumber.Uint64())
		require.NoError(t, err)
		require.Equal(t, eth.Bytes32(latestGame.RootClaim), outputComputed.OutputRoot, "output roots must match")
	} else {
		outputOracleContract, err := bindings.NewL2OutputOracle(sd.DeploymentsL1.L2OutputOracleProxy, miner.EthClient())
		require.NoError(t, err)
		blockNumber, err := outputOracleContract.LatestBlockNumber(&bind.CallOpts{})
		require.NoError(t, err)
		require.Greater(t, int64(blockNumber.Uint64()), int64(0), "latest block number must be greater than 0")
		block, err := seqEngine.EthClient().BlockByNumber(t.Ctx(), blockNumber)
		require.NoError(t, err)
		outputOnL1, err := outputOracleContract.GetL2OutputAfter(&bind.CallOpts{}, blockNumber)
		require.NoError(t, err)
		require.Less(t, block.Time(), outputOnL1.Timestamp.Uint64(), "output is registered with L1 timestamp of proposal tx, past L2 block")
		outputComputed, err := sequencer.RollupClient().OutputAtBlock(t.Ctx(), blockNumber.Uint64())
		require.NoError(t, err)
		require.Equal(t, eth.Bytes32(outputOnL1.OutputRoot), outputComputed.OutputRoot, "output roots must match")
	}
}
