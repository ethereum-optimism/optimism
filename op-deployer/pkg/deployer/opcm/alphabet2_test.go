package opcm_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewDeployAlphabetVMScript(t *testing.T) {
	t.Run("should not fail with current version of DeployAlphabetVM2 contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		deployAlphabetVM, err := opcm.NewDeployAlphabetVMScript(host1)
		require.NoError(t, err)

		// Now we run the deploy script
		output, err := deployAlphabetVM.Run(opcm.DeployAlphabetVM2Input{
			AbsolutePrestate: common.BigToHash(big.NewInt(1)),
			PreimageOracle:   common.BigToAddress(big.NewInt(2)),
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)

		// Now we run the old deployer
		//
		// We run it on a fresh host so that the deployer nonces are the same
		// which in turn means we should get identical output
		host2 := createTestHost(t)

		deprecatedOutput, err := opcm.DeployAlphabetVM(host2, opcm.DeployAlphabetVMInput{
			AbsolutePrestate: common.BigToHash(big.NewInt(1)),
			PreimageOracle:   common.BigToAddress(big.NewInt(2)),
		})

		// Make sure it succeeded
		require.NoError(t, err)
		require.NotNil(t, deprecatedOutput)

		// Now make sure the addresses are the same
		require.Equal(t, deprecatedOutput.AlphabetVM, output.AlphabetVM)

		// And just to be super sure we also compare the code deployed to the addresses
		require.Equal(t, host2.GetCode(deprecatedOutput.AlphabetVM), host1.GetCode(output.AlphabetVM))
	})
}
