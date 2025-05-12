package sync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

var SimpleInterop presets.TestSetup[*presets.SimpleInterop]

func TestMain(m *testing.M) {
	SimpleInterop = presets.NewSimpleInterop
	presets.DoMain(m, presets.ConfigureSimpleInterop())
}
