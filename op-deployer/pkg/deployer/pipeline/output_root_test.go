package pipeline

import (
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// setupChainWithGenesis builds a valid intent/state pair with an already generated L2
// genesis allocs and a pinned anchor/genesis time.
func setupChainWithGenesis(t *testing.T) (*Env, *state.Intent, *state.State, common.Hash) {
	t.Helper()

	lgr := testlog.Logger(t, slog.LevelWarn)
	_, pk, dk := shared.DefaultPrivkey(t)
	deployer := crypto.PubkeyToAddress(pk.PublicKey)

	l1ChainID := big.NewInt(900)
	l2ChainID := uint256.NewInt(1)
	loc, afacts := testutil.LocalArtifacts(t)

	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, standard.GasLimit)
	chain := intent.Chains[0]

	genesisTime := hexutil.Uint64(1_700_000_000)
	st.PinChainAnchor(chain.ID, &state.L1BlockRefJSON{
		Hash:   common.HexToHash("0xaaaa"),
		Number: 100,
		Time:   hexutil.Uint64(uint64(genesisTime) - 100),
	}, genesisTime)

	pEnv := &Env{Logger: lgr, Deployer: deployer}
	bundle := ArtifactsBundle{L1: afacts, L2: afacts}
	require.NoError(t, GenerateL2Genesis(pEnv, intent, bundle, st, chain.ID))

	return pEnv, intent, st, chain.ID
}

func TestComputeGenesisOutputRoot_RequiresAllocs(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelWarn)
	chainID := common.HexToHash("0x01")
	intent := &state.Intent{Chains: []*state.ChainIntent{{ID: chainID}}}
	st := &state.State{Version: 1}
	st.PinChainAnchor(chainID, &state.L1BlockRefJSON{Hash: common.HexToHash("0xbbbb"), Number: 100, Time: 100}, hexutil.Uint64(200))

	err := ComputeGenesisOutputRoot(&Env{Logger: lgr}, intent, st, chainID)
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

	err = ComputeGenesisOutputRoot(&Env{Logger: lgr}, intent, st, chainID)
	require.ErrorContains(t, err, "anchor block and genesis time not yet pinned")
}

func TestComputeGenesisOutputRoot_ComputesAndPersists(t *testing.T) {
	pEnv, intent, st, chainID := setupChainWithGenesis(t)

	require.NoError(t, ComputeGenesisOutputRoot(pEnv, intent, st, chainID))

	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	require.NotNil(t, chainState.GenesisBlockHash)
	require.NotNil(t, chainState.StartingAnchorRoot)
	require.Zero(t, uint64(chainState.StartingAnchorRoot.L2SequenceNumber),
		"the genesis anchor must always be sequence number 0")
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

	// Re-running must be a no-op.
	blockHashBefore := chainState.GenesisBlockHash
	anchorBefore := chainState.StartingAnchorRoot
	require.NoError(t, ComputeGenesisOutputRoot(pEnv, intent, st, chainID))
	chainStateAfter, err := st.Chain(chainID)
	require.NoError(t, err)
	require.Same(t, blockHashBefore, chainStateAfter.GenesisBlockHash)
	require.Same(t, anchorBefore, chainStateAfter.StartingAnchorRoot)
}
