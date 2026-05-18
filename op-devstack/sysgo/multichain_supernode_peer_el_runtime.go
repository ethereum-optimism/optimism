package sysgo

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// NewTwoL2SupernodeInteropPeerELRuntimeWithConfig builds a two-L2 interop
// runtime where, per chain, the sequencer is a sibling op-node + op-geth pair
// independent of the supernode, and the supernode's virtual node drives a
// separate op-geth that catches up via consensus-layer reqresp sync from the
// sibling sequencer CL.
//
// In production, the equivalent topology would have the supernode VN running
// execution-layer sync against a sibling EL over devp2p. The in-process
// devstack op-geth used here does not run a real EL sync protocol between
// sibling instances, so block transport flows through the CL: the supernode
// VN fetches unsafe payloads from the sibling sequencer CL via P2P reqresp
// and inserts them into its EL via the engine API. The supernode VN
// re-derives its own safeDB from L1.
//
// This topology lets a test wipe the supernode's data dir together with the
// supernode-fronted EL while the sibling sequencer keeps producing blocks:
// on restart, the wiped EL rebuilds chain state from the sibling via the
// supernode VN's CL reqresp + engine-API path, and the supernode VN
// cold-starts on top of the recovered engine.
//
// Returned MultiChainRuntime convention:
//   - Chains[<key>].EL / Chains[<key>].CL are the **sibling sequencer** pair
//     (so the batcher and proposer point at the live sequencer).
//   - Chains[<key>].Followers["supernode"] holds the supernode-fronted EL and
//     CL proxy — i.e. the wipe target for cold-start tests.
func NewTwoL2SupernodeInteropPeerELRuntimeWithConfig(t devtest.T, delaySeconds uint64, cfg PresetConfig) *MultiChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	wb, l1Net, l2ANet, l2BNet := buildTwoL2PeerELRuntimeWorld(t, keys, delaySeconds, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...)
	migration := newInteropMigrationState(wb)
	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock(100 * time.Millisecond)
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClockConfig(t, l1Net, jwtPath, l1Clock, cfg)

	activationTime := l2ANet.rollupCfg.Genesis.L2Time + delaySeconds
	interopActivationTimestamp := &activationTime

	var depSet *depset.StaticConfigDependencySet
	if wb.outFullCfgSet.DependencySet != nil {
		cast, ok := wb.outFullCfgSet.DependencySet.(*depset.StaticConfigDependencySet)
		require.True(ok, "expected static dependency set")
		depSet = cast
	}
	if cfg.MessageExpiryWindow != nil && depSet != nil {
		var overrideErr error
		depSet, overrideErr = depset.NewStaticConfigDependencySetWithMessageExpiryOverride(
			depSet.Dependencies(), *cfg.MessageExpiryWindow)
		require.NoError(overrideErr, "failed to override message expiry window")
	}

	// Per chain, start two ELs with distinct P2P identities so they can
	// devp2p-peer each other: a sequencer EL driven by the sibling CL, and
	// a supernode-fronted EL driven by the supernode VN. The supernode EL
	// is the one wiped by the cold-start tests; it recovers by syncing from
	// the sibling EL.
	sequencerELA := startL2ELForKey(t, l2ANet, jwtPath, jwtSecret, "sequencer", NewELNodeIdentity(0))
	supernodeELA := startL2ELForKey(t, l2ANet, jwtPath, jwtSecret, "supernode-el", NewELNodeIdentity(0))
	sequencerELB := startL2ELForKey(t, l2BNet, jwtPath, jwtSecret, "sequencer", NewELNodeIdentity(0))
	supernodeELB := startL2ELForKey(t, l2BNet, jwtPath, jwtSecret, "supernode-el", NewELNodeIdentity(0))

	connectL2ELPeers(t, t.Logger(), sequencerELA.UserRPC(), supernodeELA.UserRPC(), true)
	connectL2ELPeers(t, t.Logger(), sequencerELB.UserRPC(), supernodeELB.UserRPC(), true)

	// Sibling sequencer CLs: independent of the supernode, drive block
	// production for their chain, batcher and proposer hook into these.
	sequencerCLA := startL2CLNode(t, keys, l1Net, l2ANet, l1EL, l1CL, sequencerELA, jwtSecret, l2CLNodeStartConfig{
		Key:           "sequencer",
		IsSequencer:   true,
		NoDiscovery:   true,
		EnableReqResp: true,
		UseReqResp:    true,
		DependencySet: depSet,
	})
	sequencerCLB := startL2CLNode(t, keys, l1Net, l2BNet, l1EL, l1CL, sequencerELB, jwtSecret, l2CLNodeStartConfig{
		Key:           "sequencer",
		IsSequencer:   true,
		NoDiscovery:   true,
		EnableReqResp: true,
		UseReqResp:    true,
		DependencySet: depSet,
	})

	// Supernode VNs configured as execution-layer-sync followers of the
	// sibling sequencer CLs. The VN drives the supernode-fronted EL via
	// the engine API while the EL itself snap/full-syncs over devp2p from
	// the sibling sequencer EL.
	supernode, l2ACLProxy, l2BCLProxy := startTwoL2SharedSupernodeWithMode(
		t,
		l1Net,
		l1EL,
		l1CL,
		l2ANet,
		supernodeELA,
		supernodeVNMode{PeerCLSyncFollowSourceRPC: sequencerCLA.UserRPC()},
		l2BNet,
		supernodeELB,
		supernodeVNMode{PeerCLSyncFollowSourceRPC: sequencerCLB.UserRPC()},
		depSet,
		interopActivationTimestamp,
		cfg.InteropLogBackfillDepth,
		jwtSecret,
	)

	// CL P2P-peer the supernode VNs to the sibling sequencer CLs so block
	// gossip flows promptly even before EL sync catches up.
	connectL2CLPeers(t, t.Logger(), sequencerCLA, l2ACLProxy)
	connectL2CLPeers(t, t.Logger(), sequencerCLB, l2BCLProxy)

	// Batcher and proposer hook into the sibling sequencer pair — that is
	// the canonical chain.EL / chain.CL for this preset.
	l2ABatcher := startMinimalBatcher(t, keys, l2ANet, l1EL, sequencerCLA, sequencerELA, cfg.BatcherOptions...)
	l2AProposer := startMinimalProposer(t, keys, l2ANet, l1EL, sequencerCLA)
	l2BBatcher := startMinimalBatcher(t, keys, l2BNet, l1EL, sequencerCLB, sequencerELB, cfg.BatcherOptions...)
	l2BProposer := startMinimalProposer(t, keys, l2BNet, l1EL, sequencerCLB)

	faucetService := startFaucetsForRPCs(t, keys, map[eth.ChainID]string{
		l1Net.ChainID():  l1EL.UserRPC(),
		l2ANet.ChainID(): sequencerELA.UserRPC(),
		l2BNet.ChainID(): sequencerELB.UserRPC(),
	})

	var runtimeDepSet depset.DependencySet
	if depSet != nil {
		runtimeDepSet = depSet
	} else {
		runtimeDepSet = wb.outFullCfgSet.DependencySet
	}

	runtime := &MultiChainRuntime{
		Keys:          keys,
		Migration:     migration,
		DependencySet: runtimeDepSet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains: map[string]*MultiChainNodeRuntime{
			"l2a": {
				Name:     "l2a",
				Network:  l2ANet,
				EL:       sequencerELA,
				CL:       sequencerCLA,
				Batcher:  l2ABatcher,
				Proposer: l2AProposer,
				Followers: map[string]*SingleChainNodeRuntime{
					"supernode": {
						Name:        "supernode-el",
						IsSequencer: false,
						EL:          supernodeELA,
						CL:          l2ACLProxy,
					},
				},
			},
			"l2b": {
				Name:     "l2b",
				Network:  l2BNet,
				EL:       sequencerELB,
				CL:       sequencerCLB,
				Batcher:  l2BBatcher,
				Proposer: l2BProposer,
				Followers: map[string]*SingleChainNodeRuntime{
					"supernode": {
						Name:        "supernode-el",
						IsSequencer: false,
						EL:          supernodeELB,
						CL:          l2BCLProxy,
					},
				},
			},
		},
		Supernode:     supernode,
		FaucetService: faucetService,
		TimeTravel:    timeTravelClock,
		DelaySeconds:  delaySeconds,
	}
	attachTestSequencerToRuntime(t, runtime, "test-sequencer-2l2-peer-el")
	return runtime
}

