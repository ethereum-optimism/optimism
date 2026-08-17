package cli

import (
	"context"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestCLIPrepareCommitsSuperchainDeployment applies a superchain live, then prepares another chain
// against the OPCM that apply deployed.
func TestCLIPrepareCommitsSuperchainDeployment(t *testing.T) {
	runner := NewCLITestRunnerWithNetwork(t)

	applyDir := runner.GetWorkDir()
	l1ChainID := uint64(devnet.DefaultChainID)
	l1ChainIDBig := new(big.Int).SetUint64(l1ChainID)

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	// init supplies the artifact locators. prepare must predict against the same ones apply
	// deployed the OPCM from.
	initIntent, _ := cliInitIntent(t, runner, l1ChainID, []common.Hash{uint256.NewInt(1).Bytes32()})
	l1Loc, l2Loc := initIntent.L1ContractsLocator, initIntent.L2ContractsLocator

	applyIntent, applyState := shared.NewIntent(t, l1ChainIDBig, dk, uint256.NewInt(1), l1Loc, l2Loc, standard.GasLimit)
	writePrepareWorkdir(t, applyDir, applyIntent, applyState)

	runner.ExpectSuccessWithNetwork(t, []string{
		"apply",
		"--deployment-target", "live",
		"--workdir", applyDir,
	}, nil)

	applied, err := pipeline.ReadState(applyDir)
	require.NoError(t, err)
	require.NotNil(t, applied.SuperchainDeployment)
	require.NotNil(t, applied.SuperchainRoles)
	require.NotNil(t, applied.ImplementationsDeployment)
	opcmAddr := applied.ImplementationsDeployment.OpcmV2Impl
	require.NotEqual(t, common.Address{}, opcmAddr)

	newWorkdir := func(t *testing.T, superchainConfigProxy common.Address) string {
		workdir := t.TempDir()
		intent, st := shared.NewIntent(t, l1ChainIDBig, dk, uint256.NewInt(2), l1Loc, l2Loc, standard.GasLimit)
		intent.OPCMAddress = &opcmAddr
		intent.SuperchainConfigProxy = &superchainConfigProxy
		// The intent may not carry superchain roles alongside a pinned OPCM; they are read
		// off the chain instead.
		intent.SuperchainRoles = nil
		writePrepareWorkdir(t, workdir, intent, st)
		return workdir
	}

	t.Run("commits the superchain deployment", func(t *testing.T) {
		workdir := newWorkdir(t, applied.SuperchainDeployment.SuperchainConfigProxy)

		runner.ExpectSuccessWithNetwork(t, []string{"prepare", "--workdir", workdir}, nil)

		prepared, err := pipeline.ReadState(workdir)
		require.NoError(t, err)
		require.NotNil(t, prepared.PreparedDeployment)

		// The read must reproduce what apply recorded when it deployed the superchain.
		require.NotNil(t, prepared.SuperchainDeployment)
		require.Equal(t, *applied.SuperchainDeployment, *prepared.SuperchainDeployment)

		// prepare must record implementations the pinned OPCM installs.
		require.NotNil(t, prepared.ImplementationsDeployment)
		require.Equal(t, *applied.ImplementationsDeployment, *prepared.ImplementationsDeployment)

		// The committed proxy must match the frozen intent the continuation deploys from.
		require.NotNil(t, prepared.PreparedDeployment)
		require.NotNil(t, prepared.PreparedDeployment.Intent.SuperchainConfigProxy)
		require.Equal(
			t,
			*prepared.PreparedDeployment.Intent.SuperchainConfigProxy,
			prepared.SuperchainDeployment.SuperchainConfigProxy,
		)

		// Only the roles the superchain exposes on chain are readable, which excludes Challenger.
		require.NotNil(t, prepared.SuperchainRoles)
		require.Equal(
			t,
			applied.SuperchainRoles.SuperchainProxyAdminOwner,
			prepared.SuperchainRoles.SuperchainProxyAdminOwner,
		)
		require.Equal(t, applied.SuperchainRoles.SuperchainGuardian, prepared.SuperchainRoles.SuperchainGuardian)
		require.Equal(t, common.Address{}, prepared.SuperchainRoles.Challenger)
	})

	t.Run("writes no state when the superchain cannot be read", func(t *testing.T) {
		workdir := newWorkdir(t, common.Address{'n', 'o', 'c', 'o', 'd', 'e'})

		_, err := runner.RunWithNetwork(context.Background(), []string{"prepare", "--workdir", workdir}, nil)
		require.ErrorContains(t, err, "superchainConfigProxy has no code")

		unwritten, err := pipeline.ReadState(workdir)
		require.NoError(t, err)
		require.Nil(t, unwritten.SuperchainDeployment)
		require.Nil(t, unwritten.SuperchainRoles)
		require.Nil(t, unwritten.PreparedDeployment)
	})
}

func writePrepareWorkdir(t *testing.T, workdir string, intent *state.Intent, st *state.State) {
	t.Helper()
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
	require.NoError(t, st.WriteToFile(filepath.Join(workdir, "state.json")))
}
