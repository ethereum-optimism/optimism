package superchain

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

func TestOpChainConfig(t *testing.T) {
	t.Run("hardforks and EIP-1559 params", func(t *testing.T) {
		cfg := (&ChainConfig{
			ChainID: 1234,
			Hardforks: HardforkConfig{
				CanyonTime: ptr.New(uint64(10)),
				KarstTime:  ptr.New(uint64(20)),
			},
			Optimism: &OptimismConfig{
				EIP1559Elasticity:        6,
				EIP1559Denominator:       50,
				EIP1559DenominatorCanyon: ptr.New(uint64(250)),
			},
		}).OpChainConfig()
		require.Equal(t, big.NewInt(1234), cfg.ChainID)
		require.Equal(t, big.NewInt(0), cfg.BedrockBlock)
		require.Equal(t, ptr.New(uint64(0)), cfg.RegolithTime)
		require.Equal(t, ptr.New(uint64(10)), cfg.CanyonTime)
		require.Equal(t, ptr.New(uint64(20)), cfg.KarstTime)
		require.Nil(t, cfg.EcotoneTime)
		require.Equal(t, &opparams.OptimismConfig{
			EIP1559Elasticity:        6,
			EIP1559Denominator:       50,
			EIP1559DenominatorCanyon: ptr.New(uint64(250)),
		}, cfg.Optimism)
	})

	t.Run("OP Mainnet starts at its Bedrock-migration block", func(t *testing.T) {
		cfg := (&ChainConfig{ChainID: opparams.OPMainnetChainID}).OpChainConfig()
		require.Equal(t, big.NewInt(opparams.OPMainnetGenesisBlockNum), cfg.BedrockBlock)
		require.Nil(t, cfg.Optimism)
	})
}
