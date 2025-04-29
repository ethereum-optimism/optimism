package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewDeployMIPSScript(t *testing.T) {
	t.Run("should not fail with current version of DeployMIPS2 contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		// Then we load the script
		//
		// This would raise an error if the Go types didn't match the ABI
		deploySuperchain, err := NewDeployMIPSScript(host1)
		require.NoError(t, err)

		// Then we deploy
		output, err := deploySuperchain.Run(DeployMIPS2Input{
			PreimageOracle: common.Address{'P'},
			MipsVersion:    big.NewInt(6),
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)

		// Now we run the old deployer
		//
		// We run it on a fresh host so that the deployer nonces are the same
		// which in turn means we should get identical output
		host2 := createTestHost(t)
		deprecatedOutput, err := DeployMIPS(host2, DeployMIPSInput{
			PreimageOracle: common.Address{'P'},
			MipsVersion:    6,
		})

		// Make sure it succeeded
		require.NoError(t, err)
		require.NotNil(t, deprecatedOutput)

		// Now make sure the addresses are the same
		require.Equal(t, deprecatedOutput.MipsSingleton, output.MipsSingleton)

		// And just to be super sure we also compare the code deployed to the addresses
		require.Equal(t, host2.GetCode(deprecatedOutput.MipsSingleton), host1.GetCode(output.MipsSingleton))
	})
}
