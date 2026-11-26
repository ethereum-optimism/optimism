package sysgo

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

// NewTwoL2LokahiRuntime starts two chains with interop active at genesis, each sequenced and
// derived by its own kona-node and op-reth, and one lokahi process verifying both.
//
// A test sequencer controls L1 only, so a test can stop L1's own block production and build or
// reorg L1 by hand.
//
// Each kona-node derives local-safe from L1 by itself, records it in a safe head database, and
// follows lokahi for its safe and finalized labels over JSON-RPC. lokahi reads both chains'
// local-safe blocks from those same nodes. No supernode runs.
//
// With DEVSTACK_L2CL_KIND=op-reth, each chain's kona-node runs inside its op-reth process
// instead of as a process of its own. Any other kind starts kona-node processes.
func NewTwoL2LokahiRuntime(t devtest.T) *MultiChainRuntime {
	return NewTwoL2LokahiRuntimeWithInteropDelay(t, 0)
}

// NewTwoL2LokahiRuntimeWithInteropDelay is [NewTwoL2LokahiRuntime] with interop activating
// delaySeconds after genesis, or at genesis when it is zero.
func NewTwoL2LokahiRuntimeWithInteropDelay(t devtest.T, delaySeconds uint64) *MultiChainRuntime {
	runtime := newMultiL2LokahiRuntime(t, true, delaySeconds, PresetConfig{}, lokahiChainsConfig{}, []runtimeChainSpec{
		{Name: "l2a", ID: DefaultL2AID},
		{Name: "l2b", ID: DefaultL2BID},
	})
	jwtPath := runtime.Chains["l2a"].EL.JWTPath()
	testSequencer := startTestSequencerForL2Chains(t, runtime.Keys, "test-sequencer-lokahi-l1", jwtPath,
		readJWTSecretFromPath(t, jwtPath), runtime.L1Network, runtime.L1EL, runtime.L1CL, nil)
	runtime.TestSequencer = newTestSequencerRuntime(testSequencer, "")
	return runtime
}

// lokahiChainsConfig says how [startLokahiChains] starts the chains.
type lokahiChainsConfig struct {
	// testControl has lokahi serve its test control, which the supernode handle pauses and
	// resumes verification through.
	testControl bool
	// perChainProposer starts an op-proposer per chain on the chain's node, as the supernode
	// runtime does on each chain's supernode route.
	perChainProposer bool
}

// newMultiL2LokahiRuntime deploys chainSpecs with interop activating delaySeconds after genesis
// when enableInterop is set, and starts the chains as [startLokahiChains] does.
func newMultiL2LokahiRuntime(
	t devtest.T,
	enableInterop bool,
	delaySeconds uint64,
	cfg PresetConfig,
	chainsCfg lokahiChainsConfig,
	chainSpecs []runtimeChainSpec,
) *MultiChainRuntime {
	require := t.Require()
	require.NotEmpty(chainSpecs, "multi-chain runtime needs at least one L2 chain")

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock()
		l1Clock = timeTravelClock
	}
	require.NoError(checkL2ToL2MessageExpiryPeriod(cfg, enableInterop, delaySeconds))
	deployerOpts := cfg.DeployerOptions
	if cfg.L2ToL2MessageExpiryPeriod != 0 {
		deployerOpts = append(slices.Clone(deployerOpts), withL2ToL2MessageExpiryPeriod(cfg.L2ToL2MessageExpiryPeriod))
	}
	var l1EL *L1Geth
	var l1CL *L1CLNode
	wb, l1Net, l2Nets := buildMultiL2RuntimeWorld(
		t, keys, enableInterop, delaySeconds, cfg.LocalContractArtifactsPath,
		chainSpecs, genesisAnchorGameType(cfg),
		func(l1Net *L1Network) (*L1Geth, *L1CLNode) {
			l1EL, l1CL = startInProcessL1WithClockConfig(t, l1Net, jwtPath, l1Clock, cfg)
			return l1EL, l1CL
		}, deployerOpts...,
	)
	migration := newInteropMigrationState(wb)
	if cfg.PreGenesisSuperGame != nil {
		preparePreGenesisSuperGame(t, keys, wb, l1Net, l1EL, migration, cfg.PreGenesisSuperGame, l2Nets...)
	}

	depSet := wb.output.InteropDepSet
	if enableInterop {
		require.NotNil(depSet, "op-deployer must generate the interop dependency set")
	}
	if cfg.MessageExpiryWindow != nil && depSet != nil {
		depSet, err = depset.NewStaticConfigDependencySetWithMessageExpiryOverride(
			depSet.Dependencies(), *cfg.MessageExpiryWindow)
		require.NoError(err, "failed to override message expiry window")
	}
	runtime := &MultiChainRuntime{
		Keys:         keys,
		Migration:    migration,
		L1Network:    l1Net,
		L1EL:         l1EL,
		L1CL:         l1CL,
		TimeTravel:   timeTravelClock,
		DelaySeconds: delaySeconds,
	}
	if depSet != nil {
		runtime.DependencySet = depSet
	}
	startLokahiChains(t, runtime, jwtPath, jwtSecret, cfg, chainsCfg, chainSpecs, l2Nets, depSet)
	return runtime
}

