package cli

import (
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestCLIApplyNoOp tests apply with noop target
func TestCLIApplyNoOp(t *testing.T) {
	runner := NewCLITestRunnerWithNetwork(t)

	workDir := runner.GetWorkDir()

	intent, _ := cliInitIntent(t, runner, devnet.DefaultChainID, []common.Hash{uint256.NewInt(1).Bytes32()})

	if intent.SuperchainRoles == nil {
		t.Log("SuperchainRoles is nil, initializing...")
		intent.SuperchainRoles = &addresses.SuperchainRoles{}
	}

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	l1ChainIDBig := big.NewInt(devnet.DefaultChainID)
	intent.SuperchainRoles.SuperchainProxyAdminOwner = shared.AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainIDBig))
	intent.SuperchainRoles.SuperchainGuardian = shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
	intent.SuperchainRoles.ProtocolVersionsOwner = shared.AddrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainIDBig))
	intent.SuperchainRoles.Challenger = shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

	// Set chain-specific addresses
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

	runner.ExpectSuccessWithNetwork(t, []string{
		"apply",
		"--deployment-target", "noop",
		"--workdir", workDir,
	}, nil)
}
