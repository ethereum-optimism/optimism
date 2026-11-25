package presets

import (
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2Lokahi is two interop chains whose kona-nodes follow one lokahi process for their safe
// and finalized labels.
type TwoL2Lokahi struct {
	TwoL2

	// TestSequencer controls L1: a test stops L1's own block production to build or reorg L1 by
	// hand.
	TestSequencer *dsl.TestSequencer

	// InteropActivationTime is when interop activates on both chains, the genesis time unless a
	// delay was asked for.
	InteropActivationTime uint64

	L2ELA *dsl.L2ELNode
	L2ELB *dsl.L2ELNode

	// Wallet and the funders create funded EOAs on each chain.
	Wallet  *dsl.HDWallet
	FunderA *dsl.FunderEOA
	FunderB *dsl.FunderEOA

	// Lokahi is the verifier process, which a test can stop and start.
	Lokahi *sysgo.Lokahi
}

// NewTwoL2Lokahi starts a [TwoL2Lokahi] system with interop active at genesis for the current
// test. With DEVSTACK_L2CL_KIND=op-reth each chain runs kona inside its op-reth process, with
// kona-node as a kona-node process. It skips the test under op-node.
func NewTwoL2Lokahi(t devtest.T) *TwoL2Lokahi {
	return NewTwoL2LokahiWithInteropDelay(t, 0)
}

// NewTwoL2LokahiWithInteropDelay is [NewTwoL2Lokahi] with interop activating delaySeconds after
// genesis.
func NewTwoL2LokahiWithInteropDelay(t devtest.T, delaySeconds uint64) *TwoL2Lokahi {
	skipKonaOnlyPresetOnOpNode(t)
	runtime := sysgo.NewTwoL2LokahiRuntimeWithInteropDelay(t, delaySeconds)
	inOpReth := sysgo.ResolveMixedL2CLKind(t) == sysgo.MixedL2CLOpReth
	for name, chain := range runtime.Chains {
		if inOpReth {
			cl, ok := chain.CL.(*sysgo.OpRethRollupNode)
			t.Require().Truef(ok, "chain %s must run kona inside op-reth, not %T", name, chain.CL)
			t.Require().Samef(chain.EL, cl.EL(), "chain %s must run kona inside its own EL process", name)
		} else {
			_, ok := chain.CL.(*sysgo.KonaNode)
			t.Require().Truef(ok, "chain %s must be driven by a kona-node process, not %T", name, chain.CL)
		}
	}
	t.Require().Equal("lokahi", filepath.Base(runtime.Lokahi.ExecPath()), "the verifier must be lokahi")
	t.Logger().Info("Two-chain lokahi system is up", "kona_in_op_reth", inOpReth,
		"lokahi", runtime.Lokahi.ExecPath(), "lokahi_rpc", runtime.Lokahi.UserRPC())

	twoL2, components := twoL2FromRuntime(t, runtime)
	testSequencer := newTestSequencerFrontend(t, runtime.TestSequencer.Name, runtime.TestSequencer.AdminRPC,
		runtime.TestSequencer.ControlRPC, runtime.TestSequencer.JWTSecret)
	preset := &TwoL2Lokahi{
		TwoL2:                 *twoL2,
		TestSequencer:         dsl.NewTestSequencer(testSequencer),
		InteropActivationTime: twoL2.L2A.Escape().RollupConfig().Genesis.L2Time + delaySeconds,
		L2ELA:                 dsl.NewL2ELNode(components.l2AEL),
		L2ELB:                 dsl.NewL2ELNode(components.l2BEL),
		Wallet:                dsl.NewRandomHDWallet(t, 30),
		Lokahi:                runtime.Lokahi,
	}
	preset.FunderA = newFunderEOA(t, runtime.Keys, preset.L2ELA, preset.Wallet)
	preset.FunderB = newFunderEOA(t, runtime.Keys, preset.L2ELB, preset.Wallet)
	return preset
}

// skipKonaOnlyPresetOnOpNode skips unless the selected CL runs kona. The lokahi presets call it
// because their chains always run kona, whose interop authority follower op-node lacks, so under
// op-node the test would repeat the kona-node job.
func skipKonaOnlyPresetOnOpNode(t devtest.T) {
	if !sysgo.ResolveMixedL2CLKind(t).RunsKona() {
		t.Skip("skipping on op-node: this preset starts only kona CLs whatever DEVSTACK_L2CL_KIND says")
	}
}
