package utils

import (
	"os"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// MixedOpProofPreset sets up a two-node L2 devnet (sequencer + validator)
// with configurable EL clients via environment variables:
//
//   - OP_DEVSTACK_PROOF_SEQUENCER_EL: "op-geth", "op-reth" (default), or "op-reth-proof-v1"
//   - OP_DEVSTACK_PROOF_VALIDATOR_EL: "op-reth-proof-v1" (default), "op-reth", or "op-geth"
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

	FunderL1 *dsl.EOA
	FunderL2 *dsl.EOA

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

func resolveELSpec(envVar string, defaultKind sysgo.MixedL2ELKind) sysgo.MixedL2ELKind {
	switch os.Getenv(envVar) {
	case "op-reth-proof-v2":
		return sysgo.MixedL2ELOpRethV2
	case "op-reth-proof-v1", "op-reth":
		return sysgo.MixedL2ELOpReth
	case "op-geth":
		return sysgo.MixedL2ELOpGeth
	default:
		return defaultKind
	}
}

// NewMixedOpProofPreset creates the preset using MixedSingleChainRuntime for
// full control over EL client types.
func NewMixedOpProofPreset(t devtest.T) *MixedOpProofPreset {
	seqKind := resolveELSpec("OP_DEVSTACK_PROOF_SEQUENCER_EL", sysgo.MixedL2ELOpReth)
	valKind := resolveELSpec("OP_DEVSTACK_PROOF_VALIDATOR_EL", sysgo.MixedL2ELOpReth)

	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer",
				CLKey:       "sequencer",
				ELKind:      seqKind,
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: true,
			},
			{
				ELKey:       "validator",
				CLKey:       "validator",
				ELKind:      valKind,
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
