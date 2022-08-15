package immutables_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/state-surgery/immutables"
	"github.com/stretchr/testify/require"
)

func TestBuildOptimism(t *testing.T) {
	results, err := immutables.OptimismBuild()
	require.Nil(t, err)
	require.NotNil(t, results)

	// Only the exact contracts that we care about are being
	// modified
	require.Equal(t, len(results), 6)

	contracts := map[string]bool{
		"GasPriceOracle":         true,
		"L1Block":                true,
		"L2CrossDomainMessenger": true,
		"L2StandardBridge":       true,
		"L2ToL1MessagePasser":    true,
		"SequencerFeeVault":      true,
	}

	for name, bytecode := range results {
		// There is bytecode there
		require.Greater(t, len(bytecode), 0)
		// It is in the set of contracts that we care about
		require.True(t, contracts[name])
	}
}
