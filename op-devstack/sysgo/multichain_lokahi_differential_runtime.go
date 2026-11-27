package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// NewTwoL2LokahiDifferentialRuntime is [NewTwoL2LokahiRuntime] with an op-supernode that verifies
// the same two chains from the same L1, so that lokahi and op-supernode can be compared on
// identical inputs.
//
// The supernode sequences nothing and has no batcher. Its virtual nodes derive each chain from
// the batches the chain's batcher posts, into an op-reth of their own, with no p2p link to the
// chain's kona-node. Every chain records the supernode's route in SupernodeCL and its op-reth in
// SupernodeEL.
func NewTwoL2LokahiDifferentialRuntime(t devtest.T) *MultiChainRuntime {
	require := t.Require()
	runtime := NewTwoL2LokahiRuntime(t)
	chains := orderedRuntimeChains(runtime)
	depSet, ok := runtime.DependencySet.(*depset.StaticConfigDependencySet)
	require.True(ok, "the lokahi runtime must have a static dependency set, not %T", runtime.DependencySet)

	jwtPath := chains[0].EL.JWTPath()
	jwtSecret := readJWTSecretFromPath(t, jwtPath)
	l2Nets := make([]*L2Network, len(chains))
	l2ELs := make([]L2ELNode, len(chains))
	for i, chain := range chains {
		l2Nets[i] = chain.Network
		l2ELs[i] = startMixedOpRethNode(t, chain.Network, "supernode", jwtPath, jwtSecret, nil, ResolveMixedL2ELOpts(t)...)
	}
	activation := l2Nets[0].rollupCfg.Genesis.L2Time
	supernode, routes := startSharedSupernode(t, runtime.L1Network, runtime.L1EL, runtime.L1CL, l2Nets, l2ELs,
		depSet, &activation, 0, jwtSecret, false)
	for i, chain := range chains {
		chain.SupernodeCL = routes[i]
		chain.SupernodeEL = l2ELs[i]
	}
	runtime.Supernode = supernode
	return runtime
}
