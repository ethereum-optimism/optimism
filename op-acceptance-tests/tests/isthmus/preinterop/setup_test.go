package preinterop

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func newSimpleInteropPreinterop(t devtest.T) *presets.SimpleInterop {
	return presets.NewSimpleInteropIsthmusSuper(
		t,
		presets.WithChallengerCannonKonaEnabled(),
	)
}
