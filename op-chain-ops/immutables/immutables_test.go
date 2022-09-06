package immutables_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/immutables"
	"github.com/stretchr/testify/require"
)

func TestBuildOptimism(t *testing.T) {
	results, err := immutables.BuildOptimism()
	require.Nil(t, err)
	require.NotNil(t, results)

	contracts := map[string]bool{
		"GasPriceOracle":               true,
		"L1Block":                      true,
		"L2CrossDomainMessenger":       true,
		"L2StandardBridge":             true,
		"L2ToL1MessagePasser":          true,
		"SequencerFeeVault":            true,
		"OptimismMintableERC20Factory": true,
	}

	// Only the exact contracts that we care about are being
	// modified
	require.Equal(t, len(results), len(contracts))

	for name, bytecode := range results {
		// There is bytecode there
		require.Greater(t, len(bytecode), 0)
		// It is in the set of contracts that we care about
		require.True(t, contracts[name])
	}
}
