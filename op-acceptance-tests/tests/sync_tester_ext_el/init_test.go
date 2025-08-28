package sync_tester_ext_el

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestMain(m *testing.M) {
	L2NetworkName := "sepolia"
	L2ELEndpoint := "https://sepolia.optimism.io/"
	L1CLBeaconEndpoint := "https://beacon-api-proxy-sepolia.primary.client.dev.oplabs.cloud"
	L1ELEndpoint := "https://proxyd-l1-sepolia.primary.client.dev.oplabs.cloud"
	L1ChainID := eth.ChainIDFromUInt64(11155111)

	presets.DoMain(m, presets.WithMinimalExternalELWithSuperchainRegistry(L1CLBeaconEndpoint, L1ELEndpoint, L2ELEndpoint, L1ChainID, L2NetworkName, eth.FCUState{
		Latest:    31987000,
		Safe:      31987000,
		Finalized: 31987000,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)

}
