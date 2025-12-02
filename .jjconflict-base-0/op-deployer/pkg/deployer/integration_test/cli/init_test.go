package cli

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestCLIInit tests basic init command and file creation
func TestCLIInit(t *testing.T) {
	runner := NewCLITestRunner(t)

	workDir := runner.GetWorkDir()

	runner.ExpectSuccess(t, []string{
		"init",
		"--l1-chain-id", "11155111",
		"--l2-chain-ids", "1",
		"--workdir", workDir,
	}, nil)

	// Verify intent.toml was created and has correct content
	intent, err := pipeline.ReadIntent(workDir)
	require.NoError(t, err)
	require.Equal(t, uint64(11155111), intent.L1ChainID)
	require.Len(t, intent.Chains, 1)
	require.Equal(t, common.Hash(uint256.NewInt(1).Bytes32()), intent.Chains[0].ID)

	// Verify state.json was created (chains get populated during apply, not init)
	st, err := pipeline.ReadState(workDir)
	require.NoError(t, err)
	// State starts empty and gets populated during apply
	require.Len(t, st.Chains, 0)
}

// TestCLIInitMultipleChains tests init with multiple L2 chain IDs
func TestCLIInitMultipleChains(t *testing.T) {
	runner := NewCLITestRunner(t)

	workDir := runner.GetWorkDir()

	runner.ExpectSuccess(t, []string{
		"init",
		"--l1-chain-id", "11155111",
		"--l2-chain-ids", "1,2",
		"--workdir", workDir,
	}, nil)

	intent, err := pipeline.ReadIntent(workDir)
	require.NoError(t, err)
	require.Equal(t, uint64(11155111), intent.L1ChainID)
	require.Len(t, intent.Chains, 2)
	require.Equal(t, common.Hash(uint256.NewInt(1).Bytes32()), intent.Chains[0].ID)
	require.Equal(t, common.Hash(uint256.NewInt(2).Bytes32()), intent.Chains[1].ID)

	// State starts empty and gets populated during apply
	st, err := pipeline.ReadState(workDir)
	require.NoError(t, err)
	require.Len(t, st.Chains, 0)
}

// TestCLIInitCustomIntentType tests init with custom intent type
func TestCLIInitCustomIntentType(t *testing.T) {
	runner := NewCLITestRunner(t)

	workDir := runner.GetWorkDir()

	runner.ExpectSuccess(t, []string{
		"init",
		"--l1-chain-id", "11155111",
		"--l2-chain-ids", "1",
		"--intent-type", "custom",
		"--workdir", workDir,
	}, nil)

	intent, err := pipeline.ReadIntent(workDir)
	require.NoError(t, err)
	require.Equal(t, state.IntentTypeCustom, intent.ConfigType)
	require.Equal(t, uint64(11155111), intent.L1ChainID)
	require.Len(t, intent.Chains, 1)

	// State starts empty and gets populated during apply
	st, err := pipeline.ReadState(workDir)
	require.NoError(t, err)
	require.Len(t, st.Chains, 0)
}
