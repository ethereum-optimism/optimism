package upgrades

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

func TestHoloceneActivationAtGenesis(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	env := helpers.SetupEnv(t)

	// Start op-nodes
	env.Seq.ActL2PipelineFull(t)
	env.Verifier.ActL2PipelineFull(t)

	// Verify Holocene is active at genesis
	l2Head := env.Seq.L2Unsafe()
	require.NotZero(t, l2Head.Hash)
	require.True(t, env.SetupData.RollupCfg.IsHolocene(l2Head.Time), "Holocene should be active at genesis")

	// Advance the L1 chain
	env.Miner.ActEmptyBlock(t)
	env.Miner.ActEmptyBlock(t)

	// build empty L1 block
	env.Miner.ActEmptyBlock(t)
	// finalize it, so the L1 geth blob pool doesn't log errors about missing finality
	env.Miner.ActL1SafeNext(t)
	env.Miner.ActL1FinalizeNext(t)

	// Build L2 chain and advance safe head
	env.Seq.ActL1HeadSignal(t)
	env.Seq.ActBuildToL1Head(t)

	// verify in logs that stage transformations happened
	recs := env.Logs.FindLogs(testlog.NewMessageContainsFilter("transforming to Holocene"), testlog.NewAttributesFilter("role", e2esys.RoleSeq))
	require.Len(t, recs, 3)
	recs = env.Logs.FindLogs(testlog.NewMessageContainsFilter("transforming to Holocene"), testlog.NewAttributesFilter("role", e2esys.RoleVerif))
	require.Len(t, recs, 3)

	// Submit L2
	env.Batcher.ActSubmitAll(t)
	batchTx := env.Batcher.LastSubmitted

	// new L1 block with L2 batch
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(batchTx.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// env.Verifier picks up the L2 chain that was submitted
	env.Verifier.ActL1HeadSignal(t)
	env.Verifier.ActL2PipelineFull(t)
	require.Equal(t, env.Verifier.L2Safe(), env.Seq.L2Unsafe(), "verifier syncs from sequencer via L1")
	require.NotEqual(t, env.Seq.L2Safe(), env.Seq.L2Unsafe(), "sequencer has not processed L1 yet")
}
