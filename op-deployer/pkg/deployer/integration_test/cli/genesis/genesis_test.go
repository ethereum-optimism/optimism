package genesis

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	test "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/cli"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestApplyGenesisStrategy tests genesis deployment with custom L1 parameters
func TestApplyGenesisStrategy(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lgr, l1RPC, _, _, pk, dk, l1ChainID := test.SetupAnvilTest(t)
	l2ChainID := uint256.NewInt(1)

	pragueOffset := uint64(2000)
	l1GenesisParams := &state.L1DevGenesisParams{
		BlockParams: state.L1DevGenesisBlockParams{
			Timestamp:     1000,
			GasLimit:      42_000_000,
			ExcessBlobGas: 9000,
		},
		PragueTimeOffset: &pragueOffset,
	}

	deployChain := func(l1DevGenesisParams *state.L1DevGenesisParams) *state.State {
		intent, st := test.NewIntent(t, l1ChainID, dk, l2ChainID, artifacts.DefaultL1ContractsLocator, artifacts.DefaultL2ContractsLocator)
		intent.L1DevGenesisParams = l1DevGenesisParams

		testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

		// Deploy using genesis strategy
		require.NoError(deployer.ApplyPipeline(
			ctx,
			deployer.ApplyPipelineOpts{
				DeploymentTarget:   deployer.DeploymentTargetGenesis,
				L1RPCUrl:           l1RPC,
				DeployerPrivateKey: pk,
				Intent:             intent,
				State:              st,
				Logger:             lgr,
				StateWriter:        pipeline.NoopStateWriter(),
				CacheDir:           testCacheDir,
			},
		))

		return st
	}

	t.Run("defaults", func(t *testing.T) {
		st := deployChain(nil)
		require.Greater(st.Chains[0].StartBlock.Time, l1GenesisParams.BlockParams.Timestamp)
		require.Nil(st.L1DevGenesis.Config.PragueTime)
	})

	t.Run("custom", func(t *testing.T) {
		st := deployChain(l1GenesisParams)
		require.EqualValues(l1GenesisParams.BlockParams.Timestamp, st.Chains[0].StartBlock.Time)
		require.EqualValues(l1GenesisParams.BlockParams.Timestamp, st.L1DevGenesis.Timestamp)

		require.EqualValues(l1GenesisParams.BlockParams.GasLimit, st.L1DevGenesis.GasLimit)
		require.NotNil(st.L1DevGenesis.ExcessBlobGas)
		require.EqualValues(l1GenesisParams.BlockParams.ExcessBlobGas, *st.L1DevGenesis.ExcessBlobGas)
		require.NotNil(st.L1DevGenesis.Config.PragueTime)
		expectedPragueTimestamp := l1GenesisParams.BlockParams.Timestamp + *l1GenesisParams.PragueTimeOffset
		require.EqualValues(expectedPragueTimestamp, *st.L1DevGenesis.Config.PragueTime)
	})
}
