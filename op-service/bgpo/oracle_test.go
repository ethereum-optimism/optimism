package bgpo

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type mockBTOBackend struct {
	mock.Mock
}

func (m *mockBTOBackend) BlockNumber(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *mockBTOBackend) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	args := m.Called(ctx, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Header), args.Error(1)
}

func (m *mockBTOBackend) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	args := m.Called(ctx, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Block), args.Error(1)
}

func (m *mockBTOBackend) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	args := m.Called(ctx, ch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ethereum.Subscription), args.Error(1)
}

var _ BTOBackend = (*mockBTOBackend)(nil)

// mockSubscription implements ethereum.Subscription for testing.
type mockSubscription struct {
	errCh    chan error
	unsubbed bool
}

func newMockSubscription() *mockSubscription {
	return &mockSubscription{
		errCh: make(chan error, 1),
	}
}

func (s *mockSubscription) Unsubscribe() {
	if !s.unsubbed {
		s.unsubbed = true
	}
}

func (s *mockSubscription) Err() <-chan error {
	return s.errCh
}

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

func createBlock(blockNum uint64, baseFee *big.Int, txs []*types.Transaction) *types.Block {
	header := &types.Header{
		Number:  big.NewInt(int64(blockNum)),
		BaseFee: baseFee,
		Time:    uint64(time.Now().Unix()),
	}
	return types.NewBlock(header, &types.Body{Transactions: txs}, nil, trie.NewStackTrie(nil), types.DefaultBlockConfig)
}

func TestNewBlobGasPriceOracle(t *testing.T) {
	mbackend := new(mockBTOBackend)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelInfo)

	t.Run("with nil config", func(t *testing.T) {
		oracle := NewBlobTipOracle(mbackend, chainConfig, logger, nil)
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
		oracle := NewBlobTipOracle(mbackend, chainConfig, logger, config)
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
		oracle := NewBlobTipOracle(mbackend, chainConfig, logger, config)
		require.NotNil(t, oracle)
		// Should use defaults
		require.Equal(t, 20, oracle.config.MaxBlocks)
		require.Equal(t, 60, oracle.config.Percentile)
	})
}

func TestProcessHeader(t *testing.T) {
	mbackend := new(mockBTOBackend)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(mbackend, chainConfig, logger, &BlobTipOracleConfig{
		PricesCacheSize: 10,
		BlockCacheSize:  10,
		MaxBlocks:       5,
		Percentile:      60,
	})

	t.Run("process header with excess blob gas", func(t *testing.T) {
		excessBlobGas := uint64(1000000)
		header := createHeader(100, &excessBlobGas)

		// Mock block fetch for blob fee caps
		emptyBlock := createBlock(100, header.BaseFee, []*types.Transaction{})
		mbackend.On("BlockByNumber", mock.Anything, big.NewInt(100)).Return(emptyBlock, nil).Once()

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
		emptyBlock := createBlock(101, header.BaseFee, []*types.Transaction{})
		mbackend.On("BlockByNumber", mock.Anything, big.NewInt(101)).Return(emptyBlock, nil).Once()

		err := oracle.processHeader(header)
		require.NoError(t, err)

		// Latest block should be updated
		latestBlock, _ := oracle.GetLatestBlobBaseFee()
		require.Equal(t, uint64(101), latestBlock)
	})

	mbackend.AssertExpectations(t)
}

func TestGetLatestBlobBaseFee(t *testing.T) {
	mbackend := new(mockBTOBackend)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(mbackend, chainConfig, logger, &BlobTipOracleConfig{
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

		emptyBlock1 := createBlock(300, header1.BaseFee, []*types.Transaction{})
		emptyBlock2 := createBlock(301, header2.BaseFee, []*types.Transaction{})

		mbackend.On("BlockByNumber", mock.Anything, big.NewInt(300)).Return(emptyBlock1, nil).Once()
		mbackend.On("BlockByNumber", mock.Anything, big.NewInt(301)).Return(emptyBlock2, nil).Once()

		err := oracle.processHeader(header1)
		require.NoError(t, err)

		err = oracle.processHeader(header2)
		require.NoError(t, err)

		block, fee := oracle.GetLatestBlobBaseFee()
		require.Equal(t, uint64(301), block)
		require.NotNil(t, fee)
	})

	mbackend.AssertExpectations(t)
}

