package pipeline

import (
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// setupChainWithGenesis builds a valid single chain intent/state pair with an already generated L2
// genesis allocs and a pinned anchor/genesis time.
func setupChainWithGenesis(t *testing.T) (*Env, *state.Intent, *state.State, common.Hash) {
	t.Helper()
	pEnv, intent, st, ids := setupClusterWithGenesis(t, 1)
	return pEnv, intent, st, ids[0]
}

// setupClusterWithGenesis builds an intent of n chains that all share one anchor block and
// genesis time, with allocs generated and the interop dependency set recorded the way
// prepareChains does. Returns the chain IDs in intent order.
func setupClusterWithGenesis(t *testing.T, n int) (*Env, *state.Intent, *state.State, []common.Hash) {
	t.Helper()
	require.Greater(t, n, 0)

	lgr := testlog.Logger(t, slog.LevelWarn)
	_, pk, dk := shared.DefaultPrivkey(t)
	deployer := crypto.PubkeyToAddress(pk.PublicKey)

	l1ChainID := big.NewInt(900)
	loc, afacts := testutil.LocalArtifacts(t)

	intent, st := shared.NewIntent(t, l1ChainID, dk, uint256.NewInt(1), loc, loc, standard.GasLimit)
	for i := 2; i <= n; i++ {
		intent.Chains = append(intent.Chains, shared.NewChainIntent(t, dk, l1ChainID, uint256.NewInt(uint64(i)), standard.GasLimit))
	}

	genesisTime := hexutil.Uint64(1_700_000_000)
	anchor := &state.L1BlockRefJSON{
		Hash:   common.HexToHash("0xaaaa"),
		Number: 100,
		Time:   hexutil.Uint64(uint64(genesisTime) - 100),
	}

	pEnv := &Env{Logger: lgr, Deployer: deployer}
	bundle := artifacts.Bundle{L1: afacts, L2: afacts}
	ids := make([]common.Hash, 0, n)
	for _, chain := range intent.Chains {
		st.PinChainAnchor(chain.ID, anchor, genesisTime)
		require.NoError(t, GenerateL2Genesis(pEnv, intent, bundle, st, chain.ID))
		ids = append(ids, chain.ID)
	}

	depSet, err := BuildInteropDepSet(intent.Chains)
	require.NoError(t, err)
	st.InteropDepSet = depSet

	return pEnv, intent, st, ids
}

// useSuperGames points every chain in the intent at SUPER_CANNON_KONA.
func useSuperGames(t *testing.T, intent *state.Intent) {
	t.Helper()
	for _, chain := range intent.Chains {
		if chain.DeployOverrides == nil {
			chain.DeployOverrides = map[string]any{}
		}
		chain.DeployOverrides["respectedGameType"] = embedded.GameTypeSuperCannonKona
	}
}

// v0RootOf independently recomputes a chain's plain V0 genesis output root and genesis timestamp.
func v0RootOf(t *testing.T, intent *state.Intent, st *state.State, chainID common.Hash) (common.Hash, uint64) {
	t.Helper()
	chainIntent, err := intent.Chain(chainID)
	require.NoError(t, err)
	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	config, err := state.CombineDeployConfig(intent, chainIntent, st, chainState)
	require.NoError(t, err)
	l2Genesis, err := genesis.BuildL2Genesis(&config, chainState.Allocs.Data, chainState.StartBlock.ToBlockRef())
	require.NoError(t, err)
	header := l2Genesis.ToBlock().Header()
	require.NotNil(t, header.WithdrawalsHash)
	root, err := rollup.ComputeL2OutputRootV0(eth.HeaderBlockInfo(header), *header.WithdrawalsHash)
	require.NoError(t, err)
	return common.Hash(root), header.Time
}

func TestComputeGenesisOutputRoot_RequiresAllocs(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelWarn)
	chainID := common.HexToHash("0x01")
	intent := &state.Intent{Chains: []*state.ChainIntent{{ID: chainID}}}
	st := &state.State{Version: 1}
	st.PinChainAnchor(chainID, &state.L1BlockRefJSON{Hash: common.HexToHash("0xbbbb"), Number: 100, Time: 100}, hexutil.Uint64(200))

	err := ComputeGenesisOutputRoots(&Env{Logger: lgr}, intent, st)
	require.ErrorContains(t, err, "allocs not yet generated")
}

