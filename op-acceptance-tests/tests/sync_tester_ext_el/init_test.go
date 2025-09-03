package sync_tester_ext_el

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	InitialL2Block = uint64(32012748)
)

func TestMain(m *testing.M) {
	L2NetworkName := "op-sepolia"
	L1ChainID := eth.ChainIDFromUInt64(11155111)

	L2ELEndpoint := "https://ci-sepolia-l2.optimism.io"
	L1CLBeaconEndpoint := "https://ci-sepolia-beacon.optimism.io"
	L1ELEndpoint := "https://ci-sepolia-l1.optimism.io"

	// Endpoints when running with Tailscale networking
	if os.Getenv("TAILSCALE_NETWORKING") == "true" {
		L2ELEndpoint = "https://proxyd-l2-sepolia.primary.client.dev.oplabs.cloud"
		L1CLBeaconEndpoint = "https://beacon-api-proxy-sepolia.primary.client.dev.oplabs.cloud"
		L1ELEndpoint = "https://proxyd-l1-sepolia.primary.client.dev.oplabs.cloud"
	}

	presets.DoMain(m, presets.WithMinimalExternalELWithSuperchainRegistry(L1CLBeaconEndpoint, L1ELEndpoint, L2ELEndpoint, L1ChainID, L2NetworkName, eth.FCUState{
		Latest:    InitialL2Block,
		Safe:      InitialL2Block,
		Finalized: InitialL2Block,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)

}
