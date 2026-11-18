package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// skipOpNodeOnlyPresetOnKonaNode skips the test under DEVSTACK_L2CL_KIND=kona-node. A preset calls
// it first when its runtime starts only op-node and op-supernode CLs whatever that variable says,
// as the supernode and interop presets do: under kona-node the test would repeat the op-node job.
// It is separate from sysgo.SkipOnKonaNode so that list names only real kona-node gaps. Remove
// the call from a preset once its runtime starts the CL kind the variable selects.
func skipOpNodeOnlyPresetOnKonaNode(t devtest.T) {
	if sysgo.ResolveMixedL2CLKind(t) == sysgo.MixedL2CLKona {
		t.Skip("skipping on kona-node: this preset starts only op-node and op-supernode CLs whatever DEVSTACK_L2CL_KIND says")
	}
}
