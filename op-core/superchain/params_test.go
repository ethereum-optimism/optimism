package superchain

import (
	"testing"

	"github.com/stretchr/testify/require"

	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

func TestLoadOpChainConfig(t *testing.T) {
	t.Run("mainnet", func(t *testing.T) {
		cfg, err := LoadOpChainConfig(opparams.OPMainnetChainID)
		require.NoError(t, err)
		require.Equal(t, uint64(opparams.OPMainnetChainID), bigs.Uint64Strict(cfg.ChainID))
		require.NotNil(t, cfg.Optimism)
		require.NotNil(t, cfg.RegolithTime)
		require.Equal(t, uint64(0), *cfg.RegolithTime)
		require.NotNil(t, cfg.BedrockBlock)
		require.Equal(t, int64(opparams.OPMainnetGenesisBlockNum), cfg.BedrockBlock.Int64())
	})

	t.Run("nonexistent chain", func(t *testing.T) {
		cfg, err := LoadOpChainConfig(23409527340)
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}