func TestSuggestBlobTipCap(t *testing.T) {
	ctx := context.Background()
	mbackend := new(mockBTOBackend)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(mbackend, chainConfig, logger, &BlobTipOracleConfig{
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

			block := createBlock(i, header.BaseFee, []*types.Transaction{blobTx})
			mbackend.On("BlockByNumber", mock.Anything, big.NewInt(int64(i))).Return(block, nil).Once()

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
		oracle2 := NewBlobTipOracle(mbackend, chainConfig, logger, &BlobTipOracleConfig{
			PricesCacheSize:    10,
			BlockCacheSize:     10,
			MaxBlocks:          5,
			Percentile:         60,
			DefaultPriorityFee: big.NewInt(101),
		})

		excessBlobGas := uint64(1000000)
		header := createHeader(500, &excessBlobGas)

		emptyBlock := createBlock(500, header.BaseFee, []*types.Transaction{})
		mbackend.On("BlockByNumber", mock.Anything, big.NewInt(500)).Return(emptyBlock, nil).Once()

		err := oracle2.processHeader(header)
		require.NoError(t, err)

		suggested, err := oracle2.SuggestBlobTipCap(ctx, 0, 0)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(101), suggested)
	})

	mbackend.AssertExpectations(t)
}

func TestPrePopulateCache(t *testing.T) {
	mbackend := new(mockBTOBackend)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(mbackend, chainConfig, logger, &BlobTipOracleConfig{
		PricesCacheSize: 10,
		BlockCacheSize:  10,
		MaxBlocks:       3,
		Percentile:      60,
	})

	t.Run("pre-populate with recent blocks", func(t *testing.T) {
		latestBlock := uint64(1000)

		// Mock eth_blockNumber
		mbackend.On("BlockNumber", mock.Anything).Return(latestBlock, nil).Once()

		// Mock header and block fetches for blocks 998, 999, 1000
		excessBlobGas := uint64(1000000)
		for i := uint64(998); i <= 1000; i++ {
			header := createHeader(i, &excessBlobGas)
			block := createBlock(i, header.BaseFee, []*types.Transaction{})

			mbackend.On("HeaderByNumber", mock.Anything, big.NewInt(int64(i))).Return(header, nil).Once()
			mbackend.On("BlockByNumber", mock.Anything, big.NewInt(int64(i))).Return(block, nil).Once()
		}

		err := oracle.prePopulateCache()
		require.NoError(t, err)

		latestBlockNum, _ := oracle.GetLatestBlobBaseFee()
		require.Equal(t, uint64(1000), latestBlockNum)
	})

	mbackend.AssertExpectations(t)
}

func TestExtractBlobFeeCaps(t *testing.T) {
	mbackend := new(mockBTOBackend)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelError)

	oracle := NewBlobTipOracle(mbackend, chainConfig, logger, &BlobTipOracleConfig{
		PricesCacheSize: 10,
		BlockCacheSize:  10,
	})

	t.Run("extract_from_blob_transactions", func(t *testing.T) {
		baseFee := big.NewInt(2)      // 2 wei
		blobFeeCap := big.NewInt(300) // 300 wei
		gasFeeCap := big.NewInt(300)  // 300 wei
		block := createBlock(600, baseFee, []*types.Transaction{
			createBlobTx(big.NewInt(7), gasFeeCap, blobFeeCap),
			createBlobTx(big.NewInt(8), gasFeeCap, blobFeeCap),
			createBlobTx(big.NewInt(9), gasFeeCap, blobFeeCap),
			createBlobTx(big.NewInt(400), gasFeeCap, blobFeeCap),
		})

		tips := oracle.extractTipsForBlobTxs(block, baseFee)
		require.Len(t, tips, 4)
		require.Equal(t, big.NewInt(7), tips[0])
		require.Equal(t, big.NewInt(8), tips[1])
		require.Equal(t, big.NewInt(9), tips[2])
		require.Equal(t, big.NewInt(298), tips[3]) // gasFeeCap - baseFee; limited to gasFeeCap, even though the blob tip cap is 400 wei
	})

	t.Run("extract ignores non-blob transactions", func(t *testing.T) {
		baseFee := big.NewInt(1000000)
		block := createBlock(601, baseFee, []*types.Transaction{
			types.NewTx(&types.LegacyTx{
				Nonce:    0,
				GasPrice: big.NewInt(1000000),
				Gas:      21000,
				To:       &common.Address{},
				Value:    big.NewInt(0),
				Data:     []byte{},
			}),
		})

		feeCaps := oracle.extractTipsForBlobTxs(block, baseFee)
		require.Len(t, feeCaps, 0)
	})

	t.Run("extract_from_mixed_transactions", func(t *testing.T) {
		baseFee := big.NewInt(1000000)
		blobFeeCap := big.NewInt(3000000000)
		gasFeeCap := big.NewInt(3000000000)
		block := createBlock(602, baseFee, []*types.Transaction{
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
		})

		tips := oracle.extractTipsForBlobTxs(block, baseFee)
		require.Len(t, tips, 2)
		require.Equal(t, big.NewInt(5000000), tips[0])
		require.Equal(t, big.NewInt(6000000), tips[1])
	})
}

