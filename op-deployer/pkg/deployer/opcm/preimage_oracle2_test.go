package opcm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDeployPreimageOracleScript(t *testing.T) {
	t.Run("should not fail with current version of DeployPreimageOracle2 contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		// Then we load the script
		//
		// This would raise an error if the Go types didn't match the ABI
		deployPreimageOracle, err := NewDeployPreimageOracleScript(host1)
		require.NoError(t, err)

		// Then we deploy
		output, err := deployPreimageOracle.Run(DeployPreimageOracle2Input{
			MinProposalSize: big.NewInt(1),
			ChallengePeriod: big.NewInt(2),
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)

		// Now we run the old deployer
		//
		// We run it on a fresh host so that the deployer nonces are the same
		// which in turn means we should get identical output
		host2 := createTestHost(t)
		deprecatedOutput, err := DeployPreimageOracle(host2, DeployPreimageOracleInput{
			MinProposalSize: big.NewInt(1),
			ChallengePeriod: big.NewInt(2),
		})

		// Make sure it succeeded
		require.NoError(t, err)
		require.NotNil(t, deprecatedOutput)

		// Now make sure the addresses are the same
		require.Equal(t, deprecatedOutput.PreimageOracle, output.PreimageOracle)

		// And just to be super sure we also compare the code deployed to the addresses
		require.Equal(t, host2.GetCode(deprecatedOutput.PreimageOracle), host1.GetCode(output.PreimageOracle))
	})
}
