package opcm_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestNewDeploySuperchainScript(t *testing.T) {
	t.Run("should not fail with current version of DeploySuperchain2 contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		// Then we load the script
		//
		// This would raise an error if the Go types didn't match the ABI
		deploySuperchain, err := opcm.NewDeploySuperchainScript(host1)
		require.NoError(t, err)

		// Then we deploy
		output, err := deploySuperchain.Run(opcm.DeploySuperchain2Input{
			Guardian:                   common.BigToAddress(big.NewInt(1)),
			ProtocolVersionsOwner:      common.BigToAddress(big.NewInt(2)),
			SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
			Paused:                     true,
			RecommendedProtocolVersion: params.ProtocolVersion{1},
			RequiredProtocolVersion:    params.ProtocolVersion{2},
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)

		// Now we run the old deployer
		//
		// We run it on a fresh host so that the deployer nonces are the same
		// which in turn means we should get identical output
		host2 := createTestHost(t)
		deprecatedOutput, err := opcm.DeploySuperchain(host2, opcm.DeploySuperchainInput{
			Guardian:                   common.BigToAddress(big.NewInt(1)),
			ProtocolVersionsOwner:      common.BigToAddress(big.NewInt(2)),
			SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
			Paused:                     true,
			RecommendedProtocolVersion: params.ProtocolVersion{1},
			RequiredProtocolVersion:    params.ProtocolVersion{2},
		})

		// Make sure it succeeded
		require.NoError(t, err)
		require.NotNil(t, deprecatedOutput)

		// Now make sure the addresses are the same
		require.Equal(t, deprecatedOutput.ProtocolVersionsImpl, output.ProtocolVersionsImpl)
		require.Equal(t, deprecatedOutput.SuperchainConfigImpl, output.SuperchainConfigImpl)
		require.Equal(t, deprecatedOutput.ProtocolVersionsProxy, output.ProtocolVersionsProxy)
		require.Equal(t, deprecatedOutput.SuperchainConfigProxy, output.SuperchainConfigProxy)
		require.Equal(t, deprecatedOutput.SuperchainProxyAdmin, output.SuperchainProxyAdmin)

		// And just to be super sure we also compare the code deployed to the addresses
		require.Equal(t, host2.GetCode(deprecatedOutput.ProtocolVersionsImpl), host1.GetCode(output.ProtocolVersionsImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.SuperchainConfigImpl), host1.GetCode(output.SuperchainConfigImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.SuperchainConfigProxy), host1.GetCode(output.SuperchainConfigProxy))
		require.Equal(t, host2.GetCode(deprecatedOutput.ProtocolVersionsProxy), host1.GetCode(output.ProtocolVersionsProxy))
		require.Equal(t, host2.GetCode(deprecatedOutput.SuperchainProxyAdmin), host1.GetCode(output.SuperchainProxyAdmin))
	})
}
