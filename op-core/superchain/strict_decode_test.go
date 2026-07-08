package superchain

import (
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

// TestAllEmbeddedConfigsDecodeStrictly loads every embedded chain and superchain config with the
// strict decoder. It fails if any config carries a key the Go structs do not model — the exact
// drift that let keep_karst_upgrade_gas be silently dropped. A superchain-registry bump that adds a
// field therefore cannot merge until the structs are updated to consume it (or, for the superchain,
// until the key is added to legacySuperchainKeys as a conscious decision).
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
			require.NoError(t, jsonutil.DecodeTOMLStrict(f, &sc, isLegacySuperchainKey))
		})
	}
}

// TestSuperchainDecodeToleratesLegacyProtocolVersionsAddr documents why the superchain decode
// allow-lists protocol_versions_addr: the Superchain struct deliberately no longer models that
// legacy key, and the embedded superchain.toml files still carry it, so strict decoding must
// tolerate it — while still rejecting a genuinely unknown key.
func TestSuperchainDecodeToleratesLegacyProtocolVersionsAddr(t *testing.T) {
	const legacy = `
name = "Legacy"
protocol_versions_addr = "0x8062AbC286f5e7D9428a0Ccb9AbD71e50d93b935"
superchain_config_addr = "0x95703e0982140D16f8ebA6d158FccEde42f04a4C"
op_contracts_manager_addr = "0x0000000000000000000000000000000000000001"
safer_safes_addr = "0xA8447329e52F64AED2bFc9E7a2506F7D369f483a"

[L1]
chain_id = 1
`
	var sc Superchain
	require.NoError(t, jsonutil.DecodeTOMLStrict(strings.NewReader(legacy), &sc, isLegacySuperchainKey))
	require.Equal(t, "Legacy", sc.Name)

	const unknown = `
name = "Legacy"
some_new_key = true

[L1]
chain_id = 1
`
	err := jsonutil.DecodeTOMLStrict(strings.NewReader(unknown), &Superchain{}, isLegacySuperchainKey)
	require.Error(t, err)
	require.Contains(t, err.Error(), "some_new_key")
}
