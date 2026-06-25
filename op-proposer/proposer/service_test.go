package proposer

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestResolveProposerAddressConfigUsesSystemConfig(t *testing.T) {
	cfg := validConfig()
	cfg.DGFAddress = ""
	cfg.SystemConfigAddress = common.Address{0x11}.Hex()

	dgfAddress, systemConfigAddress, err := resolveProposerAddressConfig(cfg, true)
	require.NoError(t, err)
	require.Nil(t, dgfAddress)
	require.Equal(t, common.HexToAddress(cfg.SystemConfigAddress), *systemConfigAddress)
}

func TestResolveProposerAddressConfigUsesNetworkSystemConfig(t *testing.T) {
	network := networkWithSystemConfig(t)
	chain := chaincfg.ChainByName(network)
	require.NotNil(t, chain)
	require.NotNil(t, chain.Addresses.SystemConfigProxy)

	cfg := validConfig()
	cfg.DGFAddress = ""
	cfg.Network = network

	dgfAddress, systemConfigAddress, err := resolveProposerAddressConfig(cfg, true)
	require.NoError(t, err)
	require.Nil(t, dgfAddress)
	require.Equal(t, *chain.Addresses.SystemConfigProxy, *systemConfigAddress)
}

func TestResolveProposerAddressConfigAllowsExplicitOverrides(t *testing.T) {
	cfg := validConfig()
	cfg.SystemConfigAddress = common.Address{0x11}.Hex()
	dgfOverride := common.Address{0xaa}.Hex()
	cfg.DGFAddress = dgfOverride

	dgfAddress, systemConfigAddress, err := resolveProposerAddressConfig(cfg, true)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(dgfOverride), *dgfAddress)
	require.Equal(t, common.HexToAddress(cfg.SystemConfigAddress), *systemConfigAddress)
}

func TestResolveProposerAddressConfigRequiresSystemConfigForAuto(t *testing.T) {
	cfg := validConfig()

	_, _, err := resolveProposerAddressConfig(cfg, true)
	require.ErrorContains(t, err, "missing SystemConfig address")
}

func networkWithSystemConfig(t *testing.T) string {
	t.Helper()
	for _, network := range chaincfg.AvailableNetworks() {
		chain := chaincfg.ChainByName(network)
		if chain == nil {
			continue
		}
		if chain.Addresses.SystemConfigProxy != nil {
			return network
		}
	}
	t.Fatal("expected at least one known network with SystemConfigProxy")
	return ""
}
