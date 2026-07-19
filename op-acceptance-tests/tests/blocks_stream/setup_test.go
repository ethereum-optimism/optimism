package blocks_stream

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

const (
	sequencerELKey = "sequencer-op-reth"
	validatorELKey = "validator-op-reth"
)

type blocksStreamSystem struct {
	SourceEL    *dsl.L2ELNode
	ValidatorEL *dsl.L2ELNode
	ValidatorCL *dsl.L2CLNode
	Batcher     *dsl.L2Batcher
	Funder      *dsl.Funder
	Wallet      *dsl.HDWallet
}

func newBlocksStreamSystem(t devtest.T) *blocksStreamSystem {
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       sequencerELKey,
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: true,
			},
			{
				ELKey:             validatorELKey,
				CLKey:             "validator",
				ELKind:            sysgo.MixedL2ELOpReth,
				CLKind:            sysgo.MixedL2CLKona,
				BlocksSourceELKey: sequencerELKey,
				IsolateFromL2P2P:  true,
			},
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "blocks stream system requires a sequencer and validator")

	var sourceEL, validatorEL *dsl.L2ELNode
	var validatorCL *dsl.L2CLNode
	for _, node := range frontends.Nodes {
		switch node.Spec.ELKey {
		case sequencerELKey:
			sourceEL = node.EL
		case validatorELKey:
			validatorEL = node.EL
			validatorCL = node.CL
		}
	}
	t.Require().NotNil(sourceEL, "missing blocks stream source EL")
	t.Require().NotNil(validatorEL, "missing blocks stream validator EL")
	t.Require().NotNil(validatorCL, "missing blocks stream validator CL")

	wallet := dsl.NewHDWallet(t, devkeys.TestMnemonic, 30)
	return &blocksStreamSystem{
		SourceEL:    sourceEL,
		ValidatorEL: validatorEL,
		ValidatorCL: validatorCL,
		Batcher:     frontends.L2Batcher,
		Funder:      dsl.NewFunder(wallet, frontends.FaucetL2, sourceEL),
		Wallet:      wallet,
	}
}
