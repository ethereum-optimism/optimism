package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// skipOpNodeOnlyPresetOnKonaNode skips when the selected CL runs Kona. The supernode presets
// call it because they always start op-node and op-supernode CLs. Use sysgo.SkipOnKonaNode
// for individual tests that exercise a Kona client gap.
func skipOpNodeOnlyPresetOnKonaNode(t devtest.T) {
	if sysgo.ResolveMixedL2CLKind(t).RunsKona() {
		t.Skip("skipping on kona: this preset starts only op-node and op-supernode CLs whatever DEVSTACK_L2CL_KIND says")
	}
}
