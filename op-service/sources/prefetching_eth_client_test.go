package sources

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Define a test type for each method
type testFunction func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error

// Define test cases
var testCases = []struct {
	name              string
	prefetchingRanges []uint64
	testFunc          testFunction
}{
	{
		name:              "InfoByNumber",
		prefetchingRanges: []uint64{0, 1, 5},
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			_, rhdr := randHeader()
			expectedInfo, _ := rhdr.Info(true, false)
			n := rhdr.Number
			// Mock the call to CallContext for 'eth_getBlockByNumber', expecting it to be called at least once
			mockRPC.On("CallContext", ctx, new(*rpcHeader),
				"eth_getBlockByNumber", []any{n.String(), false}).Run(func(args mock.Arguments) {
				*args[1].(**rpcHeader) = rhdr
			}).Return([]error{nil}).Maybe()

			// Call the method which is expected to internally call 'eth_getBlockByNumber'
			info, err := client.InfoByNumber(ctx, uint64(rhdr.Number))
			require.NoError(t, err)
			require.Equal(t, info, expectedInfo)
			// Assert that 'eth_getBlockByNumber' was called at least once
			mockRPC.AssertCalled(t, "CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.Anything)

			return nil
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
	mockRPC := new(mockRPC)
	ethClient, err := NewEthClient(mockRPC, nil, nil, testEthClientConfig)
	require.NoError(t, err)
	client, err := NewPrefetchingEthClient(ethClient, prefetchingRange)
	require.NoError(t, err)

	err = testFunc(t, ctx, client, mockRPC)
	require.NoError(t, err)

	mockRPC.AssertExpectations(t)
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
