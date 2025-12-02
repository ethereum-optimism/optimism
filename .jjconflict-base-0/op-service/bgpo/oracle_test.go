package bgpo

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type mockRPC struct {
	mock.Mock
}

func (m *mockRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	callArgs := make([]any, 0, len(args))
	callArgs = append(callArgs, args...)
	args_ := m.Called(ctx, result, method, callArgs)
	return args_.Error(0)
}

func (m *mockRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	args_ := m.Called(ctx, b)
	return args_.Error(0)
}

func (m *mockRPC) Subscribe(ctx context.Context, namespace string, channel any, args ...any) (ethereum.Subscription, error) {
	args_ := m.Called(ctx, namespace, channel, args)
	sub := args_.Get(0)
	if sub == nil {
		return nil, args_.Error(1)
	}
	return sub.(ethereum.Subscription), args_.Error(1)
}

func (m *mockRPC) Close() {
	m.Called()
}

var _ client.RPC = (*mockRPC)(nil)

func createHeader(blockNum uint64, excessBlobGas *uint64) *types.Header {
	header := &types.Header{
		Number:     big.NewInt(int64(blockNum)),
		ParentHash: common.Hash{},
		Time:       uint64(time.Now().Unix()),
		BaseFee:    big.NewInt(1000000000), // 1 gwei
	}
	if excessBlobGas != nil {
		header.ExcessBlobGas = excessBlobGas
	}
	return header
}

func createBlobTx(gasTip *big.Int, gasFeeCap *big.Int, blobFeeCap *big.Int) *types.Transaction {
	// Create a minimal blob transaction
	// Note: This is a simplified version for testing
	tx := types.NewTx(&types.BlobTx{
		ChainID:    uint256.NewInt(1),
		Nonce:      0,
		GasTipCap:  uint256.MustFromBig(gasTip),
		GasFeeCap:  uint256.MustFromBig(gasFeeCap),
		Gas:        21000,
		To:         common.Address{},
		Value:      uint256.NewInt(0),
		Data:       []byte{},
		BlobFeeCap: uint256.MustFromBig(blobFeeCap),
		BlobHashes: []common.Hash{common.Hash{}},
	})
	return tx
}

func TestNewBlobGasPriceOracle(t *testing.T) {
	ctx := context.Background()
	mrpc := new(mockRPC)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelInfo)

	t.Run("with nil config", func(t *testing.T) {
		oracle := NewBlobTipOracle(ctx, mrpc, chainConfig, logger, nil)
		require.NotNil(t, oracle)
		require.Equal(t, 20, oracle.config.MaxBlocks)
		require.Equal(t, 60, oracle.config.Percentile)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &BlobTipOracleConfig{
			PricesCacheSize: 500,
			BlockCacheSize:  50,
			MaxBlocks:       10,
			Percentile:      70,
		}
		oracle := NewBlobTipOracle(ctx, mrpc, chainConfig, logger, config)
		require.NotNil(t, oracle)
		require.Equal(t, 10, oracle.config.MaxBlocks)
		require.Equal(t, 70, oracle.config.Percentile)
	})

	t.Run("with invalid config values", func(t *testing.T) {
		config := &BlobTipOracleConfig{
			PricesCacheSize: -1,
			BlockCacheSize:  -1,
			MaxBlocks:       -1,
			Percentile:      150, // Invalid
		}
		oracle := NewBlobTipOracle(ctx, mrpc, chainConfig, logger, config)
		require.NotNil(t, oracle)
		// Should use defaults
		require.Equal(t, 20, oracle.config.MaxBlocks)
		require.Equal(t, 60, oracle.config.Percentile)
	})
}