func TestComputeGenesisOutputRoot_RequiresAnchor(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelWarn)
	chainID := common.HexToHash("0x01")
	intent := &state.Intent{Chains: []*state.ChainIntent{{ID: chainID}}}
	st := &state.State{Version: 1}
	st.SetChainContracts(chainID, addresses.OpChainContracts{}, false)
	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	chainState.Allocs = &state.GzipData[foundry.ForgeAllocs]{Data: &foundry.ForgeAllocs{}}

	err = ComputeGenesisOutputRoots(&Env{Logger: lgr}, intent, st)
	require.ErrorContains(t, err, "anchor block and genesis time not yet pinned")
}

func TestComputeGenesisOutputRoot_ComputesAndPersists(t *testing.T) {
	pEnv, intent, st, chainID := setupChainWithGenesis(t)

	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))

	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	require.NotNil(t, chainState.GenesisBlockHash)
	require.NotNil(t, chainState.StartingAnchorRoot)
	require.Zero(t, uint64(chainState.StartingAnchorRoot.L2SequenceNumber),
		"the genesis anchor must always be sequence number 0 for a non super root game")
	require.NotEqual(t, common.Hash{}, chainState.StartingAnchorRoot.Root)
	require.NotEqual(t, opcm.DefaultStartingAnchorRoot.Root, chainState.StartingAnchorRoot.Root)

	// Cross-check: independently rebuild the same L2 genesis block and recompute the output root.
	chainIntent, err := intent.Chain(chainID)
	require.NoError(t, err)
	config, err := state.CombineDeployConfig(intent, chainIntent, st, chainState)
	require.NoError(t, err)
	l2Genesis, err := genesis.BuildL2Genesis(&config, chainState.Allocs.Data, chainState.StartBlock.ToBlockRef())
	require.NoError(t, err)
	block := l2Genesis.ToBlock()
	require.Equal(t, block.Hash(), *chainState.GenesisBlockHash)

	require.NotNil(t, block.Header().WithdrawalsHash, "genesis block must be Isthmus-active to carry a withdrawals root")
	wantOutputRoot, err := rollup.ComputeL2OutputRootV0(eth.HeaderBlockInfo(block.Header()), *block.Header().WithdrawalsHash)
	require.NoError(t, err)
	require.Equal(t, common.Hash(wantOutputRoot), chainState.StartingAnchorRoot.Root)

	// Re-running must be deterministic.
	blockHashBefore := *chainState.GenesisBlockHash
	anchorBefore := *chainState.StartingAnchorRoot
	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))
	chainStateAfter, err := st.Chain(chainID)
	require.NoError(t, err)
	require.Equal(t, blockHashBefore, *chainStateAfter.GenesisBlockHash)
	require.Equal(t, anchorBefore, *chainStateAfter.StartingAnchorRoot)
}

// TestComputeGenesisOutputRoot_SkipsAlreadyDeployedChain verifies that a chain deployed via the
// plain apply pipeline, which populates Allocs/StartBlock/GenesisTime but never calls
// ComputeGenesisOutputRoot, since it uses the placeholder anchor.
func TestComputeGenesisOutputRoot_SkipsAlreadyDeployedChain(t *testing.T) {
	pEnv, intent, st, chainID := setupChainWithGenesis(t)

	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	require.Nil(t, chainState.StartingAnchorRoot)
	require.Nil(t, chainState.GenesisBlockHash)

	st.SetChainContracts(chainID, chainState.OpChainContracts, true)
	require.True(t, st.IsChainDeployed(chainID))

	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))

	chainStateAfter, err := st.Chain(chainID)
	require.NoError(t, err)
	require.Nil(t, chainStateAfter.StartingAnchorRoot, "must not fabricate an anchor for an already-deployed chain")
	require.Nil(t, chainStateAfter.GenesisBlockHash, "must not fabricate a genesis block hash for an already-deployed chain")
}

