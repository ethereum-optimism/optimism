package opcm_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewDelayedWETHScript(t *testing.T) {
	t.Run("should not fail with current version of DelayedWETH2 contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		// Then we load the script
		//
		// This would raise an error if the Go types didn't match the ABI
		deploySuperchain, err := opcm.NewDelayedWETHScript(host1)
		require.NoError(t, err)

		// Then we deploy
		output, err := deploySuperchain.Run(opcm.DelayedWETH2Input{
			Release:               "dev",
			ProxyAdmin:            common.Address{'P'},
			SuperchainConfigProxy: common.Address{'S'},
			DelayedWethImpl:       common.Address{},
			DelayedWethOwner:      common.Address{'O'},
			DelayedWethDelay:      big.NewInt(100),
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)

		// Now we run the old deployer
		//
		// We run it on a fresh host so that the deployer nonces are the same
		// which in turn means we should get identical output
		host2 := createTestHost(t)
		deprecatedOutput, err := opcm.DeployDelayedWETH(host2, opcm.DeployDelayedWETHInput{
			Release:               "dev",
			ProxyAdmin:            common.Address{'P'},
			SuperchainConfigProxy: common.Address{'S'},
			DelayedWethImpl:       common.Address{},
			DelayedWethOwner:      common.Address{'O'},
			DelayedWethDelay:      big.NewInt(100),
		})

		// Make sure it succeeded
		require.NoError(t, err)
		require.NotNil(t, deprecatedOutput)

		// Now make sure the addresses are the same
		require.Equal(t, deprecatedOutput.DelayedWethImpl, output.DelayedWethImpl)
		require.Equal(t, deprecatedOutput.DelayedWethProxy, output.DelayedWethProxy)

		// And just to be super sure we also compare the code deployed to the addresses
		require.Equal(t, host2.GetCode(deprecatedOutput.DelayedWethImpl), host1.GetCode(output.DelayedWethImpl))
		require.Equal(t, host2.GetCode(deprecatedOutput.DelayedWethProxy), host1.GetCode(output.DelayedWethProxy))
	})
}
