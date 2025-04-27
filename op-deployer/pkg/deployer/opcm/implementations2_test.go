package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/addresses"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewDeployImplementationsScript(t *testing.T) {
	deployDependencies := func(host *script.Host) (proxyAdminAddress common.Address, proxyAddress common.Address, protocolVersionsAddress common.Address) {
		proxyAdminArtifact, err := host.Artifacts().ReadArtifact("ProxyAdmin.sol", "ProxyAdmin")
		require.NoError(t, err)

		encodedProxyAdmin, err := proxyAdminArtifact.ABI.Pack("", addresses.ScriptDeployer)
		require.NoError(t, err)

		proxyAdminAddress, err = host.Create(addresses.ScriptDeployer, append(proxyAdminArtifact.Bytecode.Object, encodedProxyAdmin...))
		require.NoError(t, err)

		// Then we get a proxy deployed
		proxyArtifact, err := host.Artifacts().ReadArtifact("Proxy.sol", "Proxy")
		require.NoError(t, err)

		encodedProxy, err := proxyArtifact.ABI.Pack("", proxyAdminAddress)
		require.NoError(t, err)

		proxyAddress, err = host.Create(addresses.ScriptDeployer, append(proxyArtifact.Bytecode.Object, encodedProxy...))
		require.NoError(t, err)

		// Then we get ProtocolVersions deployed
		protocolVersionsArtifact, err := host.Artifacts().ReadArtifact("ProtocolVersions.sol", "ProtocolVersions")
		require.NoError(t, err)

		encodedProtocolVersions, err := protocolVersionsArtifact.ABI.Pack("")
		require.NoError(t, err)

		protocolVersionsAddress, err = host.Create(addresses.ScriptDeployer, append(protocolVersionsArtifact.Bytecode.Object, encodedProtocolVersions...))
		require.NoError(t, err)

		return proxyAdminAddress, proxyAddress, protocolVersionsAddress
	}

	t.Run("should not fail with current version of DeployImplementations2 contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		// We'll need some contracts already deployed for this to work
		proxyAdminAddress, proxyAddress, protocolVersionsAddress := deployDependencies(host1)

		deployImplementations, err := NewDeployImplementationsScript(host1)
		require.NoError(t, err)

		// Now we run the deploy script
		output, err := deployImplementations.Run(DeployImplementations2Input{
			WithdrawalDelaySeconds:          big.NewInt(1),
			MinProposalSizeBytes:            big.NewInt(2),
			ChallengePeriodSeconds:          big.NewInt(3),
			ProofMaturityDelaySeconds:       big.NewInt(4),
			DisputeGameFinalityDelaySeconds: big.NewInt(5),
			MipsVersion:                     big.NewInt(1),
			// Release version to set OPCM implementations for, of the format `op-contracts/vX.Y.Z`.
			L1ContractsRelease:    "dev-release",
			SuperchainConfigProxy: proxyAddress,
			ProtocolVersionsProxy: protocolVersionsAddress,
			SuperchainProxyAdmin:  proxyAdminAddress,
			UpgradeController:     common.BigToAddress(big.NewInt(13)),
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)

		// Now we run the old deployer
		//
		// We run it on a fresh host so that the deployer nonces are the same
		// which in turn means we should get identical output
		host2 := createTestHost(t)

		// We'll need some contracts already deployed for this to work
		proxyAdminAddress, proxyAddress, protocolVersionsAddress = deployDependencies(host2)

		deprecatedOutput, err := DeployImplementations(host2, DeployImplementationsInput{
			WithdrawalDelaySeconds:          big.NewInt(1),
			MinProposalSizeBytes:            big.NewInt(2),
			ChallengePeriodSeconds:          big.NewInt(3),
			ProofMaturityDelaySeconds:       big.NewInt(4),
			DisputeGameFinalityDelaySeconds: big.NewInt(5),
			MipsVersion:                     big.NewInt(1),
			// Release version to set OPCM implementations for, of the format `op-contracts/vX.Y.Z`.
			L1ContractsRelease:    "dev-release",
			SuperchainConfigProxy: proxyAddress,
			ProtocolVersionsProxy: protocolVersionsAddress,
			SuperchainProxyAdmin:  proxyAdminAddress,
			UpgradeController:     common.BigToAddress(big.NewInt(13)),
		})

		// Make sure it succeeded
		require.NoError(t, err)
		require.NotNil(t, deprecatedOutput)

		// Now make sure the addresses are the same
		require.Equal(t, deprecatedOutput.AnchorStateRegistryImpl, output.AnchorStateRegistryImpl)
		require.Equal(t, deprecatedOutput.DelayedWETHImpl, output.DelayedWETHImpl)
		require.Equal(t, deprecatedOutput.DisputeGameFactoryImpl, output.DisputeGameFactoryImpl)
		require.Equal(t, deprecatedOutput.ETHLockboxImpl, output.ETHLockboxImpl)
		require.Equal(t, deprecatedOutput.L1CrossDomainMessengerImpl, output.L1CrossDomainMessengerImpl)
		require.Equal(t, deprecatedOutput.L1ERC721BridgeImpl, output.L1ERC721BridgeImpl)
		require.Equal(t, deprecatedOutput.L1StandardBridgeImpl, output.L1StandardBridgeImpl)
		require.Equal(t, deprecatedOutput.MipsSingleton, output.MipsSingleton)
		require.Equal(t, deprecatedOutput.Opcm, output.Opcm)
		require.Equal(t, deprecatedOutput.OpcmContractsContainer, output.OpcmContractsContainer)
		require.Equal(t, deprecatedOutput.OpcmDeployer, output.OpcmDeployer)
		require.Equal(t, deprecatedOutput.OpcmGameTypeAdder, output.OpcmGameTypeAdder)
		require.Equal(t, deprecatedOutput.OpcmInteropMigrator, output.OpcmInteropMigrator)
		require.Equal(t, deprecatedOutput.OpcmUpgrader, output.OpcmUpgrader)
		require.Equal(t, deprecatedOutput.OptimismMintableERC20FactoryImpl, output.OptimismMintableERC20FactoryImpl)
		require.Equal(t, deprecatedOutput.OptimismPortalImpl, output.OptimismPortalImpl)
		require.Equal(t, deprecatedOutput.PreimageOracleSingleton, output.PreimageOracleSingleton)
		require.Equal(t, deprecatedOutput.ProtocolVersionsImpl, output.ProtocolVersionsImpl)
		require.Equal(t, deprecatedOutput.SuperchainConfigImpl, output.SuperchainConfigImpl)
		require.Equal(t, deprecatedOutput.SystemConfigImpl, output.SystemConfigImpl)

		// And just to be super sure we also compare the code deployed to the addresses
		require.Equal(t, host2.GetCode(deprecatedOutput.AnchorStateRegistryImpl), host1.GetCode(output.AnchorStateRegistryImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.DelayedWETHImpl), host1.GetCode(output.DelayedWETHImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.DisputeGameFactoryImpl), host1.GetCode(output.DisputeGameFactoryImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.ETHLockboxImpl), host1.GetCode(output.ETHLockboxImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.L1CrossDomainMessengerImpl), host1.GetCode(output.L1CrossDomainMessengerImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.L1ERC721BridgeImpl), host1.GetCode(output.L1ERC721BridgeImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.L1StandardBridgeImpl), host1.GetCode(output.L1StandardBridgeImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.MipsSingleton), host1.GetCode(output.MipsSingleton))
		require.Equal(t, host2.GetCode(deprecatedOutput.Opcm), host1.GetCode(output.Opcm))
		require.Equal(t, host2.GetCode(deprecatedOutput.OpcmContractsContainer), host1.GetCode(output.OpcmContractsContainer))
		require.Equal(t, host2.GetCode(deprecatedOutput.OpcmDeployer), host1.GetCode(output.OpcmDeployer))
		require.Equal(t, host2.GetCode(deprecatedOutput.OpcmGameTypeAdder), host1.GetCode(output.OpcmGameTypeAdder))
		require.Equal(t, host2.GetCode(deprecatedOutput.OpcmInteropMigrator), host1.GetCode(output.OpcmInteropMigrator))
		require.Equal(t, host2.GetCode(deprecatedOutput.OpcmUpgrader), host1.GetCode(output.OpcmUpgrader))
		require.Equal(t, host2.GetCode(deprecatedOutput.OptimismMintableERC20FactoryImpl), host1.GetCode(output.OptimismMintableERC20FactoryImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.OptimismPortalImpl), host1.GetCode(output.OptimismPortalImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.PreimageOracleSingleton), host1.GetCode(output.PreimageOracleSingleton))
		require.Equal(t, host2.GetCode(deprecatedOutput.ProtocolVersionsImpl), host1.GetCode(output.ProtocolVersionsImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.SuperchainConfigImpl), host1.GetCode(output.SuperchainConfigImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.SystemConfigImpl), host1.GetCode(output.SystemConfigImpl))
	})
}
