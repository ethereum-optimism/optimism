package superchain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetChain(t *testing.T) {
	t.Run("OP Mainnet found", func(t *testing.T) {
		chain, err := GetChain(10)
		require.NoError(t, err)
		require.NotNil(t, chain)
	})

	// Celo mainnet skipped due to custom genesis
	t.Run("Celo Mainnet skipped", func(t *testing.T) {
		chain, err := GetChain(42220)
		require.Error(t, err)
		require.Nil(t, chain)
	})
}

func TestGetDepset(t *testing.T) {
	// Save BuiltInConfigs to restore later
	originalConfigs := BuiltInConfigs
	t.Cleanup(func() {
		BuiltInConfigs = originalConfigs
	})

	t.Run("unknown chainID", func(t *testing.T) {
		BuiltInConfigs = &ChainConfigLoader{
			Chains: map[uint64]*Chain{},
		}

		depset, err := GetDepset(999999)
		require.Nil(t, depset)
		require.ErrorIs(t, err, ErrUnknownChain)
		require.Contains(t, err.Error(), "unknown chain ID")
	})

	t.Run("nil InteropTime", func(t *testing.T) {
		mockChain := &Chain{
			Name:    "test",
			Network: "test",
			config: &ChainConfig{
				ChainID: 42,
				Hardforks: HardforkConfig{
					InteropTime: nil,
				},
			},
		}

		// Set configOnce as already done
		mockChain.configOnce.Do(func() {})

		// Replace chains map with our test chain
		BuiltInConfigs = &ChainConfigLoader{
			Chains: map[uint64]*Chain{42: mockChain},
		}

		depset, err := GetDepset(42)
		require.NoError(t, err)
		require.NotNil(t, depset)

		// Verify the default dependency was created
		selfDep, exists := depset["42"]
		require.True(t, exists)
		require.Equal(t, selfDep, Dependency{})
	})

	t.Run("nil Interop creates default depset", func(t *testing.T) {
		// Create mock chain with InteropTime but nil Interop
		activationTime := uint64(1234567890)
		mockChain := &Chain{
			Name:    "test",
			Network: "test",
			config: &ChainConfig{
				ChainID: 42,
				Hardforks: HardforkConfig{
					InteropTime: &activationTime,
				},
				Interop: nil,
			},
		}

		// Set configOnce as already done
		mockChain.configOnce.Do(func() {})

		// Replace chains map with our test chain
		BuiltInConfigs = &ChainConfigLoader{
			Chains: map[uint64]*Chain{42: mockChain},
		}

		depset, err := GetDepset(42)
		require.NoError(t, err)
		require.NotNil(t, depset)

		// Verify the default dependency was created
		selfDep, exists := depset["42"]
		require.True(t, exists)
		require.Equal(t, selfDep, Dependency{})
	})

	t.Run("existing Interop depset returned", func(t *testing.T) {
		// Create mock chain with existing Interop dependencies
		activationTime := uint64(1234567890)
		mockChain := &Chain{
			Name:    "test",
			Network: "test",
			config: &ChainConfig{
				ChainID: 42,
				Hardforks: HardforkConfig{
					InteropTime: &activationTime,
				},
				Interop: &Interop{
					Dependencies: map[string]Dependency{
						"42": {},
						"43": {},
					},
				},
			},
		}

		// Set configOnce as already done
		mockChain.configOnce.Do(func() {})

		// Replace chains map with our test chain
		BuiltInConfigs = &ChainConfigLoader{
			Chains: map[uint64]*Chain{42: mockChain},
		}

		depset, err := GetDepset(42)
		require.NoError(t, err)
		require.NotNil(t, depset)
		require.Equal(t, 2, len(depset))

		selfDep, exists := depset["42"]
		require.True(t, exists)
		require.Equal(t, selfDep, Dependency{})

		otherDep, exists := depset["43"]
		require.True(t, exists)
		require.Equal(t, otherDep, Dependency{})
	})
}
