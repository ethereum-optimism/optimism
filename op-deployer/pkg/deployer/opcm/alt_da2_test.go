package opcm_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewDeployAltDAScript(t *testing.T) {
	t.Run("should not fail with current version of DeployAltDA contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		// Then we load the script
		//
		// This would raise an error if the Go types didn't match the ABI
		deploySuperchain, err := opcm.NewDeployAltDAScript(host1)
		require.NoError(t, err)

		// Then we deploy
		output, err := deploySuperchain.Run(opcm.DeployAltDA2Input{
			Salt:                     common.BigToHash(big.NewInt(1)),
			ProxyAdmin:               common.BigToAddress(big.NewInt(2)),
			ChallengeContractOwner:   common.BigToAddress(big.NewInt(3)),
			ChallengeWindow:          big.NewInt(4),
			ResolveWindow:            big.NewInt(5),
			BondSize:                 big.NewInt(6),
			ResolverRefundPercentage: big.NewInt(7),
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)

		// Now we run the old deployer
		//
		// We run it on a fresh host so that the deployer nonces are the same
		// which in turn means we should get identical output
		host2 := createTestHost(t)
		deprecatedOutput, err := opcm.DeployAltDA(host2, opcm.DeployAltDAInput{
			Salt:                     common.BigToHash(big.NewInt(1)),
			ProxyAdmin:               common.BigToAddress(big.NewInt(2)),
			ChallengeContractOwner:   common.BigToAddress(big.NewInt(3)),
			ChallengeWindow:          big.NewInt(4),
			ResolveWindow:            big.NewInt(5),
			BondSize:                 big.NewInt(6),
			ResolverRefundPercentage: big.NewInt(7),
		})

		// Make sure it succeeded
		require.NoError(t, err)
		require.NotNil(t, deprecatedOutput)

		// Now make sure the addresses are the same
		require.Equal(t, deprecatedOutput.DataAvailabilityChallengeImpl, output.DataAvailabilityChallengeImpl)
		require.Equal(t, deprecatedOutput.DataAvailabilityChallengeProxy, output.DataAvailabilityChallengeProxy)

		// And just to be super sure we also compare the code deployed to the addresses
		require.Equal(t, host2.GetCode(deprecatedOutput.DataAvailabilityChallengeImpl), host1.GetCode(output.DataAvailabilityChallengeImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.DataAvailabilityChallengeProxy), host1.GetCode(output.DataAvailabilityChallengeProxy))
	})
}
