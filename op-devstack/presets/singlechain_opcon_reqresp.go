package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// NewSingleChainOpConSequencerLateJoinReqRespWithoutCheck creates the
// late-joining-verifier req-resp preset: the op-con-node sequencer (L2CL)
// signs and produces preproducedBlocks with no verifier and no gossip route,
// then a stock op-node verifier (L2CLB, req-resp sync enabled) and the
// op-conp2p publish sidecar come up. The sidecar joins the sequencer's Direct
// Sync feed live (the pre-produced span is never gossiped) and serves P2P
// req-resp payloads-by-number from the sequencer's signed replay ring, so the
// verifier's only route to the missed span is req-resp reverse sync through
// the sidecar. Requires DEVSTACK_L2CL_KIND=op-con-node. No batcher/proposer/
// challenger and no initial sync checks.
func NewSingleChainOpConSequencerLateJoinReqRespWithoutCheck(t devtest.T, preproducedBlocks uint64, opts ...Option) *SingleChainMultiNode {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainOpConSequencerLateJoinReqRespWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	out := singleChainMultiNodeFromRuntime(t, sysgo.NewSingleChainOpConLateJoinReqRespRuntime(t, presetCfg, preproducedBlocks), false)
	presetOpts.applyPreset(out)
	return out
}

// SingleChainOpConRestartableSidecar is the op-con-node sequencer P2P preset
// (SingleChainMultiNode: op-con-node sequencer as L2CL, stock op-node verifier
// as L2CLB) plus a control handle for the publish sidecar, so a test can model
// a sidecar outage (Stop) and its recovery (Restart).
type SingleChainOpConRestartableSidecar struct {
	SingleChainMultiNode

	Sidecar *sysgo.OpConPublishSidecarControl
}

// NewSingleChainOpConSequencerP2PRestartableSidecarWithoutCheck creates the
// op-con-node sequencer P2P preset (identical topology to
// NewSingleChainOpConSequencerP2PWithoutCheck: signed publish via the
// op-conp2p sidecar to a stock op-node verifier, no batcher) with the sidecar
// exposed for stop/restart. Requires DEVSTACK_L2CL_KIND=op-con-node. No
// initial sync checks.
func NewSingleChainOpConSequencerP2PRestartableSidecarWithoutCheck(t devtest.T, opts ...Option) *SingleChainOpConRestartableSidecar {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainOpConSequencerP2PRestartableSidecarWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	runtime, control := sysgo.NewSingleChainOpConSequencerP2PRestartableSidecarRuntime(t, presetCfg)
	out := &SingleChainOpConRestartableSidecar{
		SingleChainMultiNode: *singleChainMultiNodeFromRuntime(t, runtime, false),
		Sidecar:              control,
	}
	presetOpts.applyPreset(out)
	return out
}
