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

// NewTwoL2Lokahi starts a [TwoL2Lokahi] system for the current test.
func NewTwoL2Lokahi(t devtest.T) *TwoL2Lokahi {
	runtime := sysgo.NewTwoL2LokahiRuntime(t)
	for name, chain := range runtime.Chains {
		_, isKona := chain.CL.(*sysgo.KonaNode)
		t.Require().Truef(isKona, "chain %s must be driven by kona-node", name)
	}
	t.Require().Equal("lokahi", filepath.Base(runtime.Lokahi.ExecPath()), "the verifier must be lokahi")
	t.Logger().Info("Two-chain lokahi system is up", "lokahi", runtime.Lokahi.ExecPath(), "lokahi_rpc", runtime.Lokahi.UserRPC())

	twoL2, components := twoL2FromRuntime(t, runtime)
	return &TwoL2Lokahi{
		TwoL2:  *twoL2,
		L2ELA:  dsl.NewL2ELNode(components.l2AEL),
		L2ELB:  dsl.NewL2ELNode(components.l2BEL),
		Lokahi: runtime.Lokahi,
	}
}
