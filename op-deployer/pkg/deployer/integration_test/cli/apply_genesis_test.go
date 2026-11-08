package cli

import (
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestCLIApplyGenesisMockSP1Verifier checks that a genesis apply through the CLI needs an SP1
// verifier, and that --deploy-mock-sp1-verifier supplies one. Tools that drive the CLI, such as
// devnet launchers, have no other way to give a generated genesis a verifier.
func TestCLIApplyGenesisMockSP1Verifier(t *testing.T) {
	t.Run("without the flag", func(t *testing.T) {
		runner, workDir := newGenesisIntentRunner(t)
		runner.ExpectErrorContains(t, genesisApplyArgs(workDir), nil, "sp1Verifier must be specified for genesis deployments")
	})

	t.Run("with the flag", func(t *testing.T) {
		runner, workDir := newGenesisIntentRunner(t)
		runner.ExpectSuccess(t, append(genesisApplyArgs(workDir), "--deploy-mock-sp1-verifier"), nil)

		st, err := pipeline.ReadState(workDir)
		require.NoError(t, err)
		require.NotNil(t, st.SP1Verifier)
		require.NotEqual(t, common.Address{}, *st.SP1Verifier)
	})
}

// newGenesisIntentRunner returns a runner whose workdir holds a fresh single-chain intent and
// state that use the local contract artifacts.
func newGenesisIntentRunner(t *testing.T) (*CLITestRunner, string) {
	runner := NewCLITestRunner(t)
	workDir := runner.GetWorkDir()

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	loc, _ := testutil.LocalArtifacts(t)
	intent, st := shared.NewIntent(t, big.NewInt(devnet.DefaultChainID), dk, uint256.NewInt(1), loc, loc, 60_000_000)
	require.NoError(t, intent.WriteToFile(filepath.Join(workDir, "intent.toml")))
	require.NoError(t, st.WriteToFile(filepath.Join(workDir, "state.json")))
	return runner, workDir
}

func genesisApplyArgs(workDir string) []string {
	return []string{"apply", "--deployment-target", "genesis", "--workdir", workDir}
}
