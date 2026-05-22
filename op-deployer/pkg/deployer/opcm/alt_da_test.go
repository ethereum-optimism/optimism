package opcm

import (
	"math/big"
	"testing"

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
		deploySuperchain, err := NewDeployAltDAScript(host1)
		require.NoError(t, err)

		// Then we deploy
		output, err := deploySuperchain.Run(DeployAltDAInput{
			"salt":                     common.BigToHash(big.NewInt(1)),
			"proxyAdmin":               common.BigToAddress(big.NewInt(2)),
			"challengeContractOwner":   common.BigToAddress(big.NewInt(3)),
			"challengeWindow":          big.NewInt(4),
			"resolveWindow":            big.NewInt(5),
			"bondSize":                 big.NewInt(6),
			"resolverRefundPercentage": big.NewInt(7),
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)
	})
}
