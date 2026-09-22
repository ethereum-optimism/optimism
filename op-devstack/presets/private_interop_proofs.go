package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// PrivateInteropProofs exposes the public chains used by the proof system and,
// separately, the private sequencer used to submit application transactions.
// SimpleInterop's chain B is the public projection, never the private execution chain.
type PrivateInteropProofs struct {
	*SimpleInterop
	Private *TwoL2SupernodeInterop
}

// NewPrivateInteropProofs creates a private pair with the shared super-root proof
// lifecycle. WithZK selects the existing native execution/mock verifier test path.
func NewPrivateInteropProofs(t devtest.T, opts ...Option) *PrivateInteropProofs {
	opts = append([]Option{WithPrivateInteropChain()}, opts...)
	cfg, _ := collectSupportedPresetConfig(t, "NewPrivateInteropProofs", opts,
		twoL2SupernodeProofsPresetSupportedOptionKinds|optionKindPrivateInteropChain|optionKindGlobalL2CL)
	runtime, public := sysgo.NewPrivateInteropProofsRuntimeWithConfig(t, cfg)
	return &PrivateInteropProofs{
		SimpleInterop: simpleInteropFromSupernodeProofsRuntime(t, public),
		Private:       twoL2PrivateInteropFromRuntime(t, runtime),
	}
}
