package presets

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// MinimalWithKona embeds Minimal and bundles everything needed to run kona-host in --native mode
// against a live devstack.
type MinimalWithKona struct {
	*Minimal

	VMConfig *vm.Config
	Dir      string
}

// NewMinimalWithKona creates a Minimal preset with the kona-host binary located and the chain
// configs needed to invoke kona natively.
func NewMinimalWithKona(t devtest.T, opts ...Option) *MinimalWithKona {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewMinimalWithKona", opts, minimalPresetSupportedOptionKinds)
	out := minimalWithKonaFromRuntime(t, sysgo.NewMinimalRuntimeWithConfig(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}

func minimalWithKonaFromRuntime(t devtest.T, runtime *sysgo.SingleChainRuntime) *MinimalWithKona {
	dir := t.TempDir()
	return &MinimalWithKona{
		Minimal:  minimalFromRuntime(t, runtime),
		VMConfig: runtime.VMConfig(t, dir),
		Dir:      dir,
	}
}
