package l2

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-program/client/l2/engineapi"
	"github.com/ethereum-optimism/optimism/op-program/client/l2/engineapi/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

var fundedKey, _ = crypto.GenerateKey()
var fundedAddress = crypto.PubkeyToAddress(fundedKey.PublicKey)
var targetAddress = common.HexToAddress("0x001122334455")

func TestInitialState(t *testing.T) {
	blocks, chain := setupOracleBackedChain(t, 5)
	head := blocks[5]
	require.Equal(t, head.Header(), chain.CurrentHeader())
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

func TestCanonicalHashNotFoundPastChainHead(t *testing.T) {
	blocks, chain := setupOracleBackedChainWithLowerHead(t, 5, 3)

	for i := 0; i <= 3; i++ {
		require.Equal(t, blocks[i].Hash(), chain.GetCanonicalHash(uint64(i)))
		require.Equal(t, blocks[i].Header(), chain.GetHeaderByNumber(uint64(i)))
	}
	for i := 4; i <= 5; i++ {
		require.Equal(t, common.Hash{}, chain.GetCanonicalHash(uint64(i)))
		require.Nil(t, chain.GetHeaderByNumber(uint64(i)))
	}
}

func TestAppendToChain(t *testing.T) {
	blocks, chain := setupOracleBackedChainWithLowerHead(t, 4, 3)
	newBlock := blocks[4]
	require.Nil(t, chain.GetBlock(newBlock.Hash(), newBlock.NumberU64()), "block unknown before being added")

	require.NoError(t, chain.InsertBlockWithoutSetHead(newBlock))
	require.Equal(t, blocks[3].Header(), chain.CurrentHeader(), "should not update chain head yet")
	require.Equal(t, common.Hash{}, chain.GetCanonicalHash(uint64(4)), "not yet a canonical hash")
	require.Nil(t, chain.GetHeaderByNumber(uint64(4)), "not yet a canonical header")
	assertBlockDataAvailable(t, chain, newBlock, 4)

	canonical, err := chain.SetCanonical(newBlock)
	require.NoError(t, err)
	require.Equal(t, newBlock.Hash(), canonical)
	require.Equal(t, newBlock.Hash(), chain.GetCanonicalHash(uint64(4)), "get canonical hash for new head")
	require.Equal(t, newBlock.Header(), chain.GetHeaderByNumber(uint64(4)), "get canonical header for new head")
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
	blocks, chain := setupOracleBackedChain(t, 3)
	newBlock := createBlock(t, chain)

	db, err := chain.StateAt(blocks[1].Root())
	require.NoError(t, err)
	balance := db.GetBalance(fundedAddress)
	require.NotEqual(t, big.NewInt(0), balance, "should have balance at imported block")

	require.NotEqual(t, blocks[1].Root(), newBlock.Root(), "block should have modified world state")

	require.False(t, chain.HasBlockAndState(newBlock.Root(), newBlock.NumberU64()), "state from non-imported block should not be available")

	err = chain.InsertBlockWithoutSetHead(newBlock)
	require.NoError(t, err)
	db, err = chain.StateAt(newBlock.Root())
	require.NoError(t, err, "state should be available after importing")
	balance = db.GetBalance(fundedAddress)
	require.NotEqual(t, big.NewInt(0), balance, "should have balance from imported block")
}

func TestRejectBlockWithStateRootMismatch(t *testing.T) {
	_, chain := setupOracleBackedChain(t, 1)
	newBlock := createBlock(t, chain)
	// Create invalid block by keeping the modified state root but exclude the transaction
	invalidBlock := types.NewBlockWithHeader(newBlock.Header())

	err := chain.InsertBlockWithoutSetHead(invalidBlock)
	require.ErrorContains(t, err, "block root mismatch")
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
	chain, err := NewOracleBackedL2Chain(logger, oracle, chainCfg, head)
	require.NoError(t, err)
	return blocks, chain
}

func setupOracle(t *testing.T, blockCount int, headBlockNumber int) (*params.ChainConfig, []*types.Block, *stubBlockOracle) {
	deployConfig := &genesis.DeployConfig{
		L1ChainID:              900,
		L2ChainID:              901,
		L2BlockTime:            2,
		FundDevAccounts:        true,
		L2GenesisBlockGasLimit: 30_000_000,
	}
	l1Genesis, err := genesis.NewL1Genesis(deployConfig)
	require.NoError(t, err)
	l2Genesis, err := genesis.NewL2Genesis(deployConfig, l1Genesis.ToBlock())
	require.NoError(t, err)

	l2Genesis.Alloc[fundedAddress] = core.GenesisAccount{
		Balance: big.NewInt(1_000_000_000_000_000_000),
		Nonce:   0,
	}
	chainCfg := l2Genesis.Config
	consensus := beacon.New(nil)
	db := rawdb.NewMemoryDatabase()

	// Set minimal amount of stuff to avoid nil references later
	genesisBlock := l2Genesis.MustCommit(db)
	blocks, _ := core.GenerateChain(chainCfg, genesisBlock, consensus, db, blockCount, func(i int, gen *core.BlockGen) {})
	blocks = append([]*types.Block{genesisBlock}, blocks...)
	oracle := newStubOracleWithBlocks(t, blocks[:headBlockNumber+1], db)
	return chainCfg, blocks, oracle
}

func createBlock(t *testing.T, chain *OracleBackedL2Chain) *types.Block {
	parent := chain.GetBlockByHash(chain.CurrentHeader().Hash())
	parentDB, err := chain.StateAt(parent.Root())
	require.NoError(t, err)
	nonce := parentDB.GetNonce(fundedAddress)
	config := chain.Config()
	db := rawdb.NewDatabase(NewOracleBackedDB(chain.oracle))
	blocks, _ := core.GenerateChain(config, parent, chain.Engine(), db, 1, func(i int, gen *core.BlockGen) {
		rawTx := &types.DynamicFeeTx{
			ChainID:   config.ChainID,
			Nonce:     nonce,
			To:        &targetAddress,
			GasTipCap: big.NewInt(0),
			GasFeeCap: parent.BaseFee(),
			Gas:       21_000,
			Value:     big.NewInt(1),
		}
		tx := types.MustSignNewTx(fundedKey, types.NewLondonSigner(config.ChainID), rawTx)
		gen.AddTx(tx)
	})
	return blocks[0]
}

func TestEngineAPITests(t *testing.T) {
	test.RunEngineAPITests(t, func(t *testing.T) engineapi.EngineBackend {
		_, chain := setupOracleBackedChain(t, 0)
		return chain
	})
}
