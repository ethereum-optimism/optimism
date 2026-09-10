package sysgo

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

type fakeExternalL2CL struct {
	starts int
	stops  int
	rpc    string
}

type fakeFactoryL2EL struct {
	rpc string
}

func (n *fakeFactoryL2EL) Start()            {}
func (n *fakeFactoryL2EL) Stop()             {}
func (n *fakeFactoryL2EL) UserRPC() string   { return n.rpc }
func (n *fakeFactoryL2EL) EngineRPC() string { return "http://127.0.0.1:8551" }
func (n *fakeFactoryL2EL) JWTPath() string   { return "/tmp/fake-jwt" }

func (n *fakeExternalL2CL) Start() { n.starts++ }
func (n *fakeExternalL2CL) Stop()  { n.stops++ }
func (n *fakeExternalL2CL) UserRPC() string {
	if n.rpc != "" {
		return n.rpc
	}
	return "http://127.0.0.1:9545"
}

func TestL2CLFactoryOwnsDeferredSharedProcessLifecycle(t *testing.T) {
	shared := &fakeExternalL2CL{}
	var seen []L2CLLaunchContext

	t.Run("factory lifetime", func(t *testing.T) {
		dt := devtest.SerialT(t)
		externalSlots := 0
		factory := L2CLFactoryFn(func(t devtest.T, ctx L2CLLaunchContext) (L2CLNode, bool) {
			seen = append(seen, ctx)
			if ctx.Target.Name == "stock" {
				return nil, false
			}
			externalSlots++
			if externalSlots == 2 {
				shared.Start()
				t.Cleanup(shared.Stop)
			}
			return shared, true
		})

		sequencer, handled := factory.CreateL2CL(dt, L2CLLaunchContext{
			Target: NewComponentTarget("sequencer", eth.ChainIDFromUInt64(901)),
			Role:   L2CLRoleSequencer,
		})
		require.True(t, handled)
		require.Zero(t, shared.starts, "the factory may defer a shared launch until all slots arrive")
		require.Equal(t, "http://127.0.0.1:9545", sequencer.UserRPC(), "the pre-launch RPC address is stable")

		verifier, handled := factory.CreateL2CL(dt, L2CLLaunchContext{
			Target: NewComponentTarget("late-verifier", eth.ChainIDFromUInt64(901)),
			Role:   L2CLRoleVerifier,
		})
		require.True(t, handled)
		require.Same(t, sequencer, verifier, "slot handles may share one backing process")

		_, handled = factory.CreateL2CL(dt, L2CLLaunchContext{
			Target: NewComponentTarget("stock", eth.ChainIDFromUInt64(901)),
			Role:   L2CLRoleVerifier,
		})
		require.False(t, handled)

		require.Equal(t, []string{"sequencer", "late-verifier", "stock"}, []string{
			seen[0].Target.Name,
			seen[1].Target.Name,
			seen[2].Target.Name,
		})
		require.Equal(t, 1, shared.starts, "the factory starts the shared process once after its final slot")
		require.Zero(t, shared.stops, "the factory defers cleanup until the test ends")
	})

	require.Equal(t, 1, shared.stops, "factory-owned cleanup stops the shared process once")
}

func TestResolveL2CLNodeConfigAppliesOptionsOnce(t *testing.T) {
	dt := devtest.SerialT(t)
	applications := 0
	target := NewComponentTarget("verifier", eth.ChainIDFromUInt64(901))
	cfg := resolveL2CLNodeConfig(dt, target, l2CLNodeStartConfig{
		Key:              target.Name,
		IsSequencer:      true,
		NoDiscovery:      true,
		EnableReqResp:    true,
		L2FollowSource:   "http://source.invalid",
		SyncMode:         nodeSync.ELSync,
		SequencerStopped: true,
		L2CLOptions: []L2CLOption{L2CLOptionFn(func(_ devtest.T, got ComponentTarget, cfg *L2CLConfig) {
			applications++
			require.Equal(t, target, got)
			cfg.FollowSource = "http://proxy.invalid"
			cfg.SequencerSyncMode = nodeSync.CLSync
		})},
	})

	require.Equal(t, 1, applications)
	require.True(t, cfg.IsSequencer)
	require.True(t, cfg.NoDiscovery)
	require.True(t, cfg.EnableReqRespSync)
	require.True(t, cfg.SequencerStopped)
	require.Equal(t, nodeSync.ELSync, cfg.SequencerSyncMode, "direct start state overrides an option")
	require.Equal(t, nodeSync.ELSync, cfg.VerifierSyncMode)
	require.Equal(t, "http://proxy.invalid", cfg.FollowSource)
}

