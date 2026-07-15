package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// SingleChainOpConBootstrapRuntime is the Direct Sync BOOTSTRAP topology: an
// op-con-node SEQUENCER (signing, serving the payload websocket) starts alone
// and produces history, and the op-con-node verifier joins LATE — after the
// sequencer's bounded signed replay ring has already evicted the early chain.
//
// It differs from NewSingleChainOpConSequencerWSFollowRuntime in exactly one
// way: the verifier is NOT added at construction. The test decides when the
// verifier joins (AddWSFollowVerifier), so its cursor starts far below the
// ring's signed horizon and the whole bootstrap ladder is exercised: the
// below-horizon subscribe rejection (-32020), CATCH-UP range pulls through the
// unsigned cold tier (admitted only up to the producer's safe claim), chaining
// into the signed ring tail, and the CATCH-UP -> LIVE handoff.
//
// The sequencer is started immediately (there is no verifier to wait for, and
// unlike the gossip presets the late joiner is SUPPOSED to miss the early
// feed — recovering that history is the point). The batcher is constructed by
// the runtime; whether it runs from the start or launches stopped is the
// test's choice via PresetConfig.BatcherOptions (bss.CLIConfig.Stopped), which
// is what separates the "safe claim covers history" bootstrap test from the
// "unsigned-above-claim holds the cursor" pacing test.
type SingleChainOpConBootstrapRuntime struct {
	*SingleChainRuntime
	presetCfg PresetConfig
}

// NewSingleChainOpConBootstrapRuntime builds the late-join bootstrap runtime:
// signing op-con-node sequencer (ring depth via L2CLOpConPayloadRingBlocks in
// cfg.GlobalL2CLOptions), batcher constructed (honoring cfg.BatcherOptions),
// no verifier yet. Sequencing starts before this returns.
func NewSingleChainOpConBootstrapRuntime(t devtest.T, cfg PresetConfig) *SingleChainOpConBootstrapRuntime {
	t.Require().Equal(MixedL2CLOpCon, devstackL2CLKind(),
		"the op-con-node Direct Sync bootstrap preset requires DEVSTACK_L2CL_KIND=op-con-node")

	runtime := newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:   newDefaultSingleChainWorld,
		StartPrimary: startOpConSequencerPrimary,
		// The batcher is the safe-claim engine of the bootstrap protocol: the
		// producer's derived safe head is what admits unsigned cold-tier history
		// on the consumer. Tests control whether it runs from genesis or starts
		// stopped via PresetConfig.BatcherOptions.
		StartBatcher: true,
	})

	opcon, ok := runtime.L2CL.(*OpConNode)
	t.Require().True(ok, "primary sequencer must be op-con-node")
	t.Require().NotEmpty(opcon.SignedPayloadWS(), "op-con-node sequencer must serve the signed-payload ws")

	// Start producing immediately: the late-joining verifier is meant to find an
	// already-grown chain whose early blocks have been evicted from the ring.
	startOpConSequencer(t, opcon)

	return &SingleChainOpConBootstrapRuntime{
		SingleChainRuntime: runtime,
		presetCfg:          cfg,
	}
}

// AddWSFollowVerifier late-joins the op-con-node verifier "b": a fresh node
// (own op-reth at genesis) following the sequencer's payload websocket via
// --follow. Called mid-test, after the sequencer has produced past its ring
// depth, so the verifier's opening cursor lands below the signed horizon and
// its sync engine must bootstrap through the cold tier rather than replay the
// ring. The expected unsafe-block signer env was seeded by
// startOpConSequencerPrimary at runtime construction.
func (r *SingleChainOpConBootstrapRuntime) AddWSFollowVerifier(t devtest.T) *SingleChainNodeRuntime {
	opcon, ok := r.L2CL.(*OpConNode)
	t.Require().True(ok, "primary sequencer must be op-con-node")
	wsOpts := append(append([]L2CLOption{}, r.presetCfg.GlobalL2CLOptions...),
		L2CLOpConUnsafePayloadWS(opcon.SignedPayloadWS()))
	return addSingleChainOpNode(t, r.SingleChainRuntime, "b", false, "", wsOpts...)
}
