package upgrades

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestJovianActivationAtGenesis(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	env := helpers.SetupEnv(t, helpers.WithActiveGenesisFork(rollup.Jovian))

	// Start op-nodes
	env.Seq.ActL2PipelineFull(t)
	env.Verifier.ActL2PipelineFull(t)

	// Verify Jovian is active at genesis
	l2Head := env.Seq.L2Unsafe()
	require.NotZero(t, l2Head.Hash)
	require.True(t, env.SetupData.RollupCfg.IsJovian(l2Head.Time), "Jovian should be active at genesis")

	// build empty L1 block
	env.Miner.ActEmptyBlock(t)

	// Build L2 chain and advance safe head
	env.Seq.ActL1HeadSignal(t)
	env.Seq.ActBuildToL1Head(t)

	// verify in logs that correct stage got activated
	recs := env.Logs.FindLogs(testlog.NewMessageContainsFilter("activating Jovian stage during reset"), testlog.NewAttributesFilter("role", e2esys.RoleSeq))
	require.Len(t, recs, 2)
	recs = env.Logs.FindLogs(testlog.NewMessageContainsFilter("activating Jovian stage during reset"), testlog.NewAttributesFilter("role", e2esys.RoleVerif))
	require.Len(t, recs, 2)

	env.ActBatchSubmitAllAndMine(t)

	// verifier picks up the L2 chain that was submitted
	env.Verifier.ActL1HeadSignal(t)
	env.Verifier.ActL2PipelineFull(t)
	require.Equal(t, env.Verifier.L2Safe(), env.Seq.L2Unsafe(), "verifier syncs from sequencer via L1")
	require.NotEqual(t, env.Seq.L2Safe(), env.Seq.L2Unsafe(), "sequencer has not processed L1 yet")
}

func TestJovianInvalidPayload(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	env := helpers.SetupEnv(t, helpers.WithActiveGenesisFork(rollup.Jovian))
	ctx := context.Background()

	requireDepositOnlyLogs := func(role string, expNumLogs int) {
		t.Helper()
		recs := env.Logs.FindLogs(testlog.NewMessageContainsFilter("deposits-only attributes"), testlog.NewAttributesFilter("role", role))
		require.Len(t, recs, expNumLogs)
	}

	// Start op-nodes
	env.Seq.ActL2PipelineFull(t)

	// generate and batch buffer two empty blocks
	env.Seq.ActL2EmptyBlock(t) // 1 - genesis is 0
	env.Batcher.ActL2BatchBuffer(t)
	env.Seq.ActL2EmptyBlock(t) // 2
	env.Batcher.ActL2BatchBuffer(t)

	// send and include a single transaction
	env.Alice.L2.ActResetTxOpts(t)
	env.Alice.L2.ActSetTxToAddr(&env.DeployParams.Addresses.Bob)
	env.Alice.L2.ActMakeTx(t)

	env.Seq.ActL2StartBlock(t)
	env.SeqEngine.ActL2IncludeTx(env.Alice.Address())(t)
	env.Seq.ActL2EndBlock(t) // 3
	env.Alice.L2.ActCheckReceiptStatusOfLastTx(true)(t)
	l2Unsafe := env.Seq.L2Unsafe()
	const invalidNum = 3
	require.EqualValues(t, invalidNum, l2Unsafe.Number)
	b, err := env.SeqEngine.EthClient().BlockByNumber(ctx, big.NewInt(invalidNum))
	require.NoError(t, err)
	require.Len(t, b.Transactions(), 2)

	// buffer into the batcher, invalidating the tx via signature zeroing
	env.Batcher.ActL2BatchBuffer(t, helpers.WithBlockModifier(func(block *types.Block) *types.Block {
		// Replace the tx with one that has a bad signature.
		txs := block.Transactions()
		newTx, err := txs[1].WithSignature(env.Alice.L2.Signer(), make([]byte, 65))
		require.NoError(t, err)
		txs[1] = newTx
		return block
	}))

	// generate two more empty blocks
	env.Seq.ActL2EmptyBlock(t) // 4
	env.Seq.ActL2EmptyBlock(t) // 5
	require.EqualValues(t, 5, env.Seq.L2Unsafe().Number)

	// submit it all
	env.ActBatchSubmitAllAndMine(t)

	// derive chain on sequencer
	env.Seq.ActL1HeadSignal(t)
	env.Seq.ActL2PipelineFull(t)

	l2Safe := env.Seq.L2Safe()
	require.EqualValues(t, invalidNum, l2Safe.Number)
	require.NotEqual(t, l2Safe.Hash, l2Unsafe.Hash, // old L2Unsafe above
		"block-3 should have been replaced by deposit-only version")
	requireDepositOnlyLogs(e2esys.RoleSeq, 2)
	require.Equal(t, l2Safe, env.Seq.L2Unsafe(), "unsafe chain should have reorg'd")
	b, err = env.SeqEngine.EthClient().BlockByNumber(ctx, big.NewInt(invalidNum))
	require.NoError(t, err)
	require.Len(t, b.Transactions(), 1)

	// test that building on top of reorg'd chain and deriving further works

	env.Seq.ActL2EmptyBlock(t) // 4
	env.Seq.ActL2EmptyBlock(t) // 5
	l2Unsafe = env.Seq.L2Unsafe()
	require.EqualValues(t, 5, l2Unsafe.Number)

	env.Batcher.Reset() // need to reset batcher to become aware of reorg
	env.ActBatchSubmitAllAndMine(t)
	env.Seq.ActL1HeadSignal(t)
	env.Seq.ActL2PipelineFull(t)
	require.Equal(t, l2Unsafe, env.Seq.L2Safe())
}
