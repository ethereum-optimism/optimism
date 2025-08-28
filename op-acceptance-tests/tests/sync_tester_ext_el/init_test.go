package sync_tester_ext_el

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestMain(m *testing.M) {
	OPSepoliaEndpoint := "https://sepolia.optimism.io/"

	// OPSepoliaChainID := eth.ChainIDFromUInt64(11155420)
	presets.DoMain(m, presets.WithMinimalExternalELWithSuperchainRegistry(OPSepoliaEndpoint, "sepolia", eth.FCUState{
		Latest:    22285447,
		Safe:      22285447,
		Finalized: 22285447,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)

}
