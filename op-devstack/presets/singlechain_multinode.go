package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type SingleChainMultiNode struct {
	Minimal

	L2ELB *dsl.L2ELNode
	L2CLB *dsl.L2CLNode
}

// NewSingleChainMultiNode creates a fresh SingleChainMultiNode target for the current
// test.
//
// The target is created from the runtime plus any additional preset options.
func NewSingleChainMultiNode(t devtest.T, opts ...Option) *SingleChainMultiNode {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainMultiNode", opts, minimalPresetSupportedOptionKinds)
	out := singleChainMultiNodeFromRuntime(t, sysgo.NewSingleChainMultiNodeRuntimeWithConfig(t, true, presetCfg), true)
	presetOpts.applyPreset(out)
	return out
}

// NewSingleChainMultiNodeWithoutCheck creates a fresh SingleChainMultiNode target for the
// current test, without running the initial verifier sync checks.
//
// The target is created from the runtime plus any additional preset options.
func NewSingleChainMultiNodeWithoutCheck(t devtest.T, opts ...Option) *SingleChainMultiNode {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainMultiNodeWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	out := singleChainMultiNodeFromRuntime(t, sysgo.NewSingleChainMultiNodeRuntimeWithConfig(t, true, presetCfg), false)
	presetOpts.applyPreset(out)
	return out
}

// NewSingleChainMultiNodeWithoutP2PWithoutCheck creates a fresh SingleChainMultiNode
// target without preconfigured sequencer/verifier P2P links and without running initial sync
// checks.
//
// The target is created from the runtime plus any additional preset options.
func NewSingleChainMultiNodeWithoutP2PWithoutCheck(t devtest.T, opts ...Option) *SingleChainMultiNode {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainMultiNodeWithoutP2PWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	out := singleChainMultiNodeFromRuntime(t, sysgo.NewSingleChainMultiNodeRuntimeWithConfig(t, false, presetCfg), false)
	presetOpts.applyPreset(out)
	return out
}

// NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck creates a
// SingleChainMultiNode target with no proposer/challenger and no sequencer↔verifier
// P2P links, and without initial sync checks. The verifier's only data source is
// L1 derivation. Skipping the challenger avoids requiring cannon prestate
// artifacts. Intended for consensus-only verifier tests (e.g. op-con-node).
func NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t devtest.T, opts ...Option) *SingleChainMultiNode {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	out := singleChainMultiNodeFromRuntime(t, sysgo.NewSingleChainMultiNodeNoFaultProofsRuntimeWithConfig(t, false, presetCfg), false)
	presetOpts.applyPreset(out)
	return out
}

// NewSingleChainMultiNodeNoFaultProofsWithP2PWithoutCheck is the with-P2P
// counterpart of the above: the sequencer publishes unsafe blocks over gossip and
// the verifier joins the network. With the default op-node verifier this peers
// over CL P2P; with DEVSTACK_L2CL_KIND=op-con-node the verifier (which has no
// built-in P2P) is fronted by the op-conp2p gossip sidecar, which delegates each
// block's signature verdict back to op-con-node. No proposer/challenger (no
// cannon), no initial sync checks.
func NewSingleChainMultiNodeNoFaultProofsWithP2PWithoutCheck(t devtest.T, opts ...Option) *SingleChainMultiNode {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainMultiNodeNoFaultProofsWithP2PWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	out := singleChainMultiNodeFromRuntime(t, sysgo.NewSingleChainMultiNodeNoFaultProofsRuntimeWithConfig(t, true, presetCfg), false)
	presetOpts.applyPreset(out)
	return out
}

// NewSingleChainMultiNodeNoFaultProofsBareVerifierWithoutCheck creates a
// SingleChainMultiNode whose verifier has no unsafe source at all (no follow, no
// CL P2P, no sidecar): the test drives the verifier's unsafe chain via
// admin_postUnsafePayload. No proposer/challenger (no cannon), no initial sync
// checks. Used for unsafe-payload-queue / gap-handling tests.
func NewSingleChainMultiNodeNoFaultProofsBareVerifierWithoutCheck(t devtest.T, opts ...Option) *SingleChainMultiNode {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainMultiNodeNoFaultProofsBareVerifierWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	out := singleChainMultiNodeFromRuntime(t, sysgo.NewSingleChainMultiNodeNoFaultProofsBareVerifierRuntime(t, presetCfg), false)
	presetOpts.applyPreset(out)
	return out
}

type SingleChainMultiNodeWithTestSeq struct {
	SingleChainMultiNode

	TestSequencer *dsl.TestSequencer
}

// NewSingleChainMultiNodeNoFaultProofsBareVerifierWithTestSeqWithoutCheck is the
// bare-verifier (no follow source, no P2P, no sidecar) test-sequencer preset: the
// verifier derives only the safe chain from L1, with the test-sequencer driving
// (and able to reorg) L1. Used for L1-reorg → safe-re-derivation tests, where a
// follow/unsafe source would otherwise confound the safe-chain assertions.
func NewSingleChainMultiNodeNoFaultProofsBareVerifierWithTestSeqWithoutCheck(t devtest.T, opts ...Option) *SingleChainMultiNodeWithTestSeq {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainMultiNodeNoFaultProofsBareVerifierWithTestSeqWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	out := singleChainMultiNodeWithTestSeqFromRuntime(t, sysgo.NewSingleChainMultiNodeNoFaultProofsBareVerifierRuntime(t, presetCfg))
	presetOpts.applyPreset(out)
	return out
}

// NewSingleChainMultiNodeWithTestSeq creates a fresh
// SingleChainMultiNodeWithTestSeq target for the current test.
//
// The target is created from the runtime plus any additional preset options.
func NewSingleChainMultiNodeWithTestSeq(t devtest.T, opts ...Option) *SingleChainMultiNodeWithTestSeq {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainMultiNodeWithTestSeq", opts, minimalPresetSupportedOptionKinds)
	out := singleChainMultiNodeWithTestSeqFromRuntime(t, sysgo.NewSingleChainMultiNodeRuntimeWithConfig(t, true, presetCfg))
	presetOpts.applyPreset(out)
	return out
}
