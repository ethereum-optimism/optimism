package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// NewSingleSupernodeWithSyncTesterRuntime builds a single-chain runtime: real
// sequencer op-node + EL driving the chain, sync-tester service feeding a
// mocked EL, and a verifier-mode supernode VN wired to that mocked EL.
// Interop is always on; cfg.InteropActivationDelaySeconds picks at-genesis (0)
// or post-genesis activation.
func NewSingleSupernodeWithSyncTesterRuntime(t devtest.T) *MultiChainRuntime {
	return NewSingleSupernodeWithSyncTesterRuntimeWithConfig(t, PresetConfig{})
}

func NewSingleSupernodeWithSyncTesterRuntimeWithConfig(t devtest.T, cfg PresetConfig) *MultiChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	delaySeconds := cfg.InteropActivationDelaySeconds
	interopAtGenesis := delaySeconds == 0

	deployerOpts := []DeployerOption{
		WithDevFeatureEnabled(devfeatures.OptimismPortalInteropFlag),
	}
	if !interopAtGenesis {
		// Karst at genesis, Lagoon at offset — exercises the activation transition.
		deployerOpts = append(deployerOpts, func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtGenesis(opforks.Karst)
				l2Cfg.WithForkAtOffset(opforks.Lagoon, &delaySeconds)
			}
		})
	}
	deployerOpts = append(deployerOpts, cfg.DeployerOptions...)

	migration, l1Net, l2Net, depSet, _ := buildSingleChainWorldWithInteropAndState(t, keys, interopAtGenesis, cfg.LocalContractArtifactsPath, deployerOpts...)
	validateSimpleInteropPresetConfig(t, cfg, l2Net)

	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock()
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClockConfig(t, l1Net, jwtPath, l1Clock, cfg)

	l2EL := startSequencerEL(t, l2Net, jwtPath, jwtSecret, NewELNodeIdentity(0))
	l2CL := startL2CLNode(t, keys, l1Net, l2Net, l1EL, l1CL, l2EL, jwtSecret, l2CLNodeStartConfig{
		Key:           "sequencer",
		IsSequencer:   true,
		L2CLOptions:   cfg.GlobalL2CLOptions,
		DependencySet: depSet,
	})
	l2Batcher := startMinimalBatcher(t, keys, l2Net, l1EL, l2CL, l2EL, cfg.BatcherOptions...)

	syncTester := startSyncTesterService(t, map[eth.ChainID]string{
		l2Net.ChainID(): l2EL.UserRPC(),
	})
	syncTesterELCfg := DefaultSyncTesterELConfig()
	if len(cfg.GlobalSyncTesterELOptions) > 0 {
		target := NewComponentTarget("sync-tester-el", l2Net.ChainID())
		for _, opt := range cfg.GlobalSyncTesterELOptions {
			if opt == nil {
				continue
			}
			opt.Apply(t, target, syncTesterELCfg)
		}
	}
	syncTesterEL := startSyncTesterELNode(
		t,
		jwtPath,
		syncTester,
		NewComponentTarget("sync-tester-el", l2Net.ChainID()),
		syncTesterELCfg,
	)

	var depSetStatic *depset.StaticConfigDependencySet
	if depSet != nil {
		cast, ok := depSet.(*depset.StaticConfigDependencySet)
		require.True(ok, "expected static dependency set")
		depSetStatic = cast
	}
	if cfg.MessageExpiryWindow != nil && depSetStatic != nil {
		var overrideErr error
		depSetStatic, overrideErr = depset.NewStaticConfigDependencySetWithMessageExpiryOverride(
			depSetStatic.Dependencies(), *cfg.MessageExpiryWindow)
		require.NoError(overrideErr, "failed to override message expiry window")
	}

	verifierSyncMode := nodeSync.CLSync
	if cfg.SupernodeVerifierSyncMode != nil {
		verifierSyncMode = *cfg.SupernodeVerifierSyncMode
	}
	activationTimestamp := l2Net.rollupCfg.Genesis.L2Time + delaySeconds
	supernode, supernodeProxy := startSingleChainSharedSupernode(
		t, l1Net, l1EL, l1CL, l2Net, syncTesterEL, depSetStatic, jwtSecret, &activationTimestamp, false, verifierSyncMode,
	)
	// Peer the VN with the sequencer so unsafe payloads flow via P2P
	// (the two-L2 supernode runtime does the same for verifier-mode VNs).
	connectL2CLPeers(t, t.Logger(), l2CL, supernodeProxy)

	faucetService := startFaucets(t, keys, l1Net.ChainID(), l2Net.ChainID(), l1EL.UserRPC(), l2EL.UserRPC())

	var runtimeDepSet depset.DependencySet
	if depSetStatic != nil {
		runtimeDepSet = depSetStatic
	} else {
		runtimeDepSet = depSet
	}

	return &MultiChainRuntime{
		Keys:          keys,
		Migration:     migration,
		DependencySet: runtimeDepSet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains: map[string]*MultiChainNodeRuntime{
			"l2a": {
				Name:        "l2a",
				Network:     l2Net,
				EL:          l2EL,
				CL:          l2CL,
				SupernodeCL: supernodeProxy,
				Batcher:     l2Batcher,
			},
		},
		Supernode:     supernode,
		FaucetService: faucetService,
		TimeTravel:    timeTravelClock,
		DelaySeconds:  delaySeconds,
		SyncTester: &SyncTesterRuntime{
			Service: syncTester,
			EL:      syncTesterEL,
			CL:      supernodeProxy,
		},
	}
}
