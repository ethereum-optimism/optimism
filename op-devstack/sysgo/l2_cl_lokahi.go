package sysgo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// lokahiSupernodeChain is one chain the supernode must host: the L2 network whose rollup
// config drives derivation, and the EL that chain's virtual node advances.
type lokahiSupernodeChain struct {
	net *L2Network
	el  L2ELNode
}

// lokahiSupernodeConfig is everything an out-of-process supernode needs in order to stand
// in for the in-process Go op-supernode. It deliberately mirrors the inputs
// startTwoL2SharedSupernode and startSingleChainSharedSupernode turn into an
// snconfig.CLIConfig plus one op-node config per chain, so the launch path below has the
// same information available and the two implementations cannot drift apart in what they
// are given.
type lokahiSupernodeConfig struct {
	// l1Net supplies the L1 chain config the supernode validates L1 blocks against.
	l1Net *L1Network
	// l1ELRPC and l1BeaconAddr are the L1 data sources shared by every hosted chain.
	l1ELRPC      string
	l1BeaconAddr string
	// chains are the L2 chains to host, in the order the caller wants them routed.
	chains []lokahiSupernodeChain
	// jwtSecret authenticates the engine API connection to each chain's EL.
	jwtSecret [32]byte
	// depSet is the interop dependency set, or nil when interop is not configured.
	depSet *depset.StaticConfigDependencySet
	// interopActivationTimestamp enables the interop activity at the given timestamp; nil
	// leaves interop off.
	interopActivationTimestamp *uint64
	// sequencerEnabled runs the hosted virtual nodes as sequencers rather than verifiers.
	sequencerEnabled bool
}

// describe renders the requested configuration for diagnostics. Every field is included:
// until lokahi can host a chain, this summary is the only consumer of the config, and it
// is what tells a reader of a failed run exactly what the seam was asked to start.
func (c lokahiSupernodeConfig) describe() string {
	chains := make([]string, 0, len(c.chains))
	for _, chain := range c.chains {
		chains = append(chains, fmt.Sprintf("%s(engine=%s)", chain.net.ChainID(), chain.el.EngineRPC()))
	}

	interop := "off"
	if c.interopActivationTimestamp != nil {
		interop = "at " + strconv.FormatUint(*c.interopActivationTimestamp, 10)
	}

	depSetChains := 0
	if c.depSet != nil {
		depSetChains = len(c.depSet.Chains())
	}

	l1ChainID := "unknown"
	if c.l1Net != nil && c.l1Net.genesis != nil && c.l1Net.genesis.Config != nil {
		l1ChainID = c.l1Net.genesis.Config.ChainID.String()
	}

	return fmt.Sprintf(
		"chains=[%s] l1=%s(chainID=%s) l1Beacon=%s depSetChains=%d interop=%s sequencer=%t engineJWT=%t",
		strings.Join(chains, " "), c.l1ELRPC, l1ChainID, c.l1BeaconAddr,
		depSetChains, interop, c.sequencerEnabled, c.jwtSecret != [32]byte{},
	)
}

// startLokahiSupernode runs lokahi as the shared multi-chain consensus layer, in place of
// the in-process Go op-supernode.
//
// It does not start anything yet, and it never returns: lokahi (rust/lokahi) is still a
// CLI skeleton. Its only command parses global flags and prints a greeting
// (rust/lokahi/src/cli.rs:22), so it binds no RPC and hosts no chain. Two concrete gaps
// have to close before this function can launch a process:
//
//  1. lokahi must serve the per-chain RPC routes the presets address as
//     <base>/<chainID> (see waitForSupernodeRoute in multichain_supernode_runtime.go),
//     answering optimism_rollupConfig and the rest of the L2 CL surface.
//  2. stack.SupernodeTestControl (op-devstack/stack/supernode.go:19) has to stop handing
//     tests an in-process pointer. Its InteropActivity() returns *interop.Interop, which
//     the DSL calls PauseAt/Resume on (op-devstack/dsl/supernode.go:117), and no
//     out-of-process implementation can satisfy that signature. It needs an RPC-shaped
//     replacement first.
//
// The selection seam lands regardless, so it is written and reviewed once instead of being
// re-derived when lokahi grows a node: this function is the single place that becomes a
// real process launch, exactly as startMixedKonaNode (mixed_runtime.go) is for the
// single-chain CL switch. Until then, selecting lokahi fails immediately and says why,
// rather than quietly starting a Go supernode the caller did not ask for and reporting the
// result as a lokahi run.
func startLokahiSupernode(t devtest.T, cfg lokahiSupernodeConfig) {
	t.Require().FailNowf("lokahi supernode not implemented",
		"%s=%s selected, but lokahi cannot host a chain yet: rust/lokahi is a CLI skeleton "+
			"that prints a greeting and exits. Requested: %s. "+
			"Unset %s (or set it to %q) to run the in-process Go op-supernode.",
		devstackSupernodeKindEnv, SupernodeLokahi, cfg.describe(),
		devstackSupernodeKindEnv, SupernodeOpSupernode)
}