// startLokahiChains starts, per chain, op-reth, a sequencing kona-node and a batcher, and fills
// the runtime's chains. With a dependency set it also starts lokahi, which each kona-node follows
// for its safe and finalized labels and which reads each chain's local-safe blocks from that same
// node. Without one, interop is not deployed and no lokahi runs.
//
// Each kona-node derives for itself and records its local-safe heads in a safe head database,
// which lokahi reads through superroot_atTimestamp. It also stands in for the supernode route of
// its chain, as in the supernode presets where the supernode's virtual node sequences.
func startLokahiChains(
	t devtest.T,
	runtime *MultiChainRuntime,
	jwtPath string,
	jwtSecret [32]byte,
	cfg PresetConfig,
	chainsCfg lokahiChainsConfig,
	chainSpecs []runtimeChainSpec,
	l2Nets []*L2Network,
	depSet *depset.StaticConfigDependencySet,
) {
	require := t.Require()
	require.Len(l2Nets, len(chainSpecs), "one network per chain spec")
	inOpReth := ResolveMixedL2CLKind(t) == MixedL2CLOpReth

	var lokahi *Lokahi
	var runtimeDepSet depset.DependencySet
	if depSet != nil {
		runtimeDepSet = depSet
		lokahi = newLokahi(t)
		lokahi.testControl = chainsCfg.testControl
	}

	els := make([]*OpReth, len(l2Nets))
	var interopFilter *InteropFilter
	if cfg.UseInteropFilter {
		filterProxy := tcpproxy.New(t.Logger().New("proxy", "interop-filter"))
		require.NoError(filterProxy.Start())
		t.Cleanup(func() { filterProxy.Close() })
		filterRPC := "http://" + filterProxy.Addr()
		rollupConfigs := make(map[eth.ChainID]*rollup.Config, len(l2Nets))
		l2RPCs := make([]string, len(l2Nets))
		for i, l2Net := range l2Nets {
			els[i] = startMixedOpRethNodeWithInteropURL(t, l2Net, "sequencer", jwtPath, jwtSecret, nil, filterRPC,
				ResolveMixedL2ELOpts(t)...)
			rollupConfigs[l2Net.ChainID()] = l2Net.RollupConfig()
			l2RPCs[i] = els[i].UserRPC()
		}
		interopFilter = startInteropFilter(t, "interop-filter", l2RPCs, rollupConfigs)
		filterProxy.SetUpstream(ProxyAddr(require, interopFilter.HTTPEndpoint()))
	} else {
		for i, l2Net := range l2Nets {
			els[i] = startSequencerEL(t, l2Net, jwtPath, jwtSecret, ResolveMixedL2ELOpts(t)...)
		}
	}

	runtime.Chains = make(map[string]*MultiChainNodeRuntime, len(chainSpecs))
	lokahiChains := make([]LokahiChain, 0, len(chainSpecs))
	for i, chainSpec := range chainSpecs {
		l2Net := l2Nets[i]
		el := els[i]
		safeDBPath := t.TempDirWithPrefix("safedb-" + NewComponentTarget("sequencer", l2Net.ChainID()).String())
		startCfg := konaNodeStartConfig{
			InOpReth:      inOpReth,
			DependencySet: runtimeDepSet,
			L2CLConfig: resolveKonaL2CLConfig(t, "sequencer", l2Net.ChainID(), true, "", append([]L2CLOption{
				L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, c *L2CLConfig) {
					c.SafeDBPath = safeDBPath
				}),
			}, cfg.GlobalL2CLOptions...)),
		}
		if lokahi != nil {
			startCfg.InteropAuthorityRPC = lokahi.UserRPC()
		}
		cl := startMixedKonaNode(t, runtime.Keys, runtime.L1Network, l2Net, runtime.L1EL, runtime.L1CL, el,
			"sequencer", "sequencer", startCfg)
		batcher := startMinimalBatcher(t, runtime.Keys, l2Net, runtime.L1EL, cl, el, cfg.BatcherOptions...)
		chain := &MultiChainNodeRuntime{
			Name:        chainSpec.Name,
			Network:     l2Net,
			EL:          el,
			CL:          cl,
			SupernodeCL: cl,
			SupernodeEL: el,
			Batcher:     batcher,
		}
		if chainsCfg.perChainProposer {
			chain.Proposer = startMinimalProposer(t, runtime.Keys, l2Net, runtime.L1EL, cl)
		}
		runtime.Chains[chainSpec.Name] = chain
		lokahiChains = append(lokahiChains, LokahiChain{Network: l2Net, SourceCL: cl, SourceEL: el})
	}

	if lokahi != nil {
		depSetJSON, err := json.Marshal(depSet)
		require.NoError(err, "marshal interop dependency set")
		lokahi.configure(runtime.L1EL, depSetJSON, lokahiChains)
		lokahi.Start()
		t.Cleanup(lokahi.Stop)
		runtime.Lokahi = lokahi
	}
	if interopFilter != nil {
		interopFilter.WaitForReady(t, 120*time.Second)
		runtime.InteropFilter = interopFilter
	}
}