func TestProcessHeader(t *testing.T) {
	ctx := context.Background()
	mrpc := new(mockRPC)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(ctx, mrpc, chainConfig, logger, &BlobTipOracleConfig{
		PricesCacheSize: 10,
		BlockCacheSize:  10,
		MaxBlocks:       5,
		Percentile:      60,
	})

	t.Run("process header with excess blob gas", func(t *testing.T) {
		excessBlobGas := uint64(1000000)
		header := createHeader(100, &excessBlobGas)

		// Mock block fetch for blob fee caps
		mrpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.MatchedBy(func(args []any) bool {
			return len(args) == 2 && args[1] == true
		})).
			Run(func(args mock.Arguments) {
				block := args[1].(*rpcBlock)
				block.Number = hexutil.Uint64(100)
				block.Hash = common.Hash{}.Bytes()
				block.Transactions = []*types.Transaction{}
			}).
			Return(nil).Once()

		err := oracle.processHeader(header)
		require.NoError(t, err)

		// Check latest block
		latestBlock, latestFee := oracle.GetLatestBlobBaseFee()
		require.Equal(t, uint64(100), latestBlock)
		require.NotNil(t, latestFee)
	})

	t.Run("process header without excess blob gas", func(t *testing.T) {
		header := createHeader(101, nil)

		// Mock block fetch
		mrpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.MatchedBy(func(args []any) bool {
			return len(args) == 2 && args[1] == true
		})).
			Run(func(args mock.Arguments) {
				block := args[1].(*rpcBlock)
				block.Number = hexutil.Uint64(101)
				block.Hash = common.Hash{}.Bytes()
				block.Transactions = []*types.Transaction{}
			}).
			Return(nil).Once()

		err := oracle.processHeader(header)
		require.NoError(t, err)

		// Latest block should be updated
		latestBlock, _ := oracle.GetLatestBlobBaseFee()
		require.Equal(t, uint64(101), latestBlock)
	})

	mrpc.AssertExpectations(t)
}

func TestGetLatestBlobBaseFee(t *testing.T) {
	ctx := context.Background()
	mrpc := new(mockRPC)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(ctx, mrpc, chainConfig, logger, &BlobTipOracleConfig{
		PricesCacheSize: 10,
		BlockCacheSize:  10,
	})

	t.Run("no blocks processed", func(t *testing.T) {
		block, fee := oracle.GetLatestBlobBaseFee()
		require.Equal(t, uint64(0), block)
		require.Nil(t, fee)
	})

	t.Run("with processed blocks", func(t *testing.T) {
		excessBlobGas := uint64(1000000)
		header1 := createHeader(300, &excessBlobGas)
		header2 := createHeader(301, &excessBlobGas)

		mrpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.MatchedBy(func(args []any) bool {
			return len(args) == 2 && args[1] == true
		})).
			Return(nil).Twice().
			Run(func(args mock.Arguments) {
				block := args[1].(*rpcBlock)
				callArgs := args[3].([]any)
				blockNumHex := callArgs[0].(string)
				if blockNumHex == "0x12c" { // 300
					block.Number = hexutil.Uint64(300)
				} else {
					block.Number = hexutil.Uint64(301)
				}
				block.Hash = common.Hash{}.Bytes()
				block.Transactions = []*types.Transaction{}
			})

		err := oracle.processHeader(header1)
		require.NoError(t, err)

		err = oracle.processHeader(header2)
		require.NoError(t, err)

		block, fee := oracle.GetLatestBlobBaseFee()
		require.Equal(t, uint64(301), block)
		require.NotNil(t, fee)
	})

	mrpc.AssertExpectations(t)
}

