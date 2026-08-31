package presets

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/poller"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// PrivateInterop is what a test holds ON TOP of the ordinary two-L2 interop surfaces when chain B is
// a private interop pair.
//
// Everything a normal interop test uses is unchanged and in its usual place: L2ELB is the private
// chain's sequencer EL, so transactions, funding and receipts work exactly as before, and
// L2BSupernodeCL / L2BSupernodeEL are the rendering, which is what the supernode judges. The fields
// here are only the things that have no equivalent on an ordinary chain.
type PrivateInterop struct {
	// RenderingRollupConfig is the PUBLIC half's rollup config. It shares the private chain's ID,
	// block time, genesis number and genesis timestamp, and differs in its genesis hash -- which is
	// what makes them two chains rather than one.
	RenderingRollupConfig *rollup.Config

	// Invariant is the standing rendering-invariant checker, or nil when a test opted out with
	// sysgo.WithoutRenderingInvariantCheck. Tests assert progress through it.
	Invariant *poller.RenderingInvariant

	runtime *sysgo.MultiChainRuntime
}

// FollowSource is the URL the private chain's stock op-node polls for its own safe head: the
// supernode's `<base>/<chainID>/claimed` sibling route.
//
// There is no withhold latch to ask about any more, and no sidecar holding one. Under
// snap-to-commitment the module serves claims verbatim and a diverged sequencer force-resets onto
// the publicly claimed chain -- the claim is the operator's binding statement, so moving to it is
// recovery. "The chain diverged from its claims" is a monitoring alert now; a claim naming a block
// that exists nowhere is a loud unfindable-hash sync stall, which shows up in a test as a chain that
// stopped rather than as a flag to read.
func (p *PrivateInterop) FollowSource() string {
	return p.runtime.PrivateInterop.FollowSource
}

// Operator is the EOA that signs every replay transaction and every range claim on the rendering.
// Deleted once as dead code; op-up's endpoint printout made it live again the same night.
func (p *PrivateInterop) Operator() common.Address {
	return p.runtime.PrivateInterop.Operator
}

// twoL2PrivateInteropFromRuntime assembles the preset for a world whose chain B is a pair.
//
// It is the ordinary two-L2 interop assembler plus two things: the extra handles above, and the
// standing invariant checker. Everything else a test sees is built by exactly the same code as for
// an ordinary pair of public chains, which is what "plug-in replacement" has to mean if it means
// anything.
func twoL2PrivateInteropFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2SupernodeInterop {
	t.Require().NotNil(runtime.PrivateInterop, "the runtime was not built as a private interop pair")
	preset := twoL2SupernodeInteropFromRuntime(t, runtime)

	pi := &PrivateInterop{
		RenderingRollupConfig: runtime.PrivateInterop.Rendering.RollupConfig(),
		runtime:               runtime,
	}
	if !runtime.PrivateInterop.Config.SkipRenderingInvariant {
		pi.Invariant = poller.StartRenderingInvariant(
			preset.L2ELB, preset.L2BSupernodeEL,
			preset.L2BCL, preset.L2BSupernodeCL,
		)
	}

	// The identifier resolver. It is registered here, at the one place that holds both halves of
	// the pair, and torn down with the test -- so a suite that never builds a pair has no resolver
	// registered and every stock chain's identifiers are minted exactly as they were before this
	// existed. See private_interop_resolver.go.
	resolver := &privateInteropResolver{
		t:           t,
		privateEL:   preset.L2ELB,
		renderingEL: preset.L2BSupernodeEL,
		renderingCL: preset.L2BSupernodeCL,
		emitters:    render.NewEmitterSet(runtime.PrivateInterop.ExtraEmitters...),
		timeout:     privateInteropPositionTimeout,
	}
	t.Cleanup(txintent.RegisterPositionResolver(preset.L2ELB.ChainID(), resolver))

	preset.PrivateInterop = pi
	return preset
}
