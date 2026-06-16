package upgrades

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

// TestKarstActivationBlockGasLimit verifies that the Karst activation block is
// granted extra gas to accommodate the network upgrade transactions (NUTs)
// loaded from the Karst NUT bundle, and that the block gas limit returns to the
// system config value on the very next block.
//
// See op-node/rollup/derive/attributes.go: on the Karst (L2CM) activation block
// the payload gas limit is computed as sysConfig.GasLimit + nutGas.
func TestKarstActivationBlockGasLimit(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	// Activate Karst a few blocks after genesis so we can observe the pre-,
	// activation-, and post-activation blocks.
	karstOffset := uint64(24)
	env := helpers.SetupEnv(t, helpers.WithActiveFork(forks.Karst, karstOffset))

	// Start op-nodes.
	env.Seq.ActL2PipelineFull(t)
	env.Verifier.ActL2PipelineFull(t)

	// Karst should not be active at genesis yet.
	l2Head := env.Seq.L2Unsafe()
	require.NotZero(t, l2Head.Hash)
	require.False(t, env.SetupData.RollupCfg.IsKarst(l2Head.Time), "Karst should not be active at genesis")

	// The normal block gas limit is the system config gas limit.
	normalGasLimit := env.SeqEngine.L2Chain().CurrentBlock().GasLimit
	require.NotZero(t, normalGasLimit)

	// The Karst NUT bundle reserves additional gas for the activation block.
	_, nutGas, err := derive.UpgradeTransactions(forks.Karst)
	require.NoError(t, err)
	require.NotZero(t, nutGas, "Karst NUT bundle should reserve gas")

	// Build empty L2 blocks until Karst activates. ActBuildL2ToFork stops on
	// the first block in which the fork is active, i.e. the activation block.
	activationBlock := env.Seq.ActBuildL2ToFork(t, forks.Karst)
	require.True(t, env.SetupData.RollupCfg.IsKarstActivationBlock(activationBlock.Time),
		"expected to land exactly on the Karst activation block")

	// The activation block gas limit is bumped by the NUT gas allocation.
	activationGasLimit := env.SeqEngine.L2Chain().CurrentBlock().GasLimit
	require.Greaterf(t, activationGasLimit, normalGasLimit,
		"Karst activation block should have an increased gas limit: activation=%d normal=%d", activationGasLimit, normalGasLimit)
	require.Equalf(t, normalGasLimit+nutGas, activationGasLimit,
		"Karst activation block gas limit should be the system config gas limit plus the NUT gas: want=%d (normal=%d + nut=%d) got=%d",
		normalGasLimit+nutGas, normalGasLimit, nutGas, activationGasLimit)

	// The very next block returns to the normal system config gas limit.
	env.Seq.ActL2EmptyBlock(t)
	nextBlock := env.Seq.L2Unsafe()
	require.False(t, env.SetupData.RollupCfg.IsKarstActivationBlock(nextBlock.Time),
		"the block after activation should not be the activation block")
	postActivationGasLimit := env.SeqEngine.L2Chain().CurrentBlock().GasLimit
	require.Equalf(t, normalGasLimit, postActivationGasLimit,
		"gas limit should return to the system config value after the activation block: want=%d got=%d", normalGasLimit, postActivationGasLimit)
}
