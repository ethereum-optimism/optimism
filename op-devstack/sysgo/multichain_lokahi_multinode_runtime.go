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
	return newTwoL2LokahiNodesRuntime(t, "test-sequencer-lokahi-multinode",
		func(startNode lokahiNodeStarter, l2Net *L2Network) lokahiNodes {
			sequencer := startNode(l2Net, "sequencer", true)
			source := startNode(l2Net, LokahiSourceNode, false, lokahiSafeDB(t, l2Net))
			follower := startNode(l2Net, LokahiFollowerNode, false)
			for _, node := range []*SingleChainNodeRuntime{source, follower} {
				connectL2CLPeers(t, t.Logger(), sequencer.CL, node.CL)
				connectL2ELPeers(t, t.Logger(), sequencer.EL.UserRPC(), node.EL.UserRPC())
			}
			return lokahiNodes{sequencer: sequencer, source: source, follower: follower}
		})
}

// lokahiNodes are the nodes of one chain of a lokahi runtime with several nodes per chain.
type lokahiNodes struct {
	// sequencer builds the chain's blocks, which the batcher posts.
	sequencer *SingleChainNodeRuntime
	// source derives the chain, and is the node lokahi reads it from.
	source   *SingleChainNodeRuntime
	follower *SingleChainNodeRuntime
}

// lokahiNodeStarter starts a kona-node named name with its own op-reth on a chain. The node
// derives, and follows lokahi for its safe and finalized labels.
type lokahiNodeStarter func(l2Net *L2Network, name string, isSequencer bool,
	options ...L2CLOption) *SingleChainNodeRuntime

// lokahiSafeDB is the option that makes a node on l2Net record its safe heads, which lokahi reads
// from a source.
func lokahiSafeDB(t devtest.T, l2Net *L2Network) L2CLOption {
	path := t.TempDirWithPrefix("safedb-" + NewComponentTarget(LokahiSourceNode, l2Net.ChainID()).String())
	return L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.SafeDBPath = path
	})
}

// newTwoL2LokahiNodesRuntime starts two chains with interop active at genesis, the nodes
// startChain starts on each, and one lokahi process reading each chain's source. The batcher and
// a test sequencer, which controls L1, use each chain's sequencer.
func newTwoL2LokahiNodesRuntime(t devtest.T, testSequencerName string,
	startChain func(startNode lokahiNodeStarter, l2Net *L2Network) lokahiNodes) *MultiChainRuntime {
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
	startNode := func(l2Net *L2Network, name string, isSequencer bool,
		options ...L2CLOption) *SingleChainNodeRuntime {
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
		nodes := startChain(startNode, l2Net)
		sequencer := nodes.sequencer
		chains[chainSpec.Name] = &MultiChainNodeRuntime{
			Name:    chainSpec.Name,
			Network: l2Net,
			EL:      sequencer.EL,
			CL:      sequencer.CL,
			Batcher: startMinimalBatcher(t, keys, l2Net, l1EL, sequencer.CL, sequencer.EL),
			Followers: map[string]*SingleChainNodeRuntime{
				LokahiSourceNode:   nodes.source,
				LokahiFollowerNode: nodes.follower,
			},
		}
		lokahiChains = append(lokahiChains, LokahiChain{Network: l2Net, SourceCL: nodes.source.CL, SourceEL: nodes.source.EL})
		l2Targets = append(l2Targets, l2TestSequencerTarget{chainID: l2Net.ChainID(), l2EL: sequencer.EL, l2CL: sequencer.CL})
	}

	lokahi.configure(l1EL, depSetJSON, lokahiChains)
	lokahi.Start()
	t.Cleanup(lokahi.Stop)

	testSequencer := startTestSequencerForL2Chains(t, keys, testSequencerName, jwtPath, jwtSecret,
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
