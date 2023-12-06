package sources

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestPrefetchingEthClient runs all test cases for each prefetching range.
func TestPrefetchingEthClient(t *testing.T) {
	prefetchingRanges := []uint64{0, 1, 5}
	for _, prefetchingRange := range prefetchingRanges {
		testName := fmt.Sprintf("range-%d", prefetchingRange)
		t.Run(testName, func(t *testing.T) {
			ctx := context.Background()
			mockEthClient := new(testutils.MockEthClient)
			client, err := NewPrefetchingEthClient(mockEthClient, prefetchingRange, 30*time.Second)
			require.NoError(t, err)
			defer client.Close()
			client.wg = new(sync.WaitGroup) // Initialize the WaitGroup for testing

			// set up a random block to get from the client
			randomness := rand.New(rand.NewSource(123))
			block, _ := randomRpcBlockAndReceipts(randomness, 2)
			rhdr := block.rpcHeader
			expectedTxs := block.Transactions
			expectedInfo, err := rhdr.Info(true, false)
			require.NoError(t, err)
			mockEthClient.ExpectInfoAndTxsByNumber(uint64(rhdr.Number), expectedInfo, expectedTxs, nil)

			// also set up a window of random blocks and receipts to prefetch
			windowEnd := (uint64(rhdr.Number) + client.PrefetchingRange)
			for i := uint64(rhdr.Number) + 1; i <= windowEnd; i++ {
				// set up different info per iteration
				fillerBlock, fillerReceipts := randomRpcBlockAndReceipts(randomness, 2)
				fillerBlock.rpcHeader.Number = hexutil.Uint64(i)
				fillerInfo, err := fillerBlock.rpcHeader.Info(true, false)
				require.NoError(t, err)
				mockEthClient.ExpectInfoAndTxsByNumber(i, fillerInfo, fillerBlock.Transactions, nil)
				mockEthClient.ExpectFetchReceipts(fillerBlock.Hash, fillerInfo, fillerReceipts, nil)
			}

			info, txs, err := client.InfoAndTxsByNumber(ctx, uint64(rhdr.Number))
			require.NoError(t, err)
			require.Equal(t, info, expectedInfo)
			require.Equal(t, txs, types.Transactions(expectedTxs))
			client.wg.Wait() // Wait for all goroutines to complete before asserting expectations
			mockEthClient.AssertExpectations(t)
		})
	}
}

func TestUpdateRequestingHead_NormalRange(t *testing.T) {
	client := &PrefetchingEthClient{
		highestHeadRequesting: 10,
		PrefetchingTimeout:    30 * time.Second,
	}

	start, end := uint64(11), uint64(15)
	newStart, shouldFetch := client.updateRequestingHead(start, end)

	require.Equal(t, newStart, start)
	require.True(t, shouldFetch)
	require.Equal(t, client.highestHeadRequesting, end)
}

func TestUpdateRequestingHead_OverlappingRange(t *testing.T) {
	highestHeadBeforeUpdate := uint64(10)
	client := &PrefetchingEthClient{
		highestHeadRequesting: highestHeadBeforeUpdate,
		PrefetchingTimeout:    30 * time.Second,
	}

	start, end := uint64(8), uint64(12)
	newStart, shouldFetch := client.updateRequestingHead(start, end)

	require.Equal(t, newStart, highestHeadBeforeUpdate+1)
	require.True(t, shouldFetch)
	require.Equal(t, client.highestHeadRequesting, end)
}
