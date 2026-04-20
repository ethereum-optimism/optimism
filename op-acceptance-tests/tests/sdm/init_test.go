package sdm

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type sdmRethSystem struct {
	L2EL      *dsl.L2ELNode
	L2Batcher *dsl.L2Batcher
	FunderL2  *dsl.Funder
}

func newSDMRethSystem(t devtest.T, sdmEnabled bool) *sdmRethSystem {
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer-op-reth",
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: true,
				SDMEnabled:  sdmEnabled,
			},
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	frontends.L2Batcher.Stop()

	wallet := dsl.NewRandomHDWallet(t, 30)
	return &sdmRethSystem{
		L2EL:      frontends.L2Network.PrimaryEL(),
		L2Batcher: frontends.L2Batcher,
		FunderL2:  dsl.NewFunder(wallet, frontends.FaucetL2, frontends.L2Network.PrimaryEL()),
	}
}
