package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewDeployImplementationsScript(t *testing.T) {
	deployDependencies := func(host *script.Host) (proxyAdminAddress common.Address, proxyAddress common.Address) {
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

		return proxyAdminAddress, proxyAddress
	}

	t.Run("should not fail with current version of DeployImplementations contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		// We'll need some contracts already deployed for this to work
		proxyAdminAddress, proxyAddress := deployDependencies(host1)

		deployImplementations, err := NewDeployImplementationsScript(host1)
		require.NoError(t, err)

		// Now we run the deploy script
		mipsVersion := int64(standard.MIPSVersion)
		output, err := deployImplementations.Run(DeployImplementationsInput{
			"withdrawalDelaySeconds":          big.NewInt(1),
			"minProposalSizeBytes":            big.NewInt(2),
			"challengePeriodSeconds":          big.NewInt(3),
			"proofMaturityDelaySeconds":       big.NewInt(4),
			"disputeGameFinalityDelaySeconds": big.NewInt(5),
			"mipsVersion":                     big.NewInt(mipsVersion),
			"devFeatureBitmap":                common.Hash{},
			"faultGameV2MaxGameDepth":         big.NewInt(73),
			"faultGameV2SplitDepth":           big.NewInt(30),
			"faultGameV2ClockExtension":       big.NewInt(10800),
			"faultGameV2MaxClockDuration":     big.NewInt(302400),
			"superchainConfigProxy":           proxyAddress,
			"superchainProxyAdmin":            proxyAdminAddress,
			"l1ProxyAdminOwner":               common.BigToAddress(big.NewInt(13)),
			"challenger":                      common.BigToAddress(big.NewInt(14)),
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)
	})
}
