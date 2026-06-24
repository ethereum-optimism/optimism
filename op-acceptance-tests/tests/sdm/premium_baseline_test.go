package sdm

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// premiumBinaryAvailable reports whether the op-reth-premium binary has been provided to the
// devstack. The premium client lives in a separate repo (optimism-premium), so it cannot be built
// from the monorepo's Rust workspace; it must be supplied as a pre-built binary
// (RUST_BINARY_PATH_OP_RETH_PREMIUM) or as a JIT build pointed at the premium checkout
// (RUST_SRC_DIR_OP_RETH_PREMIUM, with RUST_JIT_BUILD=1). When neither is set the baseline test
// skips rather than fails, so it is inert in CI that has no premium image.
func premiumBinaryAvailable() bool {
	return os.Getenv("RUST_BINARY_PATH_OP_RETH_PREMIUM") != "" ||
		os.Getenv("RUST_SRC_DIR_OP_RETH_PREMIUM") != ""
}

// TestPremiumSequencerProducesDerivableBlocks is the premium-sequencer *baseline* regression guard:
// it boots op-reth-premium as the sequencer EL (running its subblocks producer with SDM disabled,
// since Interop is not activated) alongside a stock op-reth verifier, and asserts the chain
// advances and the verifier derives it. It does NOT assert any SDM/0x7D behavior — premium does not
// produce PostExec blocks yet (production is parked). The point is to catch a premium build that
// regresses plain block production / follower derivation before the SDM-production tests can run
// against it.
//
// Run it by providing the premium binary, e.g.:
//
//	RUST_BINARY_PATH_OP_RETH_PREMIUM=/path/to/op-reth-premium \
//	  go test ./op-acceptance-tests/tests/sdm/ -run TestPremiumSequencerProducesDerivableBlocks
func TestPremiumSequencerProducesDerivableBlocks(gt *testing.T) {
	t := devtest.SerialT(gt)

	if !premiumBinaryAvailable() {
		t.Skip("op-reth-premium binary not provided; set RUST_BINARY_PATH_OP_RETH_PREMIUM (or RUST_SRC_DIR_OP_RETH_PREMIUM + RUST_JIT_BUILD=1)")
	}

	// op-node only: premium's subblocks producer is exercised on the standard sequencer path; keep
	// the CL selection consistent with the rest of the SDM suite (honors DEVSTACK_L2CL_KIND).
	clKind := sysgo.ResolveMixedL2CLKind()

	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer-op-reth-premium",
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpRethPremium,
				CLKind:      clKind,
				IsSequencer: true,
			},
			{
				ELKey:       "verifier-op-reth",
				CLKey:       "verifier",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      clKind,
				IsSequencer: false,
			},
		},
		// No InteropAtGenesis: SDM stays off, so premium produces ordinary blocks (no 0x7D). This
		// is a liveness/derivation baseline, not an SDM-production test.
	})

	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "premium baseline needs a sequencer and a verifier")

	var seqEL, verEL *dsl.L2ELNode
	for _, node := range frontends.Nodes {
		if node.Spec.IsSequencer {
			seqEL = node.EL
		} else {
			verEL = node.EL
		}
	}
	t.Require().NotNil(seqEL, "missing premium sequencer EL")
	t.Require().NotNil(verEL, "missing op-reth verifier EL")

	// The premium sequencer must build blocks, and the stock verifier must follow them: both
	// unsafe heads advancing past genesis proves premium produces derivable blocks as a drop-in.
	dsl.CheckAll(t,
		seqEL.AdvancedFn(eth.Unsafe, 5),
		verEL.AdvancedFn(eth.Unsafe, 5),
	)
}
