package l2

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-program/l2/engineapi"
	"github.com/ethereum-optimism/optimism/op-program/l2/engineapi/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
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
		assertBlockDataAvailable(t, chain, block, uint64(i))
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

func assertBlockDataAvailable(t *testing.T, chain *OracleBackedL2Chain, block *types.Block, blockNumber uint64) {
	require.Equal(t, block, chain.GetBlockByHash(block.Hash()), "get block %v by hash", blockNumber)
	require.Equal(t, block.Header(), chain.GetHeaderByHash(block.Hash()), "get header %v by hash", blockNumber)
	require.Equal(t, block, chain.GetBlock(block.Hash(), blockNumber), "get block %v by hash and number", blockNumber)
	require.Equal(t, block.Header(), chain.GetHeader(block.Hash(), blockNumber), "get header %v by hash and number", blockNumber)
	require.True(t, chain.HasBlockAndState(block.Hash(), blockNumber), "has block and state for block %v", blockNumber)
	require.Equal(t, block.Hash(), chain.GetCanonicalHash(blockNumber), "get canonical hash for block %v", blockNumber)
}

func setupOracleBackedChain(t *testing.T, blockCount int) ([]*types.Block, *OracleBackedL2Chain) {
	return setupOracleBackedChainWithLowerHead(t, blockCount, blockCount)
}

func setupOracleBackedChainWithLowerHead(t *testing.T, blockCount int, headBlockNumber int) ([]*types.Block, *OracleBackedL2Chain) {
	logger := testlog.Logger(t, log.LvlDebug)
	chainCfg, blocks, oracle := setupOracle(blockCount)
	head := blocks[headBlockNumber].Hash()
	chain, err := NewOracleBackedL2Chain(context.Background(), logger, oracle, chainCfg, head)
	require.NoError(t, err)
	return blocks, chain
}

func setupOracle(blockCount int) (*params.ChainConfig, []*types.Block, *stubOracle) {
	regolithTime := uint64(0)
	chainCfg := &params.ChainConfig{
		ChainID:                       big.NewInt(4),
		HomesteadBlock:                big.NewInt(0),
		DAOForkBlock:                  nil,
		DAOForkSupport:                false,
		EIP150Block:                   big.NewInt(0),
		EIP150Hash:                    common.Hash{},
		EIP155Block:                   big.NewInt(0),
		EIP158Block:                   big.NewInt(0),
		ByzantiumBlock:                big.NewInt(0),
		ConstantinopleBlock:           big.NewInt(0),
		PetersburgBlock:               big.NewInt(0),
		IstanbulBlock:                 big.NewInt(0),
		MuirGlacierBlock:              big.NewInt(0),
		BerlinBlock:                   big.NewInt(0),
		LondonBlock:                   big.NewInt(0),
		ArrowGlacierBlock:             big.NewInt(0),
		GrayGlacierBlock:              big.NewInt(0),
		MergeNetsplitBlock:            big.NewInt(0),
		TerminalTotalDifficulty:       common.Big0,
		TerminalTotalDifficultyPassed: true,
		Optimism: &params.OptimismConfig{
			EIP1559Denominator: uint64(7),
			EIP1559Elasticity:  uint64(2),
		},
		BedrockBlock: big.NewInt(0),
		RegolithTime: &regolithTime,
	}
	// Set minimal amount of stuff to avoid nil references later
	genesisBlock := types.NewBlock(&types.Header{
		Difficulty: common.Big0,
		Number:     common.Big0,
		GasLimit:   30_000_000,
		GasUsed:    0,
		Time:       10000,
		BaseFee:    big.NewInt(7),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	consensus := beacon.New(nil)
	db := rawdb.NewMemoryDatabase()
	blocks, _ := core.GenerateChain(chainCfg, genesisBlock, consensus, db, blockCount, func(i int, gen *core.BlockGen) {})
	blocks = append([]*types.Block{genesisBlock}, blocks...)
	oracle := newStubOracle(blocks)
	return chainCfg, blocks, oracle
}

type stubOracle struct {
	blocks map[common.Hash]*types.Block
}

func newStubOracle(chain []*types.Block) *stubOracle {
	blocks := make(map[common.Hash]*types.Block, len(chain))
	for _, block := range chain {
		blocks[block.Hash()] = block
	}
	return &stubOracle{blocks: blocks}
}

func (o stubOracle) NodeByHash(ctx context.Context, nodeHash common.Hash) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (o stubOracle) BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return o.blocks[blockHash], nil
}

func TestEngineAPITests(t *testing.T) {
	// TODO: Unskip these
	//t.Skip("Not enough functionality supported to run these")
	test.RunEngineAPITests(t, func() engineapi.EngineBackend {
		_, chain := setupOracleBackedChain(t, 0)
		return chain
	})
}
