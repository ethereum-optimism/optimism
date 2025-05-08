package upgrade

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

var SimpleInterop presets.TestSetup[*presets.SimpleInterop]

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.NewSimpleInterop(&SimpleInterop),
		presets.WithSuggestedInteropActivationOffset(30),
		presets.WithInteropNotAtGenesis())
}