// TestComputeGenesisOutputRoot_SuperGameTypeWrapsSuperV1Root verifies that a chain configured
// for SUPER_CANNON_KONA gets a genesis anchor encoded as a SuperV1 root over just its own
// output, not the bare V0 root.
func TestComputeGenesisOutputRoot_SuperGameTypeWrapsSuperV1Root(t *testing.T) {
	pEnv, intent, st, chainID := setupChainWithGenesis(t)

	chainIntent, err := intent.Chain(chainID)
	require.NoError(t, err)
	if chainIntent.DeployOverrides == nil {
		chainIntent.DeployOverrides = map[string]any{}
	}
	chainIntent.DeployOverrides["respectedGameType"] = embedded.GameTypeSuperCannonKona

	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))

	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	require.NotNil(t, chainState.StartingAnchorRoot)

	config, err := state.CombineDeployConfig(intent, chainIntent, st, chainState)
	require.NoError(t, err)
	l2Genesis, err := genesis.BuildL2Genesis(&config, chainState.Allocs.Data, chainState.StartBlock.ToBlockRef())
	require.NoError(t, err)
	block := l2Genesis.ToBlock()
	header := block.Header()
	plainV0Root, err := rollup.ComputeL2OutputRootV0(eth.HeaderBlockInfo(header), *header.WithdrawalsHash)
	require.NoError(t, err)

	superRoot := eth.NewSuperV1(header.Time, eth.ChainIDAndOutput{
		ChainID: eth.ChainIDFromBytes32(chainID),
		Output:  eth.Bytes32(plainV0Root),
	})
	wantAnchor := common.Hash(eth.SuperRoot(superRoot))

	require.Equal(t, wantAnchor, chainState.StartingAnchorRoot.Root)
	require.NotEqual(t, common.Hash(plainV0Root), chainState.StartingAnchorRoot.Root,
		"the persisted anchor must be the SuperV1 wrap, not the bare V0 root")

	// A super-root game measures l2SequenceNumber in timestamps, not blocks, so the genesis
	// anchor carries the L2 genesis time.
	require.Equal(t, header.Time, uint64(chainState.StartingAnchorRoot.L2SequenceNumber),
		"the super genesis anchor must be sequenced by the L2 genesis timestamp")
	require.NotNil(t, chainState.GenesisTime)
	require.Equal(t, uint64(*chainState.GenesisTime), uint64(chainState.StartingAnchorRoot.L2SequenceNumber),
		"the anchor sequence number must be the genesis time pinned by prepare")
}

// TestComputeGenesisOutputRoot_SuperAnchorRoundTrips verifies that the persistedpair is self-consistent:
//
//	rebuilding the SuperV1 encoding from the recorded sequence number alone must reproduce the recorded anchor root.
func TestComputeGenesisOutputRoot_SuperAnchorRoundTrips(t *testing.T) {
	pEnv, intent, st, chainID := setupChainWithGenesis(t)

	chainIntent, err := intent.Chain(chainID)
	require.NoError(t, err)
	if chainIntent.DeployOverrides == nil {
		chainIntent.DeployOverrides = map[string]any{}
	}
	chainIntent.DeployOverrides["respectedGameType"] = embedded.GameTypeSuperCannonKona

	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))

	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	require.NotNil(t, chainState.StartingAnchorRoot)

	anchor := chainState.StartingAnchorRoot
	timestamp := uint64(anchor.L2SequenceNumber)
	require.NotZero(t, timestamp, "a super anchor sequenced at 0 is unreachable by any trace")

	// Recover the chain's own output root from the anchor by rebuilding the genesis block, then
	// re-wrap it using only the recorded sequence number.
	config, err := state.CombineDeployConfig(intent, chainIntent, st, chainState)
	require.NoError(t, err)
	l2Genesis, err := genesis.BuildL2Genesis(&config, chainState.Allocs.Data, chainState.StartBlock.ToBlockRef())
	require.NoError(t, err)
	header := l2Genesis.ToBlock().Header()
	v0Root, err := rollup.ComputeL2OutputRootV0(eth.HeaderBlockInfo(header), *header.WithdrawalsHash)
	require.NoError(t, err)

	rebuilt := eth.NewSuperV1(timestamp, eth.ChainIDAndOutput{
		ChainID: eth.ChainIDFromBytes32(chainID),
		Output:  eth.Bytes32(v0Root),
	})
	require.Equal(t, anchor.Root, common.Hash(eth.SuperRoot(rebuilt)),
		"the anchor root must be the SuperV1 encoding at the recorded sequence number")

	// The same root at a nearby timestamp is a different hash, so a
	// sequence number that drifts from the encoded timestamp must be caught.
	offBySecond := eth.NewSuperV1(timestamp+1, eth.ChainIDAndOutput{
		ChainID: eth.ChainIDFromBytes32(chainID),
		Output:  eth.Bytes32(v0Root),
	})
	require.NotEqual(t, anchor.Root, common.Hash(eth.SuperRoot(offBySecond)),
		"the SuperV1 encoding must commit to the timestamp")
}