func TestSuggestBlobTipCap(t *testing.T) {
	ctx := context.Background()
	mrpc := new(mockRPC)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(ctx, mrpc, chainConfig, logger, &BlobTipOracleConfig{
		PricesCacheSize: 10,
		BlockCacheSize:  10,
		MaxBlocks:       5,
		Percentile:      60,
	})

	t.Run("no blocks processed", func(t *testing.T) {
		suggested, err := oracle.SuggestBlobTipCap(ctx, 0, 0)
		require.Error(t, err)
		require.Nil(t, suggested)
		require.Contains(t, err.Error(), "no blocks have been processed")
	})

	t.Run("with_blob_transactions", func(t *testing.T) {
		// Process blocks with blob transactions
		excessBlobGas := uint64(1000000)
		for i := uint64(400); i <= 404; i++ {
			header := createHeader(i, &excessBlobGas)

			// Create blob transactions with different tip
			gasFeeCap := big.NewInt(3000000000)
			blobFeeCap := big.NewInt(3000000000)
			tip := big.NewInt(int64((i-400)*1000000 + 1000000)) // 1M, 2M, 3M, 4M, 5M
			blobTx := createBlobTx(tip, gasFeeCap, blobFeeCap)

			mrpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.MatchedBy(func(args []any) bool {
				return len(args) == 2 && args[1] == true
			})).
				Run(func(args mock.Arguments) {
					block := args[1].(*rpcBlock)
					block.Number = hexutil.Uint64(i)
					block.Hash = common.Hash{}.Bytes()
					block.Transactions = []*types.Transaction{blobTx}
				}).
				Return(nil).Once()

			err := oracle.processHeader(header)
			require.NoError(t, err)
		}

		// Test with default parameters
		suggested, err := oracle.SuggestBlobTipCap(ctx, 0, 0)
		require.NoError(t, err)
		require.NotNil(t, suggested)
		// Should be 60th percentile of [1M, 2M, 3M, 4M, 5M] = 3M (index 2 of 4)
		require.Equal(t, big.NewInt(3000000), suggested)

		// Test with custom percentile
		suggested, err = oracle.SuggestBlobTipCap(ctx, 5, 80)
		require.NoError(t, err)
		require.NotNil(t, suggested)
		// 80th percentile of [1M, 2M, 3M, 4M, 5M] = 4M (index 3 of 4)
		require.Equal(t, big.NewInt(4000000), suggested)
	})

	t.Run("no blob transactions, fallback to base fee", func(t *testing.T) {
		oracle2 := NewBlobTipOracle(ctx, mrpc, chainConfig, logger, &BlobTipOracleConfig{
			PricesCacheSize:    10,
			BlockCacheSize:     10,
			MaxBlocks:          5,
			Percentile:         60,
			DefaultPriorityFee: big.NewInt(101),
		})

		excessBlobGas := uint64(1000000)
		header := createHeader(500, &excessBlobGas)

		mrpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.MatchedBy(func(args []any) bool {
			return len(args) == 2 && args[1] == true
		})).
			Run(func(args mock.Arguments) {
				block := args[1].(*rpcBlock)
				block.Number = hexutil.Uint64(500)
				block.Hash = common.Hash{}.Bytes()
				block.Transactions = []*types.Transaction{} // No blob transactions
			}).
			Return(nil).Once()

		err := oracle2.processHeader(header)
		require.NoError(t, err)

		suggested, err := oracle2.SuggestBlobTipCap(ctx, 0, 0)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(101), suggested)
	})

	mrpc.AssertExpectations(t)
}

func TestPrePopulateCache(t *testing.T) {
	ctx := context.Background()
	mrpc := new(mockRPC)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(ctx, mrpc, chainConfig, logger, &BlobTipOracleConfig{
		PricesCacheSize: 10,
		BlockCacheSize:  10,
		MaxBlocks:       3,
		Percentile:      60,
	})

	t.Run("pre-populate with recent blocks", func(t *testing.T) {
		latestBlock := uint64(1000)

		// Mock eth_blockNumber (called with no args - empty slice)
		mrpc.On("CallContext", mock.Anything, mock.Anything, "eth_blockNumber", mock.MatchedBy(func(args []any) bool {
			return len(args) == 0
		})).
			Run(func(args mock.Arguments) {
				result := args[1].(*hexutil.Uint64)
				*result = hexutil.Uint64(latestBlock)
			}).
			Return(nil).Once()

		// Mock header fetches for blocks 998, 999, 1000
		excessBlobGas := uint64(1000000)
		for i := uint64(998); i <= 1000; i++ {
			header := createHeader(i, &excessBlobGas)

			// Mock header fetch (with false for full transactions)
			mrpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.MatchedBy(func(args []any) bool {
				return len(args) == 2 && args[0] == hexutil.EncodeUint64(i) && args[1] == false
			})).
				Run(func(args mock.Arguments) {
					result := args[1].(**types.Header)
					*result = header
				}).
				Return(nil).Once()

			// Mock block fetch for blob fee caps (with true for full transactions)
			mrpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.MatchedBy(func(args []any) bool {
				return len(args) == 2 && args[0] == hexutil.EncodeUint64(i) && args[1] == true
			})).
				Run(func(args mock.Arguments) {
					block := args[1].(*rpcBlock)
					block.Number = hexutil.Uint64(i)
					block.Hash = common.Hash{}.Bytes()
					block.Transactions = []*types.Transaction{}
				}).
				Return(nil).Once()
		}

		err := oracle.prePopulateCache()
		require.NoError(t, err)

		latestBlockNum, _ := oracle.GetLatestBlobBaseFee()
		require.Equal(t, uint64(1000), latestBlockNum)
	})

	mrpc.AssertExpectations(t)
}