func TestOracleLifecycle(t *testing.T) {
	mbackend := new(mockBTOBackend)
	chainConfig := params.MainnetChainConfig
	logger := testlog.Logger(t, log.LevelDebug)

	oracle := NewBlobTipOracle(mbackend, chainConfig, logger, &BlobTipOracleConfig{
		PricesCacheSize: 10,
		BlockCacheSize:  10,
		MaxBlocks:       2,
		Percentile:      60,
		NetworkTimeout:  time.Second,
	})

	t.Run("start and close", func(t *testing.T) {
		latestBlock := uint64(100)

		// Mock pre-population calls
		mbackend.On("BlockNumber", mock.Anything).Return(latestBlock, nil).Once()

		excessBlobGas := uint64(1000000)
		for i := uint64(99); i <= 100; i++ {
			header := createHeader(i, &excessBlobGas)
			block := createBlock(i, header.BaseFee, []*types.Transaction{})
			mbackend.On("HeaderByNumber", mock.Anything, big.NewInt(int64(i))).Return(header, nil).Once()
			mbackend.On("BlockByNumber", mock.Anything, big.NewInt(int64(i))).Return(block, nil).Once()
		}

		// Mock subscription
		sub := newMockSubscription()
		var headerCh chan<- *types.Header
		mbackend.On("SubscribeNewHead", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			headerCh = args.Get(1).(chan<- *types.Header)
		}).Return(sub, nil).Once()

		// Start the oracle
		err := oracle.Start()
		require.NoError(t, err)
		select {
		case <-oracle.loopDone:
			require.Fail(t, "oracle loop should not be done")
		default:
			// Expect loop to block done channel
		}

		// Verify cache was pre-populated
		latestBlockNum, fee := oracle.GetLatestBlobBaseFee()
		require.Equal(t, uint64(100), latestBlockNum)
		require.NotNil(t, fee)

		// Send a new header through the subscription to verify processing works
		newHeader := createHeader(101, &excessBlobGas)
		newBlock := createBlock(101, newHeader.BaseFee, []*types.Transaction{})
		mbackend.On("BlockByNumber", mock.Anything, big.NewInt(101)).Return(newBlock, nil).Once()

		headerCh <- newHeader

		// Give the goroutine time to process
		require.Eventually(t, func() bool {
			latestBlockNum, _ = oracle.GetLatestBlobBaseFee()
			return latestBlockNum == 101
		}, time.Second, 10*time.Millisecond)

		// Close the oracle
		oracle.Close()

		// Verify subscription was unsubscribed
		require.True(t, sub.unsubbed, "subscription should be unsubscribed after Close")
		select {
		case <-oracle.loopDone:
			// Expect loop to have exited
		default:
			require.Fail(t, "oracle loop should have exited after Close")
		}

		mbackend.AssertExpectations(t)
	})

	t.Run("close before start is safe", func(t *testing.T) {
		oracle2 := NewBlobTipOracle(mbackend, chainConfig, logger, &BlobTipOracleConfig{
			PricesCacheSize: 10,
			BlockCacheSize:  10,
		})

		// Should not panic
		oracle2.Close()
	})
}
