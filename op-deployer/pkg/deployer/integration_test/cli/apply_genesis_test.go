package cli

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestCLIApplyGenesisMockSP1Verifier checks that a genesis apply through the CLI needs an SP1
// verifier, and that --deploy-mock-sp1-verifier puts one in the generated L1 genesis. Tools that
// drive the CLI, such as devnet launchers, have no other way to give a generated genesis a verifier.
func TestCLIApplyGenesisMockSP1Verifier(t *testing.T) {
	t.Run("without the flag", func(t *testing.T) {
		runner, args := newGenesisApply(t)
		runner.ExpectErrorContains(t, args, nil, "sp1Verifier must be specified for genesis deployments")
	})

	t.Run("with the flag", func(t *testing.T) {
		runner, args := newGenesisApply(t)
		runner.ExpectSuccess(t, append(args, "--deploy-mock-sp1-verifier"), nil)

		st, err := pipeline.ReadState(runner.GetWorkDir())
		require.NoError(t, err)
		require.NotNil(t, st.SP1Verifier)
		require.NotEqual(t, common.Address{}, *st.SP1Verifier)
		require.NotNil(t, st.L1StateDump)
		verifier, ok := st.L1StateDump.Data.Accounts[*st.SP1Verifier]
		require.True(t, ok, "mock SP1 verifier missing from the L1 genesis")
		require.NotEmpty(t, verifier.Code)
	})
}

// TestCLIApplyRejectsMockSP1VerifierForLive checks that the flag's value reaches ApplyPipeline,
// which rejects it before dialling the L1 for anything but a genesis deployment.
func TestCLIApplyRejectsMockSP1VerifierForLive(t *testing.T) {
	runner := NewCLITestRunner(t)
	workDir := runner.GetWorkDir()
	intent := &state.Intent{ConfigType: state.IntentTypeCustom, L1ChainID: 11155111}
	writePrepareWorkdir(t, workDir, intent, &state.State{Version: 1})
	pkHex, _, _ := shared.DefaultPrivkey(t)

	runner.ExpectErrorContains(t, []string{
		"apply",
		"--deployment-target", "live",
		"--workdir", workDir,
		"--l1-rpc-url", "http://127.0.0.1:1",
		"--private-key", pkHex,
		"--deploy-mock-sp1-verifier",
	}, nil, "mock SP1 verifier deployment is only supported for genesis")
}

// newGenesisApply returns a runner whose workdir holds a fresh single-chain intent and state
// that use the local contract artifacts, and the args for a keyed genesis apply of it.
func newGenesisApply(t *testing.T) (*CLITestRunner, []string) {
	runner := NewCLITestRunner(t)
	workDir := runner.GetWorkDir()

	pkHex, _, dk := shared.DefaultPrivkey(t)
	loc, _ := testutil.LocalArtifacts(t)
	intent, st := shared.NewIntent(t, big.NewInt(devnet.DefaultChainID), dk, uint256.NewInt(1), loc, loc, standard.GasLimit)
	writePrepareWorkdir(t, workDir, intent, st)

	return runner, []string{
		"apply",
		"--deployment-target", "genesis",
		"--workdir", workDir,
		"--private-key", pkHex,
	}
}