// newSingleChainLokahiRuntime is the lokahi backend of the single-chain supernode runtime: the
// same deployment, with the chain's kona-node following lokahi in place of op-supernode.
func newSingleChainLokahiRuntime(t devtest.T, lagoonAtGenesis bool, cfg PresetConfig) *MultiChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock()
		l1Clock = timeTravelClock
	}
	var l1EL *L1Geth
	var l1CL *L1CLNode
	migration, l1Net, l2Net, depSet := buildSingleChainWorld(t, keys, lagoonAtGenesis, cfg.LocalContractArtifactsPath, genesisAnchorGameType(cfg),
		func(l1Net *L1Network) (*L1Geth, *L1CLNode) {
			l1EL, l1CL = startInProcessL1WithClockConfig(t, l1Net, jwtPath, l1Clock, cfg)
			return l1EL, l1CL
		}, cfg.DeployerOptions...)
	validateSimpleInteropPresetConfig(t, cfg, l2Net)
	if cfg.MessageExpiryWindow != nil {
		depSet, err = depset.NewStaticConfigDependencySetWithMessageExpiryOverride(
			depSet.Dependencies(), *cfg.MessageExpiryWindow)
		require.NoError(err, "failed to override message expiry window")
	}

	runtime := &MultiChainRuntime{
		Keys:          keys,
		Migration:     migration,
		DependencySet: depSet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		TimeTravel:    timeTravelClock,
	}
	startLokahiChains(t, runtime, jwtPath, jwtSecret, cfg, lokahiChainsConfig{testControl: true},
		[]runtimeChainSpec{{Name: "l2a", ID: l2Net.ChainID()}}, []*L2Network{l2Net}, depSet)
	return runtime
}