func TestExtractBlobFeeCaps(t *testing.T) {
	ctx := context.Background()
	mrpc := new(mockRPC)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(ctx, mrpc, chainConfig, logger, &BlobTipOracleConfig{
		PricesCacheSize: 10,
		BlockCacheSize:  10,
	})

	t.Run("extract_from_blob_transactions", func(t *testing.T) {
		baseFee := big.NewInt(2)      // 2 wei
		blobFeeCap := big.NewInt(300) // 300 wei
		gasFeeCap := big.NewInt(300)  // 300 wei
		block := rpcBlock{
			Number: hexutil.Uint64(600),
			Hash:   common.Hash{}.Bytes(),
			Transactions: []*types.Transaction{
				createBlobTx(big.NewInt(7), gasFeeCap, blobFeeCap),
				createBlobTx(big.NewInt(8), gasFeeCap, blobFeeCap),
				createBlobTx(big.NewInt(9), gasFeeCap, blobFeeCap),
				createBlobTx(big.NewInt(400), gasFeeCap, blobFeeCap),
			},
		}

		tips := oracle.extractTipsForBlobTxs(block, baseFee)
		require.Len(t, tips, 4)
		require.Equal(t, big.NewInt(7), tips[0])
		require.Equal(t, big.NewInt(8), tips[1])
		require.Equal(t, big.NewInt(9), tips[2])
		require.Equal(t, big.NewInt(298), tips[3]) // gasFeeCap - baseFee; limited to gasFeeCap, even though the blob tip cap is 400 wei
	})

	t.Run("extract ignores non-blob transactions", func(t *testing.T) {
		baseFee := big.NewInt(1000000)
		block := rpcBlock{
			Number: hexutil.Uint64(601),
			Hash:   common.Hash{}.Bytes(),
			Transactions: []*types.Transaction{
				types.NewTx(&types.LegacyTx{
					Nonce:    0,
					GasPrice: big.NewInt(1000000),
					Gas:      21000,
					To:       &common.Address{},
					Value:    big.NewInt(0),
					Data:     []byte{},
				}),
			},
		}

		feeCaps := oracle.extractTipsForBlobTxs(block, baseFee)
		require.Len(t, feeCaps, 0)
	})

	t.Run("extract_from_mixed_transactions", func(t *testing.T) {
		baseFee := big.NewInt(1000000)
		blobFeeCap := big.NewInt(3000000000)
		gasFeeCap := big.NewInt(3000000000)
		block := rpcBlock{
			Number: hexutil.Uint64(602),
			Hash:   common.Hash{}.Bytes(),
			Transactions: []*types.Transaction{
				types.NewTx(&types.LegacyTx{
					Nonce:    0,
					GasPrice: big.NewInt(1000000),
					Gas:      21000,
					To:       &common.Address{},
					Value:    big.NewInt(0),
					Data:     []byte{},
				}),
				createBlobTx(big.NewInt(5000000), gasFeeCap, blobFeeCap),
				createBlobTx(big.NewInt(6000000), gasFeeCap, blobFeeCap),
			},
		}

		tips := oracle.extractTipsForBlobTxs(block, baseFee)
		require.Len(t, tips, 2)
		require.Equal(t, big.NewInt(5000000), tips[0])
		require.Equal(t, big.NewInt(6000000), tips[1])
	})
}
