package superchain

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

// TestAllEmbeddedConfigsDecodeStrictly loads every embedded chain and superchain config with the
// strict decoder. It fails if any config carries a key the Go structs do not model — the exact
// drift that let keep_karst_upgrade_gas be silently dropped. A superchain-registry bump that adds a
// field therefore cannot merge until the structs are updated to consume it.
func TestAllEmbeddedConfigsDecodeStrictly(t *testing.T) {
	networks := map[string]struct{}{}
	for _, ch := range BuiltInConfigs.Chains {
		networks[ch.Network] = struct{}{}
		t.Run(ch.Network+"/"+ch.Name, func(t *testing.T) {
			f, err := BuiltInConfigs.configDataReader.Open(path.Join("configs", ch.Network, ch.Name+".toml"))
			require.NoError(t, err)
			defer f.Close()
			var cfg ChainConfig
			require.NoError(t, jsonutil.DecodeTOMLStrict(f, &cfg, nil))
		})
	}

	for network := range networks {
		t.Run("superchain/"+network, func(t *testing.T) {
			f, err := BuiltInConfigs.configDataReader.Open(path.Join("configs", network, "superchain.toml"))
			require.NoError(t, err)
			defer f.Close()
			var sc Superchain
			require.NoError(t, jsonutil.DecodeTOMLStrict(f, &sc, nil))
		})
	}
}
