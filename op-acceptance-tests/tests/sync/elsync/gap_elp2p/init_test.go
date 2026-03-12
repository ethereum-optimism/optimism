package gap_elp2p

import (
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func gapELP2POpts() []presets.Option {
	return []presets.Option{
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			// For stopping derivation, not to advance safe heads
			cfg.Stopped = true
		}),
	}
}

func newGapELP2PSystem(t devtest.T) *presets.SingleChainMultiNode {
	return presets.NewSingleChainMultiNodeWithoutP2PWithoutCheck(t, gapELP2POpts()...)
}
