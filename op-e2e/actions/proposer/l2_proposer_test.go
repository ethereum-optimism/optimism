package proposer

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/config"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	upgradesHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/upgrades/helpers"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/bindingspreview"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-proposer/proposer/source"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestProposerBatchType run each proposer-related test case in singular batch mode and span batch mode.
func TestProposerBatchType(t *testing.T) {
	t.Run("SingularBatch/Standard", func(t *testing.T) {
		runProposerTest(t, nil, config.DefaultAllocType)
	})
	t.Run("SpanBatch/Standard", func(t *testing.T) {
		deltaTimeOffset := hexutil.Uint64(0)
		runProposerTest(t, &deltaTimeOffset, config.DefaultAllocType)
	})
}

func runProposerTest(gt *testing.T, deltaTimeOffset *hexutil.Uint64, allocType config.AllocType) {
	t := actionsHelpers.NewDefaultTesting(gt)
	params := actionsHelpers.DefaultRollupTestParams()
	params.AllocType = allocType
	dp := e2eutils.MakeDeployParams(t, params)
	upgradesHelpers.ApplyDeltaTimeOffset(dp, deltaTimeOffset)
	sd := e2eutils.Setup(t, dp, actionsHelpers.DefaultAlloc)
	const (
		testClockOffset  = 30 * time.Minute
		proposalInterval = time.Hour
	)
	proposerClock := clock.NewDeterministicClock(time.Unix(int64(sd.L1Cfg.Timestamp), 0).Add(testClockOffset))
	log := testlog.Logger(t, log.LevelDebug)
	miner, seqEngine, sequencer := actionsHelpers.SetupSequencerTest(t, sd, log)
	sequencer.EnableProposerSuperRootAPI(t)
	superNodeClient := sources.NewSuperNodeClient(sequencer.RPCClient())
	proposalSource := source.NewSuperRootProposalSource(log, superNodeClient)

	rollupSeqCl := sequencer.RollupClient()
	batcher := actionsHelpers.NewL2Batcher(log, sd.RollupCfg, actionsHelpers.DefaultBatcherCfg(dp),
		rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	optimismPortal2Contract, err := bindingspreview.NewOptimismPortal2(sd.DeploymentsL1.OptimismPortalProxy, miner.EthClient())
	require.NoError(t, err)
	respectedGameType, err := optimismPortal2Contract.RespectedGameType(&bind.CallOpts{})
	require.NoError(t, err)
	proposer := actionsHelpers.NewL2Proposer(t, log, &actionsHelpers.ProposerCfg{
		DisputeGameFactoryAddr: &sd.DeploymentsL1.DisputeGameFactoryProxy,
		ProposalInterval:       proposalInterval,
		ProposalRetryInterval:  3 * time.Second,
		DisputeGameType:        respectedGameType,
		ProposerKey:            dp.Secrets.Proposer,
		AllowNonFinalized:      true,
		AllocType:              allocType,
		ChainID:                eth.ChainIDFromBig(sd.L1Cfg.Config.ChainID),
		Clock:                  proposerClock,
	}, miner.EthClient(), proposalSource)

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
	proposedL2 := sequencer.SyncStatus().FinalizedL2
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, proposedL2)
	status := sequencer.SyncStatus()
	rpcOutput, err := superNodeClient.SuperRootAtTimestamp(t.Ctx(), proposedL2.Time)
	require.NoError(t, err)
	require.Equal(t, status.CurrentL1.ID(), rpcOutput.CurrentL1)
	require.Equal(t, status.SafeL2.Time, rpcOutput.CurrentSafeTimestamp)
	require.Equal(t, status.FinalizedL2.Time, rpcOutput.CurrentFinalizedTimestamp)
	require.Equal(t, []eth.ChainID{eth.ChainIDFromBig(sd.RollupCfg.L2ChainID)}, rpcOutput.ChainIDs)
	require.NotNil(t, rpcOutput.Data)
	rpcSuperV1, ok := rpcOutput.Data.Super.(*eth.SuperV1)
	require.True(t, ok)
	require.Len(t, rpcSuperV1.Chains, 1)
	require.Equal(t, proposedL2.Time, rpcSuperV1.Timestamp)
	require.Equal(t, rpcOutput.Data.SuperRoot, eth.SuperRoot(rpcSuperV1))
	outputComputed, err := rollupSeqCl.OutputAtBlock(t.Ctx(), proposedL2.Number)
	require.NoError(t, err)
	require.Equal(t, outputComputed.OutputRoot, rpcSuperV1.Chains[0].Output)
	require.Equal(t, eth.ChainIDFromBig(sd.RollupCfg.L2ChainID), rpcSuperV1.Chains[0].ChainID)
	require.True(t, proposer.CanPropose(t))

	proposer.ActMakeProposalTx(t)
	// include proposal on L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Proposer)(t)
	miner.ActL1EndBlock(t)
	// Check proposal was successful
	receipt, err := miner.EthClient().TransactionReceipt(t.Ctx(), proposer.LastProposalTx())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "proposal failed")

	// Check the created game's timestamp exercises a real, nonzero proposal interval.
	disputeGameFactoryContract, err := bindings.NewDisputeGameFactory(sd.DeploymentsL1.DisputeGameFactoryProxy, miner.EthClient())
	require.NoError(t, err)
	gameCount, err := disputeGameFactoryContract.GameCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.Greater(t, bigs.Uint64Strict(gameCount), uint64(0), "game count must be greater than 0")
	latestGames, err := disputeGameFactoryContract.FindLatestGames(&bind.CallOpts{}, respectedGameType, new(big.Int).Sub(gameCount, common.Big1), common.Big1)
	require.NoError(t, err)
	require.Greater(t, len(latestGames), 0, "latest games must be greater than 0")
	latestGame := latestGames[0]
	gameTimestamp := time.Unix(int64(latestGame.Timestamp), 0)
	require.True(t, gameTimestamp.Before(proposerClock.Now()), "test clock must be after the created game's timestamp")
	require.True(t, gameTimestamp.After(proposerClock.Now().Add(-proposalInterval)), "created game must be inside the nonzero proposal interval")

	clockAdvance := proposalInterval + time.Second
	proposerClock.AdvanceTime(clockAdvance)
	require.False(t, proposer.CanPropose(t), "exact duplicate-game lookup must suppress the unchanged proposal after the proposal interval expires")
	proposerClock.AdvanceTime(-clockAdvance)

	// Advance the finalized L2 head while the first proposal is still inside the proposal interval.
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	sequencer.ActBuildToL1Head(t)
	batcher.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)
	miner.ActL1SafeNext(t)
	miner.ActL1SafeNext(t)
	miner.ActL1FinalizeNext(t)
	miner.ActL1FinalizeNext(t)
	sequencer.ActL2PipelineFull(t)
	sequencer.ActL1SafeSignal(t)
	sequencer.ActL1FinalizedSignal(t)
	sequencer.ActL2PipelineFull(t)
	newFinalizedL2 := sequencer.SyncStatus().FinalizedL2
	require.Greater(t, newFinalizedL2.Time, proposedL2.Time)
	newProposal, err := proposalSource.ProposalAtSequenceNum(t.Ctx(), newFinalizedL2.Time)
	require.NoError(t, err)
	require.NotEqual(t, common.Hash(rpcOutput.Data.SuperRoot), newProposal.Root, "new finalized L2 head must produce a different proposal root before testing interval throttling")
	require.False(t, proposer.CanPropose(t), "proposal interval must suppress a different proposal while the first game remains recent")

	// check that L1 stored the expected output root
	superRoot, err := eth.UnmarshalSuperRoot(latestGame.ExtraData)
	require.NoError(t, err)
	superV1, ok := superRoot.(*eth.SuperV1)
	require.True(t, ok)
	require.Len(t, superV1.Chains, 1)
	require.Equal(t, eth.ChainIDFromBig(sd.RollupCfg.L2ChainID), superV1.Chains[0].ChainID)
	require.Equal(t, proposedL2.Time, superV1.Timestamp)
	block, err := seqEngine.EthClient().BlockByNumber(t.Ctx(), new(big.Int).SetUint64(proposedL2.Number))
	require.NoError(t, err)
	require.Less(t, block.Time(), latestGame.Timestamp, "output is registered with L1 timestamp of proposal tx, past L2 block")
	require.Equal(t, outputComputed.OutputRoot, superV1.Chains[0].Output, "output roots must match")
	require.Equal(t, eth.Bytes32(latestGame.RootClaim), eth.SuperRoot(superV1), "super roots must match")
}
