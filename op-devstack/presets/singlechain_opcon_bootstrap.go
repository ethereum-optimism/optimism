package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// SingleChainOpConBootstrap is the Direct Sync late-join bootstrap preset: an
// op-con-node sequencer (signing, payload websocket, bounded replay ring) runs
// from construction, but the op-con-node verifier does NOT exist yet — the
// test adds it mid-flight with AddWSFollowVerifier, once the chain has grown
// past the ring depth. Until then L2ELB/L2CLB are nil.
//
// L2Batcher is available from the embedded Minimal (constructed by the
// runtime; start it stopped via WithBatcherOption + bss.CLIConfig.Stopped and
// resume it with L2Batcher.Start() to move the producer's safe claim).
type SingleChainOpConBootstrap struct {
	Minimal

	runtime *sysgo.SingleChainOpConBootstrapRuntime

	// L2ELB / L2CLB are the late-joining verifier's frontends, nil until
	// AddWSFollowVerifier is called.
	L2ELB *dsl.L2ELNode
	L2CLB *dsl.L2CLNode
}

// NewSingleChainOpConBootstrapWithoutCheck creates the late-join bootstrap
// preset: signing op-con-node sequencer + batcher, NO verifier yet. Requires
// DEVSTACK_L2CL_KIND=op-con-node. No proposer/challenger, no initial sync
// checks (there is nothing to sync-check until the verifier joins).
func NewSingleChainOpConBootstrapWithoutCheck(t devtest.T, opts ...Option) *SingleChainOpConBootstrap {
	presetCfg, presetOpts := collectSupportedPresetConfig(t, "NewSingleChainOpConBootstrapWithoutCheck", opts, minimalPresetSupportedOptionKinds)
	runtime := sysgo.NewSingleChainOpConBootstrapRuntime(t, presetCfg)
	minimal := minimalFromRuntime(t, runtime.SingleChainRuntime)
	out := &SingleChainOpConBootstrap{
		Minimal: *minimal,
		runtime: runtime,
	}
	presetOpts.applyPreset(out)
	return out
}

// AddWSFollowVerifier late-joins the op-con-node verifier "b" (fresh op-reth at
// genesis, --follow pointed at the sequencer's payload websocket) and wires its
// DSL frontends into L2ELB/L2CLB. The verifier's cursor opens at the bottom of
// the chain, far below the sequencer's signed replay ring, so its sync engine
// must bootstrap: below-horizon subscribe rejection, CATCH-UP through the
// unsigned cold tier (policed by the producer's safe claim), then LIVE.
func (s *SingleChainOpConBootstrap) AddWSFollowVerifier(t devtest.T) {
	t.Require().Nil(s.L2CLB, "verifier b already added")
	nodeB := s.runtime.AddWSFollowVerifier(t)

	l2ChainID := s.runtime.L2Network.ChainID()
	l2ELB := newL2ELFrontend(
		t,
		"b",
		l2ChainID,
		nodeB.EL.UserRPC(),
		nodeB.EL.EngineRPC(),
		nodeB.EL.JWTPath(),
		s.runtime.L2Network.RollupConfig(),
		nodeB.EL,
	)
	l2CLB := newL2CLFrontend(
		t,
		"b",
		l2ChainID,
		nodeB.CL.UserRPC(),
		nodeB.CL,
	)
	l2CLB.attachEL(l2ELB)
	l2Net, ok := s.L2Chain.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network")
	l2Net.AddL2ELNode(l2ELB)
	l2Net.AddL2CLNode(l2CLB)

	s.L2ELB = dsl.NewL2ELNode(l2ELB)
	s.L2CLB = dsl.NewL2CLNode(l2CLB)
}
