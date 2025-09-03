package upgrade

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithSimpleInterop(),
		presets.WithSuggestedInteropActivationOffset(120), // Increased from 30 to 120 seconds to prevent race condition
		presets.WithInteropNotAtGenesis())
}
