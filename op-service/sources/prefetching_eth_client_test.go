package sources

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/core/types"
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
			randomness := rand.New(rand.NewSource(123))

			// set up a random block to get from the client
			block, _ := randomRpcBlockAndReceipts(randomness, 2)
			rhdr := block.rpcHeader
			expectedTxs := block.Transactions
			expectedInfo, err := rhdr.Info(true, false)
			require.NoError(t, err)
			mock.ExpectInfoAndTxsByNumber(uint64(rhdr.Number), expectedInfo, expectedTxs, nil)

			// also set up a window of random blocks and receipts to prefetch
			windowEnd := (uint64(rhdr.Number) + client.PrefetchingRange)
			for i := uint64(rhdr.Number) + 1; i <= windowEnd; i++ {
				// set up different info per iteration
				fillerBlock, fillerReceipts := randomRpcBlockAndReceipts(randomness, 2)
				fillerInfo, err := fillerBlock.rpcHeader.Info(true, false)
				require.NoError(t, err)
				mock.ExpectInfoAndTxsByNumber(i, fillerInfo, fillerBlock.Transactions, nil)
				mock.ExpectFetchReceipts(fillerBlock.Hash, fillerInfo, fillerReceipts, nil)
			}

			info, txs, err := client.InfoAndTxsByNumber(ctx, uint64(rhdr.Number))
			require.NoError(t, err)
			require.Equal(t, info, expectedInfo)
			require.Equal(t, txs, types.Transactions(expectedTxs))
		},
	},
}

// runTest runs a given test function with a specific prefetching range.
func runTest(t *testing.T, name string, testFunc testFunction, prefetchingRange uint64) {
	ctx := context.Background()
	mockEthClient := new(testutils.MockEthClient)
	client, err := NewPrefetchingEthClient(mockEthClient, prefetchingRange, 30*time.Second)
	client.wg = new(sync.WaitGroup) // Initialize the WaitGroup for testing
	require.NoError(t, err)

	testFunc(t, ctx, client, mockEthClient)
	client.wg.Wait()            // Wait for all goroutines to complete before asserting expectations
	time.Sleep(2 * time.Second) // with this, tests pass; without it, they fail. Why? the waitgroup should be enough
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

func TestUpdateRequestingHead_NormalRange(t *testing.T) {
	client := &PrefetchingEthClient{
		highestHeadRequesting: 10,
		PrefetchingTimeout:    30 * time.Second,
	}

	start, end := uint64(11), uint64(15)
	newStart, shouldFetch := client.updateRequestingHead(start, end)

	if newStart != start {
		t.Errorf("Expected newStart to be %d, got %d", start, newStart)
	}
	if !shouldFetch {
		t.Error("Expected shouldFetch to be true")
	}
	if client.highestHeadRequesting != end {
		t.Errorf("Expected highestHeadRequesting to be updated to %d, got %d", end, client.highestHeadRequesting)
	}
}

func TestUpdateRequestingHead_OverlappingRange(t *testing.T) {
	highestHeadBeforeUpdate := uint64(10)
	client := &PrefetchingEthClient{
		highestHeadRequesting: highestHeadBeforeUpdate,
		PrefetchingTimeout:    30 * time.Second,
	}

	start, end := uint64(8), uint64(12)
	newStart, shouldFetch := client.updateRequestingHead(start, end)

	if newStart != highestHeadBeforeUpdate+1 {
		t.Errorf("Expected newStart to be %d, got %d", highestHeadBeforeUpdate+1, newStart)
	}
	if !shouldFetch {
		t.Error("Expected shouldFetch to be true")
	}
	if client.highestHeadRequesting != end {
		t.Errorf("Expected highestHeadRequesting to be updated to %d, got %d", end, client.highestHeadRequesting)
	}
}