func TestSingleChainFactoryHandlingControlsPeering(t *testing.T) {
	stock := &SingleChainNodeRuntime{}
	external := &SingleChainNodeRuntime{FactoryHandledCL: true}

	require.True(t, shouldConnectSingleChainCLPeers(stock, stock),
		"stock slots retain op-node admin P2P peering")
	require.False(t, shouldConnectSingleChainCLPeers(stock, external))
	require.False(t, shouldConnectSingleChainCLPeers(external, stock))
	require.False(t, shouldConnectSingleChainCLPeers(external, external))
}

func TestSingleChainNodeSourcesKeepFollowAndUnsafeSourceConsistent(t *testing.T) {
	source := &SingleChainNodeRuntime{
		EL: &fakeFactoryL2EL{rpc: "http://127.0.0.1:8545"},
		CL: &fakeExternalL2CL{rpc: "http://127.0.0.1:9545"},
	}

	follow, unsafeEL := singleChainNodeSources(source, "")
	require.Empty(t, follow, "stock-to-stock nodes retain P2P-only unsafe sync")
	require.Same(t, source.EL, unsafeEL)

	source.FactoryHandledCL = true
	follow, unsafeEL = singleChainNodeSources(source, "")
	require.Equal(t, source.CL.UserRPC(), follow,
		"a declined stock verifier follows an external source through its rollup RPC")
	require.Same(t, source.EL, unsafeEL)

	follow, unsafeEL = singleChainNodeSources(source, "http://explicit-stock-rollup.invalid")
	require.Equal(t, "http://explicit-stock-rollup.invalid", follow)
	require.Same(t, source.EL, unsafeEL,
		"an explicit source node supplies matching execution endpoints to the external factory")
}

func TestSingleChainInteropPrimaryRetainsOpNodeFallback(t *testing.T) {
	t.Setenv("DEVSTACK_L2CL_KIND", string(MixedL2CLKona))
	require.Equal(t, MixedL2CLKona, singleChainPrimaryFallbackKind(singleChainRuntimeWorld{}))
	require.Equal(t, MixedL2CLOpNode, singleChainPrimaryFallbackKind(singleChainRuntimeWorld{
		Interop: &SingleChainInteropSupport{},
	}))
}

func TestSingleChainAddedNodeDependencySetPreservesNilFactoryBehavior(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	depSet, err := depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
		chainID: {},
	})
	require.NoError(t, err)
	runtime := &SingleChainRuntime{Interop: &SingleChainInteropSupport{DependencySet: depSet}}

	require.Nil(t, singleChainAddedNodeDependencySet(runtime),
		"without a factory, added nodes retain their historical nil dependency set")
	runtime.L2CLFactory = L2CLFactoryFn(func(devtest.T, L2CLLaunchContext) (L2CLNode, bool) {
		return nil, false
	})
	require.Same(t, depSet, singleChainAddedNodeDependencySet(runtime),
		"a factory receives the interop dependency set for added-node slots")
}

func TestDeclinedKonaFallbackRejectsFactoryHandledSource(t *testing.T) {
	require.NoError(t, validateDeclinedL2CLFallback(MixedL2CLOpNode, true, "http://external-rollup.invalid"))
	require.NoError(t, validateDeclinedL2CLFallback(MixedL2CLKona, false, ""),
		"factory-declines-all keeps the ordinary Kona fallback")
	require.NoError(t,
		validateDeclinedL2CLFallback(MixedL2CLKona, false, "http://explicit-rollup.invalid"),
		"a pre-existing explicit follow source is not evidence of a factory-handled source",
	)
	require.EqualError(t,
		validateDeclinedL2CLFallback(MixedL2CLKona, true, "http://external-rollup.invalid"),
		`Kona fallback cannot follow external source "http://external-rollup.invalid"; handle the slot or select op-node`,
	)
}
