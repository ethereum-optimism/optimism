package sources

import (
	"context"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// Define a test type for each method
type testFunction func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error

// Define test cases
var testCases = []struct {
	name              string
	testFunc          testFunction
	prefetchingRanges []uint64
}{
	{
		name: "InfoByNumber",
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			_, rhdr := randHeader()
			expectedInfo, _ := rhdr.Info(true, false)
			mockRPC.On("InfoByNumber", ctx, rhdr.Number).Return(expectedInfo, nil).Once()
			info, err := client.InfoByNumber(ctx, uint64(rhdr.Number))
			require.NoError(t, err)
			require.Equal(t, expectedInfo, info)
			return nil
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	{
		name: "InfoByHash",
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			hash := randHash()
			_, rhdr := randHeader()
			rhdr.Hash = hash
			expectedInfo, _ := rhdr.Info(true, false)
			mockRPC.On("InfoByHash", ctx, hash).Return(expectedInfo, nil).Once()
			info, err := client.InfoByHash(ctx, hash)
			require.NoError(t, err)
			require.Equal(t, expectedInfo, info)
			return nil
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	{
		name: "InfoByLabel",
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			label := eth.BlockLabel(eth.Unsafe)
			_, rhdr := randHeader()
			expectedInfo, _ := rhdr.Info(true, false)
			mockRPC.On("InfoByLabel", ctx, label).Return(expectedInfo, nil).Once()
			info, err := client.InfoByLabel(ctx, label)
			require.NoError(t, err)
			require.Equal(t, expectedInfo, info)
			return nil
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	{
		name: "InfoAndTxsByHash",
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			hash := common.Hash{}
			block := &types.Block{}
			blockInfo := eth.BlockToInfo(block)
			txs := types.Transactions{}
			mockRPC.On("InfoAndTxsByHash", ctx, hash).Return(blockInfo, txs, nil).Once()
			_, _, err := client.InfoAndTxsByHash(ctx, hash)
			return err
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	{
		name: "InfoAndTxsByNumber",
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			number := uint64(1234)
			block := &types.Block{}
			blockInfo := eth.BlockToInfo(block)
			txs := types.Transactions{}
			mockRPC.On("InfoAndTxsByNumber", ctx, number).Return(blockInfo, txs, nil).Once()
			_, _, err := client.InfoAndTxsByNumber(ctx, number)
			return err
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	{
		name: "PayloadByHash",
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			hash := common.Hash{}
			payload := &eth.ExecutionPayload{}
			mockRPC.On("PayloadByHash", ctx, hash).Return(payload, nil).Once()
			_, err := client.PayloadByHash(ctx, hash)
			return err
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	{
		name: "PayloadByNumber",
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			number := uint64(1234)
			payload := &eth.ExecutionPayload{}
			mockRPC.On("PayloadByNumber", ctx, number).Return(payload, nil).Once()
			_, err := client.PayloadByNumber(ctx, number)
			return err
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	{
		name: "PayloadByLabel",
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			label := eth.BlockLabel(eth.Unsafe)
			payload := &eth.ExecutionPayload{}
			mockRPC.On("PayloadByLabel", ctx, label).Return(payload, nil).Once()
			_, err := client.PayloadByLabel(ctx, label)
			return err
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
	{
		name: "FetchReceipts",
		testFunc: func(t *testing.T, ctx context.Context, client *PrefetchingEthClient, mockRPC *mockRPC) error {
			blockHash := common.Hash{}
			blockInfo := eth.BlockToInfo(&types.Block{})
			receipts := types.Receipts{}
			mockRPC.On("FetchReceipts", ctx, blockHash).Return(blockInfo, receipts, nil).Once()
			_, _, err := client.FetchReceipts(ctx, blockHash)
			return err
		},
		prefetchingRanges: []uint64{0, 1, 5},
	},
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
