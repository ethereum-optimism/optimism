package bundle

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAllEmbeddedConfigsDecodeStrictly loads every embedded chain and superchain config, which the
// loader decodes strictly. It fails if any config carries a key the Go structs do not model — the exact
// drift that let keep_karst_upgrade_gas be silently dropped. A superchain-registry bump that adds a
// field therefore cannot merge until the structs are updated to consume it.
//
// It also pins that each Chains index key equals the chain ID inside the config, so lookups keyed
// by the index (e.g. chaincfg's display-name map) need not parse the TOMLs.
func TestAllEmbeddedConfigsDecodeStrictly(t *testing.T) {
	networks := map[string]struct{}{}
	for chainID, ch := range Chains {
		networks[ch.Network] = struct{}{}
		t.Run(ch.Network+"/"+ch.Name, func(t *testing.T) {
			cfg, err := ch.Config()
			require.NoError(t, err)
			require.Equal(t, chainID, cfg.ChainID, "chains.json index key must match the config's chain ID")
		})
	}

	for network := range networks {
		t.Run("superchain/"+network, func(t *testing.T) {
			_, err := GetSuperchain(network)
			require.NoError(t, err)
		})
	}
}
