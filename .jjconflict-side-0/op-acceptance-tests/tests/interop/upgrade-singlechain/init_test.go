package upgrade

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithSingleChainInterop(),
		presets.WithSuggestedInteropActivationOffset(30),
		presets.WithInteropNotAtGenesis(),
		presets.WithL2NetworkCount(1), // Specifically testing dependency set of 1 upgrade
	)
}
