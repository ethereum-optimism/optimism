package sync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

var RedundancyInterop presets.TestSetup[*presets.RedundancyInterop]

func TestMain(m *testing.M) {
	RedundancyInterop = presets.NewRedundancyInterop
	presets.DoMain(m, presets.ConfigureRedundancyInterop())
}
