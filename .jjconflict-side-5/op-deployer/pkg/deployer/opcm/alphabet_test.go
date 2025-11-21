package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewDeployAlphabetVMScript(t *testing.T) {
	t.Run("should not fail with current version of DeployAlphabetVM contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		deployAlphabetVM, err := NewDeployAlphabetVMScript(host1)
		require.NoError(t, err)

		// Now we run the deploy script
		output, err := deployAlphabetVM.Run(DeployAlphabetVMInput{
			AbsolutePrestate: common.BigToHash(big.NewInt(1)),
			PreimageOracle:   common.BigToAddress(big.NewInt(2)),
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)
	})
}
