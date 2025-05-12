package upgrade

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

var SimpleInterop presets.TestSetup[*presets.SimpleInterop]

func TestMain(m *testing.M) {
	SimpleInterop = presets.NewSimpleInterop
	presets.DoMain(m,
		presets.ConfigureSimpleInterop(),
		presets.WithSuggestedInteropActivationOffset(30),
		presets.WithInteropNotAtGenesis())
}
