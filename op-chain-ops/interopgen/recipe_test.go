package interopgen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
)

func TestInteropDevRecipeBuildUseL2CM(t *testing.T) {
	rec := InteropDevRecipe{
		L1ChainID:        900100,
		L2s:              []InteropDevL2Recipe{{ChainID: 900200, UseL2CM: true}, {ChainID: 900201}},
		GenesisTimestamp: uint64(1234567),
	}
	hd, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	worldCfg, err := rec.Build(hd)
	require.NoError(t, err)

	require.True(t, worldCfg.L2s["900200"].UseL2CM)
	require.False(t, worldCfg.L2s["900201"].UseL2CM)
}
