package upgrades

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
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

	env.ActBatchSubmitAllAndMine(t)

	// verifier picks up the L2 chain that was submitted
	env.Verifier.ActL1HeadSignal(t)
	env.Verifier.ActL2PipelineFull(t)
	require.Equal(t, env.Verifier.L2Safe(), env.Seq.L2Unsafe(), "verifier syncs from sequencer via L1")
	require.NotEqual(t, env.Seq.L2Safe(), env.Seq.L2Unsafe(), "sequencer has not processed L1 yet")
}
