package bundle

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/genesis"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
)

// TestEmbeddedGenesesMatchRegistryHash decodes each embedded genesis with the op-core codec and
// computes its block with the chain config built from the registry TOML, whatever config the
// genesis file carries. The block hash must be the registry's L2 genesis hash. OP Mainnet's
// genesis carries a stateHash in place of its alloc, which op-core does not support.
func TestEmbeddedGenesesMatchRegistryHash(t *testing.T) {
	for chainID, ch := range Chains {
		t.Run(ch.Network+"/"+ch.Name, func(t *testing.T) {
			t.Parallel()
			data, err := ch.GenesisData()
			require.NoError(t, err)
			var g genesis.Genesis
			err = json.Unmarshal(data, &g)
			if chainID == opparams.OPMainnetChainID {
				require.ErrorContains(t, err, "stateHash")
				return
			}
			require.NoError(t, err)
			cfg, err := ch.Config()
			require.NoError(t, err)
			g.Config = cfg.OpChainConfig()
			require.Equal(t, cfg.Genesis.L2.Hash, g.ToBlock().Hash())
		})
	}
}
