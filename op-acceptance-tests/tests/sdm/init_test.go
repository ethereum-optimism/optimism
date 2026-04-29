package sdm

import (
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type sdmRethSystem struct {
	L1EL         *dsl.L1ELNode
	L2EL         *dsl.L2ELNode
	L2CL         *dsl.L2CLNode
	L2ELVerifier *dsl.L2ELNode
	L2CLVerifier *dsl.L2CLNode
	L2Batcher    *dsl.L2Batcher
	FunderL2     *dsl.Funder
}

func newSDMRethSystem(t devtest.T, sdmEnabled bool) *sdmRethSystem {
	return newSDMRethSystemWithBatcherOptions(t, sdmEnabled)
}

func newSDMRethSystemWithBatcherOptions(t devtest.T, sdmEnabled bool, batcherOpts ...sysgo.BatcherOption) *sdmRethSystem {
	var opRethOpts []sysgo.OpRethOption
	if sdmEnabled {
		opRethOpts = append(opRethOpts, sysgo.OpRethWithSDMEnabled())
	}

	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer-op-reth",
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: true,
			},
			{
				ELKey:       "verifier-op-reth",
				CLKey:       "verifier",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: false,
			},
		},
		BatcherOptions: batcherOpts,
		OpRethOptions:  opRethOpts,
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	frontends.L2Batcher.Stop()
	t.Require().Len(frontends.Nodes, 2, "SDM op-reth system must include sequencer and verifier nodes")

	var verifierEL *dsl.L2ELNode
	var verifierCL *dsl.L2CLNode
	for _, node := range frontends.Nodes {
		if !node.Spec.IsSequencer {
			verifierEL = node.EL
			verifierCL = node.CL
			break
		}
	}
	t.Require().NotNil(verifierEL, "missing SDM verifier EL node")
	t.Require().NotNil(verifierCL, "missing SDM verifier CL node")

	wallet := dsl.NewRandomHDWallet(t, 30)
	return &sdmRethSystem{
		L1EL:         frontends.L1EL,
		L2EL:         frontends.L2Network.PrimaryEL(),
		L2CL:         frontends.L2Network.PrimaryCL(),
		L2ELVerifier: verifierEL,
		L2CLVerifier: verifierCL,
		L2Batcher:    frontends.L2Batcher,
		FunderL2:     dsl.NewFunder(wallet, frontends.FaucetL2, frontends.L2Network.PrimaryEL()),
	}
}

func withSingularBatcher(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
	cfg.BatchType = derive.SingularBatchType
}