// TestComputeGenesisOutputRoots_SuperAnchorSpansTheDependencySet verifies that every chain
// in a multi-chain super-root deployment is seeded with the *same* anchor, the SuperV1 root over
// every dependency set member's genesis output root at the shared genesis timestamp. Each chain
// still gets its own AnchorStateRegistry, what they share is the root they are anchored to.
func TestComputeGenesisOutputRoots_SuperAnchorSpansTheDependencySet(t *testing.T) {
	pEnv, intent, st, ids := setupClusterWithGenesis(t, 3)
	useSuperGames(t, intent)

	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))

	// Independently assemble the expected cluster-wide super root.
	chainOutputs := make([]eth.ChainIDAndOutput, 0, len(ids))
	var sharedTimestamp uint64
	for _, id := range ids {
		v0Root, timestamp := v0RootOf(t, intent, st, id)
		if sharedTimestamp == 0 {
			sharedTimestamp = timestamp
		}
		require.Equal(t, sharedTimestamp, timestamp, "the fixture must give every chain one genesis time")
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromBytes32(id),
			Output:  eth.Bytes32(v0Root),
		})
	}
	wantAnchor := common.Hash(eth.SuperRoot(eth.NewSuperV1(sharedTimestamp, chainOutputs...)))

	for _, id := range ids {
		chainState, err := st.Chain(id)
		require.NoError(t, err)
		require.NotNil(t, chainState.StartingAnchorRoot, "chain %s must have an anchor", id.Hex())
		require.Equal(t, wantAnchor, chainState.StartingAnchorRoot.Root,
			"chain %s must be anchored to the cluster-wide super root", id.Hex())
		require.Equal(t, sharedTimestamp, uint64(chainState.StartingAnchorRoot.L2SequenceNumber),
			"chain %s must be sequenced by the shared genesis timestamp", id.Hex())

		// The anchor must not be the chain's own solo super root - that is the defect this fixes.
		v0Root, _ := v0RootOf(t, intent, st, id)
		solo := common.Hash(eth.SuperRoot(eth.NewSuperV1(sharedTimestamp, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromBytes32(id),
			Output:  eth.Bytes32(v0Root),
		})))
		require.NotEqual(t, solo, chainState.StartingAnchorRoot.Root,
			"chain %s must not be anchored to a solo N=1 super root", id.Hex())
	}
}

// TestComputeGenesisOutputRoots_OutputRootAnchorsStayPerChain is the regression guard for the
// non super path: a multi-chain output root deployment must still give every chain its own plain
// V0 root at sequence number 0.
func TestComputeGenesisOutputRoots_OutputRootAnchorsStayPerChain(t *testing.T) {
	pEnv, intent, st, ids := setupClusterWithGenesis(t, 3)

	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))

	seen := make(map[common.Hash]struct{}, len(ids))
	for _, id := range ids {
		wantRoot, _ := v0RootOf(t, intent, st, id)
		chainState, err := st.Chain(id)
		require.NoError(t, err)
		require.NotNil(t, chainState.StartingAnchorRoot)
		require.Equal(t, wantRoot, chainState.StartingAnchorRoot.Root,
			"chain %s must be anchored to its own V0 root", id.Hex())
		require.Zero(t, uint64(chainState.StartingAnchorRoot.L2SequenceNumber),
			"an output-root anchor is sequenced by block number, and genesis is block 0")
		seen[chainState.StartingAnchorRoot.Root] = struct{}{}
	}
	require.Len(t, seen, len(ids), "each chain must get a distinct anchor, not a shared one")
}

// TestComputeGenesisOutputRoots_SuperAnchorFollowsTheDependencySet verifies the chain list comes
// from the recorded dependency set rather than from the intent.
func TestComputeGenesisOutputRoots_SuperAnchorFollowsTheDependencySet(t *testing.T) {
	pEnv, intent, st, ids := setupClusterWithGenesis(t, 2)
	useSuperGames(t, intent)

	// Record a dependency set naming a chain the intent does not describe.
	strayDepSet, err := BuildInteropDepSet([]*state.ChainIntent{
		{ID: ids[0]},
		{ID: common.HexToHash("0xbeef")},
	})
	require.NoError(t, err)
	st.InteropDepSet = strayDepSet

	err = ComputeGenesisOutputRoots(pEnv, intent, st)
	require.ErrorContains(t, err, "has no genesis output root in this run")
}

