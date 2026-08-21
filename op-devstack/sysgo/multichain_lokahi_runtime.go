package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

// LokahiRuntime is a two-chain world whose verifier is one lokahi process.
//
// It is not the shared-supernode runtime with a different consensus layer dropped in: those
// presets reach into the supernode through stack.SupernodeTestControl, which an
// out-of-process implementation cannot satisfy (see failLokahiUnsupportedByPresets). This
// runtime asks only for what an out-of-process supernode can do — host N chains, answer per
// chain, and keep one chain's failure to itself — and is what the lokahi component is
// exercised against.
//
// Each chain is produced the production way, by its own op-node sequencer and batcher over
// its own execution layer, and is then verified by lokahi from L1 over a second execution
// layer. Nothing joins the two sides but L1, so a chain's safe head advancing under lokahi
// means lokahi derived it rather than being handed it.
type LokahiRuntime struct {
	Keys devkeys.Keys

	L1Network *L1Network
	L1EL      *L1Geth
	L1CL      *L1CLNode

	// Chains are the hosted chains, in the order lokahi was given them.
	Chains []*LokahiChainRuntime

	// Lokahi is the supernode verifying every chain.
	Lokahi *LokahiSupernode
}

// LokahiChainRuntime is one chain of a LokahiRuntime.
type LokahiChainRuntime struct {
	Name    string
	Network *L2Network

	// SequencerEL, SequencerCL and Batcher produce the chain and publish it to L1.
	SequencerEL L2ELNode
	SequencerCL L2CLNode
	Batcher     *L2Batcher

	// VerifierEL is the execution layer lokahi advances for this chain. Stopping it is how a
	// test takes one chain's execution layer away without touching the other's.
	VerifierEL L2ELNode

	// CL is lokahi's endpoint for this chain.
	CL L2CLNode
}

// NewTwoL2LokahiRuntime brings up two L2 chains and one lokahi process verifying both.
func NewTwoL2LokahiRuntime(t devtest.T) *LokahiRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	cfg := PresetConfig{}
	_, l1Net, l2ANet, l2BNet := buildTwoL2RuntimeWorld(t, keys, false, 0, cfg.LocalContractArtifactsPath)

	jwtPath, jwtSecret := writeJWTSecret(t)
	l1EL, l1CL := startInProcessL1WithClockConfig(t, l1Net, jwtPath, clock.SystemClock, cfg)

	// Ordered, not a map: Chains[i] and the chain lokahi was given at index i have to be the
	// same chain, and a map's iteration order would decide that per run.
	networks := []struct {
		name  string
		l2Net *L2Network
	}{{"l2a", l2ANet}, {"l2b", l2BNet}}

	chains := make([]*LokahiChainRuntime, 0, len(networks))
	hosted := make([]lokahiSupernodeChain, 0, len(networks))
	for _, network := range networks {
		chain := startLokahiChain(
			t, keys, l1Net, l1EL, l1CL, network.l2Net, network.name, jwtPath, jwtSecret)
		chains = append(chains, chain)
		hosted = append(hosted, lokahiSupernodeChain{net: network.l2Net, el: chain.VerifierEL})
	}

	lokahi := startLokahiSupernode(t, lokahiSupernodeConfig{
		l1Net:        l1Net,
		l1ELRPC:      l1EL.UserRPC(),
		l1BeaconAddr: l1CL.beaconHTTPAddr,
		chains:       hosted,
	})
	for i, chain := range chains {
		chain.CL = lokahi.ChainCL(i)
	}

	return &LokahiRuntime{
		Keys:      keys,
		L1Network: l1Net,
		L1EL:      l1EL,
		L1CL:      l1CL,
		Chains:    chains,
		Lokahi:    lokahi,
	}
}

// startLokahiChain brings up one chain's producing side and the execution layer lokahi will
// verify it on. The verifier EL is a separate node from the sequencer's: sharing one would
// let lokahi read a head the sequencer put there, and the test could not tell derivation
// from observation.
func startLokahiChain(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	l2Net *L2Network,
	name string,
	jwtPath string,
	jwtSecret [32]byte,
) *LokahiChainRuntime {
	elOpts := ResolveMixedL2ELOpts(t)
	sequencerEL := startL2ELForKey(t, l2Net, jwtPath, jwtSecret, "sequencer", NewELNodeIdentity(0), elOpts...)
	verifierEL := startL2ELForKey(t, l2Net, jwtPath, jwtSecret, "verifier", NewELNodeIdentity(0), elOpts...)

	sequencerCL := startL2CLNode(t, keys, l1Net, l2Net, l1EL, l1CL, sequencerEL, jwtSecret, l2CLNodeStartConfig{
		Key:         "sequencer",
		IsSequencer: true,
		NoDiscovery: true,
	})

	return &LokahiChainRuntime{
		Name:        name,
		Network:     l2Net,
		SequencerEL: sequencerEL,
		SequencerCL: sequencerCL,
		Batcher:     startMinimalBatcher(t, keys, l2Net, l1EL, sequencerCL, sequencerEL),
		VerifierEL:  verifierEL,
	}
}
