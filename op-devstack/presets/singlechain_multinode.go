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

type SingleChainMultiNodeWithTestSeq struct {
	SingleChainMultiNode

	TestSequencer *dsl.TestSequencer
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