// TestComputeGenesisOutputRoots_SuperAnchorRejectsDivergentGenesisTimes verifies that
// every set member share the same genesis time.
func TestComputeGenesisOutputRoots_SuperAnchorRejectsDivergentGenesisTimes(t *testing.T) {
	pEnv, intent, st, ids := setupClusterWithGenesis(t, 2)
	useSuperGames(t, intent)

	// Re-pin the second chain 60s later, as a per-chain l1StartBlockHash override would.
	second, err := st.Chain(ids[1])
	require.NoError(t, err)
	later := hexutil.Uint64(uint64(*second.GenesisTime) + 60)
	second.GenesisTime = &later

	err = ComputeGenesisOutputRoots(pEnv, intent, st)
	require.ErrorContains(t, err, "disagree on the L2 genesis timestamp")
	require.ErrorContains(t, err, "must share a genesis time")
}

// TestComputeGenesisOutputRoots_SuperAnchorRejectsDeployedMember checks that an already deployed chain
// in the dependency set causes ComputeGenesisOutputRoots to fail, because it cannot be re-anchored to
// a new super root.
func TestComputeGenesisOutputRoots_SuperAnchorRejectsDeployedMember(t *testing.T) {
	pEnv, intent, st, ids := setupClusterWithGenesis(t, 2)
	useSuperGames(t, intent)

	first, err := st.Chain(ids[0])
	require.NoError(t, err)
	st.SetChainContracts(ids[0], first.OpChainContracts, true)
	require.True(t, st.IsChainDeployed(ids[0]))

	err = ComputeGenesisOutputRoots(pEnv, intent, st)
	require.ErrorContains(t, err, "has no genesis output root in this run")
	require.ErrorContains(t, err, "already deployed")
}

// TestComputeGenesisOutputRoots_PermissionedMemberJoinsSuperCluster covers a permissioned chain
// alongside a SUPER_CANNON_KONA one.
func TestComputeGenesisOutputRoots_PermissionedMemberJoinsSuperCluster(t *testing.T) {
	pEnv, intent, st, ids := setupClusterWithGenesis(t, 2)

	// Chain 0 keeps the default PERMISSIONED_CANNON; only chain 1 asks for a super game.
	if intent.Chains[1].DeployOverrides == nil {
		intent.Chains[1].DeployOverrides = map[string]any{}
	}
	intent.Chains[1].DeployOverrides["respectedGameType"] = embedded.GameTypeSuperCannonKona

	require.NoError(t, ComputeGenesisOutputRoots(pEnv, intent, st))

	chainOutputs := make([]eth.ChainIDAndOutput, 0, len(ids))
	var sharedTimestamp uint64
	for _, id := range ids {
		v0Root, timestamp := v0RootOf(t, intent, st, id)
		sharedTimestamp = timestamp
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromBytes32(id),
			Output:  eth.Bytes32(v0Root),
		})
	}
	wantAnchor := common.Hash(eth.SuperRoot(eth.NewSuperV1(sharedTimestamp, chainOutputs...)))

	for _, id := range ids {
		chainState, err := st.Chain(id)
		require.NoError(t, err)
		require.NotNil(t, chainState.StartingAnchorRoot)
		require.Equal(t, wantAnchor, chainState.StartingAnchorRoot.Root,
			"chain %s must be anchored to the cluster-wide super root", id.Hex())
		require.Equal(t, sharedTimestamp, uint64(chainState.StartingAnchorRoot.L2SequenceNumber))
	}
}

// TestComputeGenesisOutputRoots_SuperAnchorRequiresDependencySet guards the case where a prepared
// state predates the dependency set being recorded.
func TestComputeGenesisOutputRoots_SuperAnchorRequiresDependencySet(t *testing.T) {
	pEnv, intent, st, _ := setupClusterWithGenesis(t, 2)
	useSuperGames(t, intent)
	st.InteropDepSet = nil

	err := ComputeGenesisOutputRoots(pEnv, intent, st)
	require.ErrorContains(t, err, "no interop dependency set recorded")
}
