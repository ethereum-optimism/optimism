package presets

import (
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TwoL2Lokahi is two chains with interop active at genesis whose kona-nodes follow one lokahi
// process for their safe and finalized labels.
type TwoL2Lokahi struct {
	TwoL2

	L2ELA *dsl.L2ELNode
	L2ELB *dsl.L2ELNode

	// Lokahi is the verifier process, which a test can stop and start.
	Lokahi *sysgo.Lokahi
}

// NewTwoL2Lokahi starts a [TwoL2Lokahi] system for the current test. With
// DEVSTACK_L2CL_KIND=op-reth each chain runs kona inside its op-reth process, otherwise as a
// kona-node process.
func NewTwoL2Lokahi(t devtest.T) *TwoL2Lokahi {
	runtime := sysgo.NewTwoL2LokahiRuntime(t)
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
	return &TwoL2Lokahi{
		TwoL2:  *twoL2,
		L2ELA:  dsl.NewL2ELNode(components.l2AEL),
		L2ELB:  dsl.NewL2ELNode(components.l2BEL),
		Lokahi: runtime.Lokahi,
	}
}
