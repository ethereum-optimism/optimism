package bundle

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/superchain"
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
		require.ErrorIs(t, err, superchain.ErrUnknownChain)
		require.Nil(t, chain)
	})
}

func TestGetSuperchain(t *testing.T) {
	mainnet, err := GetSuperchain("mainnet")
	require.NoError(t, err)

	require.Equal(t, "Mainnet", mainnet.Name)
	require.Equal(t, common.HexToAddress("0x95703e0982140D16f8ebA6d158FccEde42f04a4C"), mainnet.SuperchainConfigAddr)
	require.Equal(t, common.HexToAddress("0xA8447329e52F64AED2bFc9E7a2506F7D369f483a"), mainnet.SaferSafesAddr)
	require.Equal(t, uint64(1764691201), *mainnet.Hardforks.JovianTime)
	require.EqualValues(t, 1, mainnet.L1.ChainID)

	_, err = GetSuperchain("not a network")
	require.Error(t, err)
}
