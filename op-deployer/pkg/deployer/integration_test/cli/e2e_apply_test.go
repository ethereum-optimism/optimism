package cli

import (
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestCLIEndToEndApply tests the full end-to-end apply workflow via CLI
func TestCLIEndToEndApply(t *testing.T) {
	runner := NewCLITestRunnerWithNetwork(t)

	workDir := runner.GetWorkDir()
	// Use the same chain ID that anvil runs on
	l1ChainID := uint64(devnet.DefaultChainID)
	l2ChainID1 := uint256.NewInt(1)
	l2ChainID2 := uint256.NewInt(2)

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	t.Run("two chains one after another", func(t *testing.T) {
		intent, _ := cliInitIntent(t, runner, l1ChainID, []common.Hash{l2ChainID1.Bytes32()})

		if intent.SuperchainRoles == nil {
			t.Log("SuperchainRoles is nil, initializing...")
			intent.SuperchainRoles = &addresses.SuperchainRoles{}
		}

		l1ChainIDBig := big.NewInt(int64(l1ChainID))
		intent.SuperchainRoles.SuperchainProxyAdminOwner = shared.AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainIDBig))
		intent.SuperchainRoles.SuperchainGuardian = shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
		intent.SuperchainRoles.ProtocolVersionsOwner = shared.AddrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainIDBig))
		intent.SuperchainRoles.Challenger = shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

		for _, chain := range intent.Chains {
			chain.Roles.L1ProxyAdminOwner = shared.AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainIDBig))
			chain.Roles.L2ProxyAdminOwner = shared.AddrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainIDBig))
			chain.Roles.SystemConfigOwner = shared.AddrFor(t, dk, devkeys.SystemConfigOwner.Key(l1ChainIDBig))
			chain.Roles.UnsafeBlockSigner = shared.AddrFor(t, dk, devkeys.SequencerP2PRole.Key(l1ChainIDBig))
			chain.Roles.Batcher = shared.AddrFor(t, dk, devkeys.BatcherRole.Key(l1ChainIDBig))
			chain.Roles.Proposer = shared.AddrFor(t, dk, devkeys.ProposerRole.Key(l1ChainIDBig))
			chain.Roles.Challenger = shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

			chain.BaseFeeVaultRecipient = shared.AddrFor(t, dk, devkeys.BaseFeeVaultRecipientRole.Key(l1ChainIDBig))
			chain.L1FeeVaultRecipient = shared.AddrFor(t, dk, devkeys.L1FeeVaultRecipientRole.Key(l1ChainIDBig))
			chain.SequencerFeeVaultRecipient = shared.AddrFor(t, dk, devkeys.SequencerFeeVaultRecipientRole.Key(l1ChainIDBig))

			chain.Eip1559DenominatorCanyon = standard.Eip1559DenominatorCanyon
			chain.Eip1559Denominator = standard.Eip1559Denominator
			chain.Eip1559Elasticity = standard.Eip1559Elasticity
		}
		require.NoError(t, intent.WriteToFile(filepath.Join(workDir, "intent.toml")))

		// Apply first chain with live deployment
		runner.ExpectSuccessWithNetwork(t, []string{
			"apply",
			"--deployment-target", "live",
			"--workdir", workDir,
		}, nil)

		// Add second chain to intent
		intent, err := pipeline.ReadIntent(workDir)
		require.NoError(t, err)

		secondChain := shared.NewChainIntent(t, dk, new(big.Int).SetUint64(l1ChainID), l2ChainID2, 60_000_000)
		secondChain.Eip1559DenominatorCanyon = standard.Eip1559DenominatorCanyon
		secondChain.Eip1559Denominator = standard.Eip1559Denominator
		secondChain.Eip1559Elasticity = standard.Eip1559Elasticity
		intent.Chains = append(intent.Chains, secondChain)

		require.NoError(t, intent.WriteToFile(filepath.Join(workDir, "intent.toml")))

		// Apply again with both chains
		runner.ExpectSuccessWithNetwork(t, []string{
			"apply",
			"--deployment-target", "live",
			"--workdir", workDir,
		}, nil)

		// Verify final state
		finalState, err := pipeline.ReadState(workDir)
		require.NoError(t, err)
		require.Len(t, finalState.Chains, 2)
		require.Equal(t, common.Hash(l2ChainID1.Bytes32()), finalState.Chains[0].ID)
		require.Equal(t, common.Hash(l2ChainID2.Bytes32()), finalState.Chains[1].ID)

		require.NotNil(t, finalState.AppliedIntent)
	})
}
