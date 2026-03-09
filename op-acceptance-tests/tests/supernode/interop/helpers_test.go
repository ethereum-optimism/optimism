package interop

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func newSupernodeInteropWithTimeTravel(t devtest.T, delaySeconds uint64) *presets.TwoL2SupernodeInterop {
	return presets.NewTwoL2SupernodeInterop(t, delaySeconds,
		presets.WithTimeTravelEnabled(),
	)
}
