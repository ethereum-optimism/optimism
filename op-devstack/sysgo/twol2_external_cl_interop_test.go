package sysgo

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"
)

var (
	_ func(devtest.T, uint64) *MultiChainRuntime               = NewTwoL2ExternalCLInteropRuntime
	_ func(devtest.T, uint64, PresetConfig) *MultiChainRuntime = NewTwoL2ExternalCLInteropRuntimeWithConfig
)

type fakeInteropL1EL struct{}

func (fakeInteropL1EL) l1ELNode()       {}
func (fakeInteropL1EL) UserRPC() string { return "http://127.0.0.1:8545" }
func (fakeInteropL1EL) AuthRPC() string { return "http://127.0.0.1:8551" }

type fakeInteropL2EL struct {
	user   string
	engine string
	jwt    string
}

func (fakeInteropL2EL) Start()              {}
func (fakeInteropL2EL) Stop()               {}
func (n fakeInteropL2EL) UserRPC() string   { return n.user }
func (n fakeInteropL2EL) EngineRPC() string { return n.engine }
func (n fakeInteropL2EL) JWTPath() string   { return n.jwt }

func TestTwoL2ExternalCLInteropVerifierSlotsUseFactory(t *testing.T) {
	dt := devtest.SerialT(t)
	chainAID := eth.ChainIDFromUInt64(901)
	chainBID := eth.ChainIDFromUInt64(902)
	depSet, err := depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
		chainAID: {},
		chainBID: {},
	})
	require.NoError(t, err)

	l1Net := &L1Network{name: "l1", chainID: eth.ChainIDFromUInt64(900), genesis: &core.Genesis{}}
	l1CL := &L1CLNode{name: "l1", beaconHTTPAddr: "http://127.0.0.1:4000"}
	l1EL := fakeInteropL1EL{}
	newL2Net := func(name string, id eth.ChainID) *L2Network {
		return &L2Network{
			name:      name,
			chainID:   id,
			l1ChainID: l1Net.ChainID(),
			genesis:   &core.Genesis{},
			rollupCfg: &rollup.Config{L2ChainID: id.ToBig()},
		}
	}
	l2ANet := newL2Net("l2a", chainAID)
	l2BNet := newL2Net("l2b", chainBID)
	verifierAEL := fakeInteropL2EL{user: "http://127.0.0.1:9001", engine: "http://127.0.0.1:9002", jwt: "/jwt/a"}
	verifierBEL := fakeInteropL2EL{user: "http://127.0.0.1:9101", engine: "http://127.0.0.1:9102", jwt: "/jwt/b"}
	sequencerAEL := fakeInteropL2EL{user: "http://127.0.0.1:9545", engine: "http://127.0.0.1:9551", jwt: "/jwt/seq-a"}
	sequencerBEL := fakeInteropL2EL{user: "http://127.0.0.1:9645", engine: "http://127.0.0.1:9651", jwt: "/jwt/seq-b"}
	sequencerACL := &fakeExternalL2CL{rpc: "http://127.0.0.1:7545"}
	sequencerBCL := &fakeExternalL2CL{rpc: "http://127.0.0.1:7645"}

	var seen []L2CLLaunchContext
	factory := L2CLFactoryFn(func(_ devtest.T, ctx L2CLLaunchContext) (L2CLNode, bool) {
		seen = append(seen, ctx)
		return &fakeExternalL2CL{}, true
	})
	optionApplications := 0
	opt := L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		optionApplications++
		cfg.SequencerMaxSafeLag = 7
	})

	verifierACL := startInteropVerifierCL(
		dt, nil, l1Net, l2ANet, l1EL, l1CL, verifierAEL, [32]byte{}, sequencerAEL, sequencerACL,
		depSet, []L2CLOption{opt}, factory,
	)
	verifierBCL := startInteropVerifierCL(
		dt, nil, l1Net, l2BNet, l1EL, l1CL, verifierBEL, [32]byte{}, sequencerBEL, sequencerBCL,
		depSet, []L2CLOption{opt}, factory,
	)

	require.NotNil(t, verifierACL)
	require.NotNil(t, verifierBCL)
	require.Len(t, seen, 2)
	require.Equal(t, 2, optionApplications, "each slot resolves options exactly once")
	for _, ctx := range seen {
		require.Equal(t, L2CLRoleVerifier, ctx.Role)
		require.Equal(t, twoL2VerifierNodeKey, ctx.Target.Name)
		require.False(t, ctx.Config.IsSequencer)
		require.Equal(t, uint64(7), ctx.Config.SequencerMaxSafeLag)
		require.Same(t, depSet, ctx.DependencySet)
		require.True(t, ctx.DependencySet.HasChain(chainAID))
		require.True(t, ctx.DependencySet.HasChain(chainBID))
		require.NotNil(t, ctx.L1Genesis)
		require.NotNil(t, ctx.L2Genesis)
		require.NotNil(t, ctx.RollupConfig)
		require.Equal(t, ctx.Config.FollowSource, ctx.FollowSource,
			"external and fallback selection share one rollup-RPC follow configuration")
		require.Nil(t, ctx.RegisterMetrics, "classic runtime has no metrics registrar")
	}

	require.Equal(t, chainAID, seen[0].Target.ChainID)
	require.Equal(t, verifierAEL.user, seen[0].L2UserRPC)
	require.Equal(t, verifierAEL.engine, seen[0].L2EngineRPC)
	require.Equal(t, verifierAEL.jwt, seen[0].L2JWTPath)
	require.Equal(t, sequencerACL.UserRPC(), seen[0].FollowSource)
	require.Equal(t, sequencerAEL.user, seen[0].UnsafeSourceUserRPC)
	require.Equal(t, sequencerAEL.engine, seen[0].UnsafeSourceEngineRPC)
	require.Equal(t, sequencerAEL.jwt, seen[0].UnsafeSourceJWTPath)

	require.Equal(t, chainBID, seen[1].Target.ChainID)
	require.Equal(t, verifierBEL.user, seen[1].L2UserRPC)
	require.Equal(t, verifierBEL.engine, seen[1].L2EngineRPC)
	require.Equal(t, verifierBEL.jwt, seen[1].L2JWTPath)
	require.Equal(t, sequencerBCL.UserRPC(), seen[1].FollowSource)
	require.Equal(t, sequencerBEL.user, seen[1].UnsafeSourceUserRPC)
	require.Equal(t, sequencerBEL.engine, seen[1].UnsafeSourceEngineRPC)
	require.Equal(t, sequencerBEL.jwt, seen[1].UnsafeSourceJWTPath)
}

func TestTwoL2ExternalCLInteropVerifierFollowSources(t *testing.T) {
	sequencerCL := &fakeExternalL2CL{rpc: "http://127.0.0.1:7545"}

	stock := interopVerifierFollowSource(sequencerCL)

	require.Equal(t, sequencerCL.UserRPC(), stock,
		"a nil or declining factory falls back to op-node's rollup-RPC follow source")
}
