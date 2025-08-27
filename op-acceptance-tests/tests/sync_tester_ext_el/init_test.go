package sync_tester_ext_el

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

func TestMain(m *testing.M) {
	OPSepoliaChainID := eth.ChainIDFromUInt64(11155420)
	OPSepoliaEndpoint := "https://sepolia.optimism.io/"
	presets.DoMain(m, presets.WithMinimalExternalEL(OPSepoliaEndpoint, OPSepoliaChainID, sttypes.FCUState{
		Latest:    22285447,
		Safe:      22285447,
		Finalized: 22285447,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
