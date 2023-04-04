package l2

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-program/l2/engineapi"
	"github.com/ethereum-optimism/optimism/op-program/l2/engineapi/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestInitialState(t *testing.T) {
	blocks, chain := setupOracleBackedChain(t, 5)
	head := blocks[5]
	require.Equal(t, head.Header(), chain.CurrentBlock())
	require.Equal(t, head.Header(), chain.CurrentSafeBlock())
	require.Equal(t, head.Header(), chain.CurrentFinalBlock())
}

func TestGetBlocks(t *testing.T) {
	blocks, chain := setupOracleBackedChain(t, 5)

	for i, block := range blocks {
		blockNumber := uint64(i)
		assertBlockDataAvailable(t, chain, block, blockNumber)
		require.Equal(t, block.Hash(), chain.GetCanonicalHash(blockNumber), "get canonical hash for block %v", blockNumber)
	}
}

func TestUnknownBlock(t *testing.T) {
	_, chain := setupOracleBackedChain(t, 1)
	hash := common.HexToHash("0x556677881122")
	blockNumber := uint64(1)
	require.Nil(t, chain.GetBlockByHash(hash))
	require.Nil(t, chain.GetHeaderByHash(hash))
	require.Nil(t, chain.GetBlock(hash, blockNumber))
	require.Nil(t, chain.GetHeader(hash, blockNumber))
	require.False(t, chain.HasBlockAndState(hash, blockNumber))
}

func TestCanonicalHashNotFoundPastChainHead(t *testing.T) {
	blocks, chain := setupOracleBackedChainWithLowerHead(t, 5, 3)

	require.Equal(t, blocks[0].Hash(), chain.GetCanonicalHash(0))
	require.Equal(t, blocks[1].Hash(), chain.GetCanonicalHash(1))
	require.Equal(t, blocks[2].Hash(), chain.GetCanonicalHash(2))
	require.Equal(t, blocks[3].Hash(), chain.GetCanonicalHash(3))
	require.Equal(t, common.Hash{}, chain.GetCanonicalHash(4))
	require.Equal(t, common.Hash{}, chain.GetCanonicalHash(5))
}

func TestAppendToChain(t *testing.T) {
	blocks, chain := setupOracleBackedChainWithLowerHead(t, 4, 3)
	newBlock := blocks[4]
	require.Nil(t, chain.GetBlockByHash(newBlock.Hash()), "block unknown before being added")

	require.NoError(t, chain.InsertBlockWithoutSetHead(newBlock))
	require.Equal(t, blocks[3].Header(), chain.CurrentBlock(), "should not update chain head yet")
	require.Equal(t, common.Hash{}, chain.GetCanonicalHash(uint64(4)), "not yet a canonical hash")
	assertBlockDataAvailable(t, chain, newBlock, 4)

	canonical, err := chain.SetCanonical(newBlock)
	require.NoError(t, err)
	require.Equal(t, newBlock.Hash(), canonical)
	require.Equal(t, newBlock.Hash(), chain.GetCanonicalHash(uint64(4)), "get canonical hash for new head")
}

func TestSetFinalized(t *testing.T) {
	blocks, chain := setupOracleBackedChainWithLowerHead(t, 5, 0)
	for _, block := range blocks[1:] {
		require.NoError(t, chain.InsertBlockWithoutSetHead(block))
	}
	chain.SetFinalized(blocks[2].Header())
	require.Equal(t, blocks[2].Header(), chain.CurrentFinalBlock())
}

func TestSetSafe(t *testing.T) {
	blocks, chain := setupOracleBackedChainWithLowerHead(t, 5, 0)
	for _, block := range blocks[1:] {
		require.NoError(t, chain.InsertBlockWithoutSetHead(block))
	}
	chain.SetSafe(blocks[2].Header())
	require.Equal(t, blocks[2].Header(), chain.CurrentSafeBlock())
}

