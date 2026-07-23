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

// Compile-time: the two-L2 external-CL interop runtime constructors — including
// the nil-factory fallback path that starts stock op-node verifiers — are part
// of the sysgo API surface consumed by presets.
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

// TestTwoL2ExternalCLInteropVerifierSlotsUseFactory drives the exact per-chain
// verifier-slot call the two-L2 external-CL interop runtime makes
// (startInteropVerifierCL) with fakes at the L2CLFactory seam, mirroring
// TestL2CLFactorySupportsPerSlotSelectionAndRestart. Running the full world in
// a unit test is too heavy (contract deployment + real EL/CL processes); the
// factory contract is validated here and the world wiring by devstack runs.
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
	followSourceA := "http://127.0.0.1:9545"
	followSourceB := "http://127.0.0.1:9645"
	sequencerAEL := fakeInteropL2EL{user: followSourceA, engine: "http://127.0.0.1:9551", jwt: "/jwt/seq-a"}
	sequencerBEL := fakeInteropL2EL{user: followSourceB, engine: "http://127.0.0.1:9651", jwt: "/jwt/seq-b"}

	var seen []L2CLLaunchContext
	nodes := make(map[eth.ChainID]*fakeExternalL2CL)
	factory := L2CLFactoryFn(func(_ devtest.T, ctx L2CLLaunchContext) (L2CLNode, bool) {
		seen = append(seen, ctx)
		node := &fakeExternalL2CL{}
		nodes[ctx.Target.ChainID] = node
		return node, true
	})

	verifierACL := startInteropVerifierCL(dt, nil, l1Net, l2ANet, l1EL, l1CL, verifierAEL, [32]byte{}, sequencerAEL, depSet, nil, factory)
	verifierBCL := startInteropVerifierCL(dt, nil, l1Net, l2BNet, l1EL, l1CL, verifierBEL, [32]byte{}, sequencerBEL, depSet, nil, factory)

	require.Len(t, seen, 2, "factory must be consulted exactly once per chain")
	require.Same(t, nodes[chainAID], verifierACL, "chain A slot uses the factory-provided node")
	require.Same(t, nodes[chainBID], verifierBCL, "chain B slot uses the factory-provided node")

	require.Equal(t, chainAID, seen[0].Target.ChainID)
	require.Equal(t, chainBID, seen[1].Target.ChainID)
	require.NotEqual(t, seen[0].Target.ChainID, seen[1].Target.ChainID, "per-chain launch contexts must carry distinct chain IDs")

	for _, ctx := range seen {
		require.Equal(t, L2CLRoleVerifier, ctx.Role, "external CL slots in this runtime are verifiers")
		require.Equal(t, twoL2VerifierNodeKey, ctx.Target.Name)
		require.False(t, ctx.Config.IsSequencer)

		require.NotNil(t, ctx.DependencySet, "interop world must share its dependency set with the factory")
		require.True(t, ctx.DependencySet.HasChain(chainAID), "dependency set must contain chain A")
		require.True(t, ctx.DependencySet.HasChain(chainBID), "dependency set must contain chain B")

		require.NotEmpty(t, ctx.L1UserRPC)
		require.NotEmpty(t, ctx.L1BeaconRPC)
		require.NotNil(t, ctx.L1Genesis)
		require.NotNil(t, ctx.L2Genesis)
		require.NotNil(t, ctx.RollupConfig)
	}

	require.Equal(t, verifierAEL.user, seen[0].L2UserRPC)
	require.Equal(t, verifierAEL.engine, seen[0].L2EngineRPC)
	require.Equal(t, verifierAEL.jwt, seen[0].L2JWTPath)
	require.Equal(t, followSourceA, seen[0].FollowSource)
	require.Equal(t, sequencerAEL.engine, seen[0].UnsafeSourceEngineRPC)
	require.Equal(t, sequencerAEL.jwt, seen[0].UnsafeSourceJWTPath)
	require.Equal(t, chainAID.ToBig(), seen[0].RollupConfig.L2ChainID)

	require.Equal(t, verifierBEL.user, seen[1].L2UserRPC)
	require.Equal(t, verifierBEL.engine, seen[1].L2EngineRPC)
	require.Equal(t, verifierBEL.jwt, seen[1].L2JWTPath)
	require.Equal(t, followSourceB, seen[1].FollowSource)
	require.Equal(t, sequencerBEL.engine, seen[1].UnsafeSourceEngineRPC)
	require.Equal(t, sequencerBEL.jwt, seen[1].UnsafeSourceJWTPath)
	require.Equal(t, chainBID.ToBig(), seen[1].RollupConfig.L2ChainID)
}
