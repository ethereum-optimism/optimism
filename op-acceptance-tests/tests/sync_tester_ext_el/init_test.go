package sync_tester_ext_el

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestMain(m *testing.M) {
	if os.Getenv("NIGHTLY_CI_TAILSCALE_JOB") != "true" {
		// Skipping tests because NIGHTLY_CI_TAILSCALE_JOB is not set
		return
	}

	L2NetworkName := "sepolia"
	L1ChainID := eth.ChainIDFromUInt64(11155111)

	// tailscale
	L2ELEndpoint := "https://sepolia.optimism.io/"
	L1CLBeaconEndpoint := "https://beacon-api-proxy-sepolia.primary.client.dev.oplabs.cloud"
	L1ELEndpoint := "https://proxyd-l1-sepolia.primary.client.dev.oplabs.cloud"

	presets.DoMain(m, presets.WithMinimalExternalELWithSuperchainRegistry(L1CLBeaconEndpoint, L1ELEndpoint, L2ELEndpoint, L1ChainID, L2NetworkName, eth.FCUState{
		Latest:    32012748,
		Safe:      32012748,
		Finalized: 32012748,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)

}
