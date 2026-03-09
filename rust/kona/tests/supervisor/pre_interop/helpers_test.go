package preinterop

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	spresets "github.com/ethereum-optimism/optimism/rust/kona/tests/supervisor/presets"
)

func newMinimalPreInterop(t devtest.T) *presets.SimpleInterop {
	return spresets.NewSimpleInteropMinimal(t,
		spresets.WithSuggestedInteropActivationOffset(30),
		spresets.WithInteropNotAtGenesis(),
	)
}
