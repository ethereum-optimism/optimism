package superchain

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

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

// TestSuperchain_DecodeIgnoresLegacyProtocolVersionsAddr proves the TOML
// decoder silently ignores the legacy `protocol_versions_addr` key. The
// embedded superchain-registry configs still carry it, so removing the
// struct field must not break decoding.
func TestSuperchain_DecodeIgnoresLegacyProtocolVersionsAddr(t *testing.T) {
	const data = `
name = "Legacy"
protocol_versions_addr = "0x8062AbC286f5e7D9428a0Ccb9AbD71e50d93b935"
superchain_config_addr = "0x95703e0982140D16f8ebA6d158FccEde42f04a4C"
op_contracts_manager_addr = "0x0000000000000000000000000000000000000001"
safer_safes_addr = "0xA8447329e52F64AED2bFc9E7a2506F7D369f483a"

[L1]
chain_id = 1
`
	var sc Superchain
	_, err := toml.NewDecoder(strings.NewReader(data)).Decode(&sc)
	require.NoError(t, err)
	require.Equal(t, "Legacy", sc.Name)
	require.Equal(t, common.HexToAddress("0x95703e0982140D16f8ebA6d158FccEde42f04a4C"), sc.SuperchainConfigAddr)
}
