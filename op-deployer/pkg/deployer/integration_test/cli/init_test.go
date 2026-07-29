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

	// Standard intent must have zero PAOs - user must specify manually
	require.Equal(t, common.Address{}, intent.Chains[0].Roles.L1ProxyAdminOwner, "L1ProxyAdminOwner must be zero by default")
	require.Equal(t, common.Address{}, intent.Chains[0].Roles.L2ProxyAdminOwner, "L2ProxyAdminOwner must be zero by default")

	// Verify state.json was created (chains get populated during apply, not init)
	st, err := pipeline.ReadState(workDir)
	require.NoError(t, err)
	// State starts empty and gets populated during apply
	require.Len(t, st.Chains, 0)
}

// TestCLIInitStandardOverrides_producesZeroPAOs ensures standard-overrides intent type
// also produces zero PAOs - user must specify manually.
func TestCLIInitStandardOverrides_producesZeroPAOs(t *testing.T) {
	runner := NewCLITestRunner(t)
	workDir := runner.GetWorkDir()

	runner.ExpectSuccess(t, []string{
		"init",
		"--l1-chain-id", "11155111",
		"--l2-chain-ids", "1",
		"--intent-type", string(state.IntentTypeStandardOverrides),
		"--workdir", workDir,
	}, nil)

	intent, err := pipeline.ReadIntent(workDir)
	require.NoError(t, err)
	require.Equal(t, state.IntentTypeStandardOverrides, intent.ConfigType)
	require.Equal(t, common.Address{}, intent.Chains[0].Roles.L1ProxyAdminOwner, "L1ProxyAdminOwner must be zero by default")
	require.Equal(t, common.Address{}, intent.Chains[0].Roles.L2ProxyAdminOwner, "L2ProxyAdminOwner must be zero by default")
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

	// All chains must have zero PAOs
	for i, chain := range intent.Chains {
		require.Equal(t, common.Address{}, chain.Roles.L1ProxyAdminOwner, "chain %d L1ProxyAdminOwner must be zero", i)
		require.Equal(t, common.Address{}, chain.Roles.L2ProxyAdminOwner, "chain %d L2ProxyAdminOwner must be zero", i)
	}

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
