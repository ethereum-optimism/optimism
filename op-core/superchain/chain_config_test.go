package superchain

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/stretchr/testify/require"
)

func TestLoadChainConfigFromChainID(t *testing.T) {
	t.Run("mainnet", func(t *testing.T) {
		chainID := uint64(10)
		cfg, err := LoadChainConfigFromChainID(chainID)
		require.NoError(t, err)
		require.Equal(t, chainID, bigs.Uint64Strict(cfg.ChainID))
	})

	t.Run("nonexistent chain", func(t *testing.T) {
		chainID := uint64(23409527340)
		cfg, err := LoadChainConfigFromChainID(chainID)
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}