func buildTwoL2PeerELRuntimeWorld(t devtest.T, keys devkeys.Keys, delaySeconds uint64, localContractArtifactsPath string, deployerOpts ...DeployerOption) (*worldBuilder, *L1Network, *L2Network, *L2Network) {
	wb := &worldBuilder{
		p:       t,
		logger:  t.Logger(),
		require: t.Require(),
		keys:    keys,
		builder: intentbuilder.New(),
	}

	applyConfigLocalContractSources(t, keys, wb.builder, localContractArtifactsPath)
	applyConfigCommons(t, keys, DefaultL1ID, wb.builder)
	applyConfigPrefundedL2(t, keys, DefaultL1ID, DefaultL2AID, wb.builder)
	applyConfigPrefundedL2(t, keys, DefaultL1ID, DefaultL2BID, wb.builder)

	deployerOpts = append([]DeployerOption{
		WithDevFeatureEnabled(devfeatures.OptimismPortalInteropFlag),
	}, deployerOpts...)
	for _, l2Cfg := range wb.builder.L2s() {
		if delaySeconds > 0 {
			l2Cfg.WithForkAtGenesis(opforks.Karst)
			l2Cfg.WithForkAtOffset(opforks.Interop, &delaySeconds)
		} else {
			l2Cfg.WithForkAtGenesis(opforks.Interop)
		}
	}
	applyConfigDeployerOptions(t, keys, wb.builder, deployerOpts)
	wb.Build()

	t.Require().Len(wb.l2Chains, 2, "expected exactly two L2 chains in TwoL2 peer-EL world")
	l1ID := eth.ChainIDFromUInt64(wb.output.AppliedIntent.L1ChainID)

	l1Net := &L1Network{
		name:      "l1",
		chainID:   l1ID,
		genesis:   wb.outL1Genesis,
		blockTime: 6,
	}
	l2ANet := l2NetworkFromWorldBuilder(t, wb, l1ID, DefaultL2AID, keys)
	l2BNet := l2NetworkFromWorldBuilder(t, wb, l1ID, DefaultL2BID, keys)
	return wb, l1Net, l2ANet, l2BNet
}
