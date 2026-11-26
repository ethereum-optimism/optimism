package sysgo

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

// The names of the nodes besides the sequencer that every chain of
// [NewTwoL2LokahiMultiNodeRuntime] runs, as keys of [MultiChainNodeRuntime.Followers].
const (
	// LokahiSourceNode derives its chain and is the node lokahi reads the chain from.
	LokahiSourceNode = "source"
	// LokahiFollowerNode derives its chain and is not read by lokahi.
	LokahiFollowerNode = "follower"
)

// NewTwoL2LokahiMultiNodeRuntime starts two chains with interop active at genesis, each run by
// three kona-nodes with their own op-reth, and one lokahi process verifying both:
//
//   - the sequencer, which also derives the chain, and whose blocks the batcher posts;
//   - the source, which derives the chain and records its safe heads, and which is the only node
//     lokahi reads;
//   - the follower, which derives the chain.
//
// The source and the follower are peered with the sequencer and import its unsafe blocks over
// p2p. All three follow lokahi for their safe and finalized labels. A test sequencer controls L1
// and builds blocks on each chain's sequencer.
//
// With DEVSTACK_L2CL_KIND=op-reth, each kona-node runs inside its op-reth process.
func NewTwoL2LokahiMultiNodeRuntime(t devtest.T) *MultiChainRuntime {
	require := t.Require()
	inOpReth := ResolveMixedL2CLKind(t) == MixedL2CLOpReth

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	jwtPath, jwtSecret := writeJWTSecret(t)
	var l1EL *L1Geth
	var l1CL *L1CLNode
	chainSpecs := []runtimeChainSpec{
		{Name: "l2a", ID: DefaultL2AID},
		{Name: "l2b", ID: DefaultL2BID},
	}
	wb, l1Net, l2Nets := buildMultiL2RuntimeWorld(
		t, keys, true, 0, "", chainSpecs, nil,
		func(l1Net *L1Network) (*L1Geth, *L1CLNode) {
			l1EL, l1CL = startInProcessL1WithClockConfig(t, l1Net, jwtPath, clock.SystemClock, PresetConfig{})
			return l1EL, l1CL
		},
	)
	depSet := wb.output.InteropDepSet
	require.NotNil(depSet, "op-deployer must generate the interop dependency set")
	depSetJSON, err := json.Marshal(depSet)
	require.NoError(err, "marshal interop dependency set")

	lokahi := newLokahi(t)
	startNode := func(l2Net *L2Network, name string, isSequencer bool, options ...L2CLOption) *SingleChainNodeRuntime {
		el := startMixedOpRethNode(t, l2Net, name, jwtPath, jwtSecret, nil)
		cl := startMixedKonaNode(t, keys, l1Net, l2Net, l1EL, l1CL, el, name, name, konaNodeStartConfig{
			InOpReth:            inOpReth,
			DependencySet:       depSet,
			L2CLConfig:          resolveKonaL2CLConfig(t, name, l2Net.ChainID(), isSequencer, "", options),
			InteropAuthorityRPC: lokahi.UserRPC(),
		})
		return &SingleChainNodeRuntime{Name: name, IsSequencer: isSequencer, EL: el, CL: cl}
	}

	chains := make(map[string]*MultiChainNodeRuntime, len(chainSpecs))
	lokahiChains := make([]LokahiChain, 0, len(chainSpecs))
	l2Targets := make([]l2TestSequencerTarget, 0, len(chainSpecs))
	for i, chainSpec := range chainSpecs {
		l2Net := l2Nets[i]
		sequencer := startNode(l2Net, "sequencer", true)
		safeDBPath := t.TempDirWithPrefix("safedb-" + NewComponentTarget(LokahiSourceNode, l2Net.ChainID()).String())
		source := startNode(l2Net, LokahiSourceNode, false, L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
			cfg.SafeDBPath = safeDBPath
		}))
		follower := startNode(l2Net, LokahiFollowerNode, false)
		for _, node := range []*SingleChainNodeRuntime{source, follower} {
			connectL2CLPeers(t, t.Logger(), sequencer.CL, node.CL)
			connectL2ELPeers(t, t.Logger(), sequencer.EL.UserRPC(), node.EL.UserRPC())
		}

		chains[chainSpec.Name] = &MultiChainNodeRuntime{
			Name:    chainSpec.Name,
			Network: l2Net,
			EL:      sequencer.EL,
			CL:      sequencer.CL,
			Batcher: startMinimalBatcher(t, keys, l2Net, l1EL, sequencer.CL, sequencer.EL),
			Followers: map[string]*SingleChainNodeRuntime{
				LokahiSourceNode:   source,
				LokahiFollowerNode: follower,
			},
		}
		lokahiChains = append(lokahiChains, LokahiChain{Network: l2Net, SourceCL: source.CL, SourceEL: source.EL})
		l2Targets = append(l2Targets, l2TestSequencerTarget{chainID: l2Net.ChainID(), l2EL: sequencer.EL, l2CL: sequencer.CL})
	}

	lokahi.configure(l1EL, depSetJSON, lokahiChains)
	lokahi.Start()
	t.Cleanup(lokahi.Stop)

	testSequencer := startTestSequencerForL2Chains(t, keys, "test-sequencer-lokahi-multinode", jwtPath, jwtSecret,
		l1Net, l1EL, l1CL, l2Targets)

	return &MultiChainRuntime{
		Keys:          keys,
		Migration:     newInteropMigrationState(wb),
		DependencySet: depSet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains:        chains,
		Lokahi:        lokahi,
		TestSequencer: newTestSequencerRuntime(testSequencer, ""),
	}
}
