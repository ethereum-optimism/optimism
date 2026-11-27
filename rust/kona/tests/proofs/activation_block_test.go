package proofs

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
)

// TestActivationBlockTxOmission tests that the sequencer omits user transactions in activation
// blocks and that batches that contain user transactions in an activation block are dropped.
func TestActivationBlockTxOmission(gt *testing.T) {
	matrix := helpers.NewMatrix[forks.Name]()
	for _, fork := range forks.From(forks.Jovian) {
		preFork := lookupHardforkHelper(forks.Prev(fork))
		require.NotNil(gt, preFork, "no hardfork helper for the fork before %s", fork)
		matrix.AddTestCase(string(fork), fork, helpers.NewForkMatrix(preFork), testActivationBlockTxOmission, helpers.ExpectNoError())
	}
	matrix.Run(gt)
}

// activationUpgradeTxs returns the upgrade deposits of fork's activation block on a chain whose
// dependency set holds only itself.
func activationUpgradeTxs(t actionsHelpers.Testing, fork forks.Name) []hexutil.Bytes {
	var (
		txs []hexutil.Bytes
		err error
	)
	switch fork {
	case forks.Jovian:
		txs, err = derive.JovianNetworkUpgradeTransactions()
	case forks.Lagoon:
		txs, _, err = derive.LagoonActivationUpgradeTransactions(false)
	default:
		txs, _, err = derive.UpgradeTransactions(fork)
	}
	require.NoError(t, err, "upgrade transactions of %s", fork)
	return txs
}

func testActivationBlockTxOmission(gt *testing.T, testCfg *helpers.TestCfg[forks.Name]) {
	fork := testCfg.Custom
	t := actionsHelpers.NewDefaultTesting(gt)
	offset := uint64(4)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		// activate fork after a few blocks
		dc.SetForkTimeOffset(fork, &offset)
	}
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	engine := env.Engine
	sequencer := env.Sequencer
	miner := env.Miner
	rollupCfg := env.Sd.RollupCfg
	blockTime := rollupCfg.BlockTime

	miner.ActEmptyBlock(t)

	sequencer.ActL1HeadSignal(t)
	for i := 0; i < int(offset)-1; i++ {
		sequencer.ActL2EmptyBlock(t)
	}
	tx := env.Alice.L2.ActMakeTx(t)
	sequencer.ActL2StartBlock(t)
	// we assert later that the sequencer actually omits this tx in the activation block
	engine.ActL2IncludeTx(env.Alice.Address())
	sequencer.ActL2EndBlock(t)

	actHeader := engine.LatestHeader(t)
	require.Equal(t, fork,
		rollupCfg.IsActivationBlock(actHeader.Time-blockTime, actHeader.Time),
		"this block should be the activation block")
	deposits, userTxs := engine.BlockTxs(t, actHeader.Hash())
	require.Len(t, deposits, len(activationUpgradeTxs(t, fork))+1, "activation block contains unexpected deposits")
	require.Empty(t, userTxs, "activation block contains unexpected user txs")

	batcher := env.Batcher
	for i := 0; i < int(offset)-1; i++ {
		batcher.ActL2BatchBuffer(t)
	}
	batcher.ActL2BatchBuffer(t,
		actionsHelpers.WithBlockModifier(func(payload *eth.ExecutionPayload) *eth.ExecutionPayload {
			// inject user tx into activation batch
			payload.Transactions = append(payload.Transactions, actionsHelpers.EncodeTx(t, tx))
			return payload
		}))

	batcher.ActL2ChannelClose(t)
	batcher.ActL2BatchSubmit(t)
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	recs := env.Logs.FindLogs(testlog.NewMessageFilter("dropping batch with user transactions in fork activation block"))
	require.Len(t, recs, 1)

	l2SafeHead := engine.SafeHeader(t)
	preActHeader := engine.HeaderByHash(t, actHeader.ParentHash)
	require.Equal(t, eth.HeaderBlockID(preActHeader), eth.HeaderBlockID(l2SafeHead), "derivation only reaches pre-upgrade block")

	env.RunFaultProofProgramFromGenesis(t, bigs.Uint64Strict(l2SafeHead.Number), testCfg.CheckResult, testCfg.InputParams...)

	// The L1 chain up to its head holds no batch for the activation block once the batch carrying
	// the user tx is dropped, and the sequencing window has not expired, so the honest claim for
	// the activation block is the invalid transition.
	actNum := bigs.Uint64Strict(actHeader.Number)
	env.RunFaultProofProgram(t, actNum, helpers.ExpectNoError(), helpers.WithL2Claim(eth.InvalidTransitionHash))
	// A program that kept the batch would have the engine reject the user tx and replace the block
	// with the deposits-only block the sequencer built, which the default claim commits to.
	env.RunFaultProofProgram(t, actNum, helpers.ExpectError(helpers.ErrClaimNotValid))
}
