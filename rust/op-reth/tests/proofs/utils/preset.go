package utils

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// MixedOpProofPreset sets up a two-node L2 devnet (sequencer + validator) on op-reth.
type MixedOpProofPreset struct {
	Log log.Logger
	T   devtest.T

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2Chain   *dsl.L2Network
	L2Batcher *dsl.L2Batcher

	L2ELSequencer *dsl.L2ELNode
	L2CLSequencer *dsl.L2CLNode

	L2ELValidator *dsl.L2ELNode
	L2CLValidator *dsl.L2CLNode

	Wallet *dsl.HDWallet

	FunderL1 *dsl.FunderEOA
	FunderL2 *dsl.FunderEOA

	TestSequencer *dsl.TestSequencer
}

func (m *MixedOpProofPreset) L2ELSequencerNode() *dsl.L2ELNode {
	return m.L2ELSequencer
}

func (m *MixedOpProofPreset) L2ELValidatorNode() *dsl.L2ELNode {
	return m.L2ELValidator
}

// RethWithProofL2ELNode returns the first node running op-reth with proof history.
// Falls back to the validator, then sequencer.
func (m *MixedOpProofPreset) RethWithProofL2ELNode() *dsl.L2ELNode {
	return m.L2ELValidator
}

// NewMixedOpProofPreset creates the preset using MixedSingleChainRuntime.
func NewMixedOpProofPreset(t devtest.T) *MixedOpProofPreset {
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer",
				CLKey:       "sequencer",
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: true,
			},
			{
				ELKey:       "validator",
				CLKey:       "validator",
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: false,
			},
		},
		WithTestSequencer: true,
		TestSequencerName: "test-sequencer",
	})

	frontends := presets.NewMixedSingleChainFrontends(t, runtime)

	t.Require().Equal(2, len(frontends.Nodes), "expected exactly 2 nodes")

	var seqNode, valNode *presets.MixedSingleChainNodeFrontends
	for i := range frontends.Nodes {
		if frontends.Nodes[i].Spec.IsSequencer {
			seqNode = &frontends.Nodes[i]
		} else {
			valNode = &frontends.Nodes[i]
		}
	}
	t.Require().NotNil(seqNode, "expected a sequencer node")
	t.Require().NotNil(valNode, "expected a validator node")

	out := &MixedOpProofPreset{
		Log:           t.Logger(),
		T:             t,
		L1Network:     frontends.L1Network,
		L1EL:          frontends.L1EL,
		L2Chain:       frontends.L2Network,
		L2Batcher:     frontends.L2Batcher,
		L2ELSequencer: seqNode.EL,
		L2CLSequencer: seqNode.CL,
		L2ELValidator: valNode.EL,
		L2CLValidator: valNode.CL,
		Wallet:        frontends.Wallet,
		FunderL1:      frontends.FunderL1,
		FunderL2:      frontends.FunderL2,
		TestSequencer: frontends.TestSequencer,
	}
	return out
}
