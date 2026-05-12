package presets

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// MinimalWithKona embeds Minimal and bundles everything needed to run kona-host in --native mode
// against a live devstack.
type MinimalWithKona struct {
	*Minimal

	vmConfig *vm.Config
	dir      string
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
		vmConfig: runtime.VMConfig(t, dir),
		dir:      dir,
	}
}

func (m *MinimalWithKona) RunKonaNative(agreedBlock, claimBlock uint64) bool {
	return rustbin.RunKonaNative(m.T, m.T.Logger(), m.vmConfig, m.dir, m.L2CL.LocalGameInputs(agreedBlock, claimBlock))
}

// WithKonaKarstAtGenesis configures kona-host to think Karst was active at
// genesis, regardless of what the live chain's rollup config says. Use to
// deliberately mismatch kona's view of the chain from the live devstack so
// kona disagrees on state transitions that depend on Karst.
func WithKonaKarstAtGenesis() Option {
	return option{
		kinds: optionKindAfterBuild,
		applyPresetFn: func(target any) {
			sys, ok := target.(*MinimalWithKona)
			if !ok {
				return
			}
			rollupPath := filepath.Join(sys.dir, "rollup.json")
			data, err := os.ReadFile(rollupPath)
			sys.T.Require().NoError(err, "read kona rollup config")

			var cfg rollup.Config
			sys.T.Require().NoError(json.Unmarshal(data, &cfg), "unmarshal kona rollup config")

			cfg.KarstTime = ptr.Zero64

			out, err := json.Marshal(&cfg)
			sys.T.Require().NoError(err, "marshal kona rollup config")
			sys.T.Require().NoError(
				os.WriteFile(rollupPath, out, 0o644),
				"write kona rollup config")
		},
	}
}
