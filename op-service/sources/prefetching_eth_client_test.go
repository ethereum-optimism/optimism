package sources

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/stretchr/testify/require"
)

// Define a test type for each method
type testFunction func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mock *testutils.MockEthClient)

// Define test cases
var testCases = []struct {
	name              string
	prefetchingRanges []uint64
	testFunc          testFunction
}{
	{
		name:              "InfoByNumber",
		prefetchingRanges: []uint64{0, 1, 5},
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mock *testutils.MockEthClient) {
			_, rhdr := randHeader()
			expectedInfo, err := rhdr.Info(true, false)
			require.NoError(t, err)

			windowEnd := (uint64(rhdr.Number) + client.PrefetchingRange)
			for i := uint64(rhdr.Number); i <= windowEnd; i++ {
				// tweak header per iteration
				mock.ExpectInfoByNumber(i, expectedInfo, nil)
				// set up different receipts per info - use ExpectReceipts
			}

			mock.SetupDefaultFetchReceipts(expectedInfo, nil)
			info, err := client.InfoByNumber(ctx, uint64(rhdr.Number))
			require.NoError(t, err)
			require.Equal(t, info, expectedInfo)
		},
	},
	// {
	// 	name:              "InfoByHash",
	// 	prefetchingRanges: []uint64{0, 1, 5},
	// 	testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
	// 		hash := randHash()
	// 		_, rhdr := randHeader()
	// 		rhdr.Hash = hash
	// 		expectedInfo, _ := rhdr.Info(true, false)
	// 		mockRPC.On("InfoByHash", ctx, hash).Return(expectedInfo, nil).Once()
	// 		info, err := client.InfoByHash(ctx, hash)
	// 		require.NoError(t, err)
	// 		require.Equal(t, expectedInfo, info)
	// 		return nil
	// 	},
	// },
	// {
	// 	name:              "InfoByLabel",
	// 	prefetchingRanges: []uint64{0, 1, 5},
	// 	testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
	// 		label := eth.BlockLabel(eth.Unsafe)
	// 		_, rhdr := randHeader()
	// 		expectedInfo, _ := rhdr.Info(true, false)
	// 		mockRPC.On("InfoByLabel", ctx, label).Return(expectedInfo, nil).Once()
	// 		info, err := client.InfoByLabel(ctx, label)
	// 		require.NoError(t, err)
	// 		require.Equal(t, expectedInfo, info)
	// 		return nil
	// 	},
	// },
	// {
	// 	name:              "InfoAndTxsByHash",
	// 	prefetchingRanges: []uint64{0, 1, 5},
	// 	testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
	// 		hash := common.Hash{}
	// 		block := &types.Block{}
	// 		blockInfo := eth.BlockToInfo(block)
	// 		txs := types.Transactions{}
	// 		mockRPC.On("InfoAndTxsByHash", ctx, hash).Return(blockInfo, txs, nil).Once()
	// 		_, _, err := client.InfoAndTxsByHash(ctx, hash)
	// 		return err
	// 	},
	// },
	// {
	// 	name:              "InfoAndTxsByNumber",
	// 	prefetchingRanges: []uint64{0, 1, 5},
	// 	testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
	// 		number := uint64(1234)
	// 		block := &types.Block{}
	// 		blockInfo := eth.BlockToInfo(block)
	// 		txs := types.Transactions{}
	// 		mockRPC.On("InfoAndTxsByNumber", ctx, number).Return(blockInfo, txs, nil).Once()
	// 		_, _, err := client.InfoAndTxsByNumber(ctx, number)
	// 		return err
	// 	},
	// },
	// {
	// 	name:              "PayloadByHash",
	// 	prefetchingRanges: []uint64{0, 1, 5},
	// 	testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
	// 		hash := common.Hash{}
	// 		payload := &eth.ExecutionPayload{}
	// 		mockRPC.On("PayloadByHash", ctx, hash).Return(payload, nil).Once()
	// 		_, err := client.PayloadByHash(ctx, hash)
	// 		return err
	// 	},
	// },
	// {
	// 	name:              "PayloadByNumber",
	// 	prefetchingRanges: []uint64{0, 1, 5},
	// 	testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
	// 		number := uint64(1234)
	// 		payload := &eth.ExecutionPayload{}
	// 		mockRPC.On("PayloadByNumber", ctx, number).Return(payload, nil).Once()
	// 		_, err := client.PayloadByNumber(ctx, number)
	// 		return err
	// 	},
	// },
	// {
	// 	name:              "PayloadByLabel",
	// 	prefetchingRanges: []uint64{0, 1, 5},
	// 	testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
	// 		label := eth.BlockLabel(eth.Unsafe)
	// 		payload := &eth.ExecutionPayload{}
	// 		mockRPC.On("PayloadByLabel", ctx, label).Return(payload, nil).Once()
	// 		_, err := client.PayloadByLabel(ctx, label)
	// 		return err
	// 	},
	// },
	// {
	// 	name:              "FetchReceipts",
	// 	prefetchingRanges: []uint64{0, 1, 5},
	// 	testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
	// 		blockHash := common.Hash{}
	// 		blockInfo := eth.BlockToInfo(&types.Block{})
	// 		receipts := types.Receipts{}
	// 		mockRPC.On("FetchReceipts", ctx, blockHash).Return(blockInfo, receipts, nil).Once()
	// 		_, _, err := client.FetchReceipts(ctx, blockHash)
	// 		return err
	// 	},
	// },
}

// runTest runs a given test function with a specific prefetching range.
func runTest(t *testing.T, name string, testFunc testFunction, prefetchingRange uint64) {
	ctx := context.Background()
	mockEthClient := new(testutils.MockEthClient)
	client, err := NewPrefetchingEthClient(mockEthClient, prefetchingRange, 30*time.Second)
	require.NoError(t, err)

	testFunc(t, ctx, client, mockEthClient)

	mockEthClient.AssertExpectations(t)
}

// TestPrefetchingEthClient runs all test cases for each prefetching range.
func TestPrefetchingEthClient(t *testing.T) {
	for _, tc := range testCases {
		for _, prefetchingRange := range tc.prefetchingRanges {
			t.Run(tc.name+"_with_prefetching_range_"+strconv.Itoa(int(prefetchingRange)), func(t *testing.T) {
				runTest(t, tc.name, tc.testFunc, prefetchingRange)
			})
		}
	}
}