func TestUpdateStateDatabaseWhenImportingBlock(t *testing.T) {
	blocks, chain := setupOracleBackedChainWithLowerHead(t, 4, 3)
	newBlock := blocks[4]

	state, err := chain.StateAt(blocks[0].Root())
	require.NoError(t, err)
	balance := state.GetBalance(genesis.DevAccounts[0])
	require.NotEqual(t, balance, big.NewInt(0), "should have balance at imported block")

	state, err = chain.StateAt(newBlock.Root())
	require.NoError(t, err)
	balance = state.GetBalance(genesis.DevAccounts[0])
	require.Equal(t, balance, big.NewInt(0), "should not have balance from not-yet-imported block")
}

func assertBlockDataAvailable(t *testing.T, chain *OracleBackedL2Chain, block *types.Block, blockNumber uint64) {
	require.Equal(t, block, chain.GetBlockByHash(block.Hash()), "get block %v by hash", blockNumber)
	require.Equal(t, block.Header(), chain.GetHeaderByHash(block.Hash()), "get header %v by hash", blockNumber)
	require.Equal(t, block, chain.GetBlock(block.Hash(), blockNumber), "get block %v by hash and number", blockNumber)
	require.Equal(t, block.Header(), chain.GetHeader(block.Hash(), blockNumber), "get header %v by hash and number", blockNumber)
	require.True(t, chain.HasBlockAndState(block.Hash(), blockNumber), "has block and state for block %v", blockNumber)
}

func setupOracleBackedChain(t *testing.T, blockCount int) ([]*types.Block, *OracleBackedL2Chain) {
	return setupOracleBackedChainWithLowerHead(t, blockCount, blockCount)
}

func setupOracleBackedChainWithLowerHead(t *testing.T, blockCount int, headBlockNumber int) ([]*types.Block, *OracleBackedL2Chain) {
	logger := testlog.Logger(t, log.LvlDebug)
	chainCfg, blocks, oracle := setupOracle(t, blockCount, headBlockNumber)
	head := blocks[headBlockNumber].Hash()
	chain, err := NewOracleBackedL2Chain(context.Background(), logger, oracle, chainCfg, head)
	require.NoError(t, err)
	return blocks, chain
}

func setupOracle(t *testing.T, blockCount int, headBlockNumber int) (*params.ChainConfig, []*types.Block, *stubOracle) {
	deployConfig := &genesis.DeployConfig{
		L1ChainID:       900,
		L2ChainID:       901,
		L2BlockTime:     2,
		FundDevAccounts: true,
	}
	l1Genesis, err := genesis.NewL1Genesis(deployConfig)
	require.NoError(t, err)
	l2Genesis, err := genesis.NewL2Genesis(deployConfig, l1Genesis.ToBlock())
	chainCfg := l2Genesis.Config
	consensus := beacon.New(nil)
	db := rawdb.NewMemoryDatabase()

	// Set minimal amount of stuff to avoid nil references later
	genesisBlock := l2Genesis.MustCommit(db)
	blocks, _ := core.GenerateChain(chainCfg, genesisBlock, consensus, db, blockCount, func(i int, gen *core.BlockGen) {})
	blocks = append([]*types.Block{genesisBlock}, blocks...)
	oracle := newStubOracle(blocks[:headBlockNumber+1], db)
	return chainCfg, blocks, oracle
}

type stubOracle struct {
	blocks map[common.Hash]*types.Block
	db     ethdb.Database
}

func newStubOracle(chain []*types.Block, db ethdb.Database) *stubOracle {
	blocks := make(map[common.Hash]*types.Block, len(chain))
	for _, block := range chain {
		blocks[block.Hash()] = block
	}
	return &stubOracle{
		blocks: blocks,
		db:     db,
	}
}

func (o stubOracle) NodeByHash(ctx context.Context, nodeHash common.Hash) ([]byte, error) {
	return o.db.Get(nodeHash[:])
}

func (o stubOracle) BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return o.blocks[blockHash], nil
}

func TestEngineAPITests(t *testing.T) {
	test.RunEngineAPITests(t, func() engineapi.EngineBackend {
		_, chain := setupOracleBackedChain(t, 0)
		return chain
	})
}
