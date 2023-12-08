package sources

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
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
			logger := testlog.Logger(t, log.LvlDebug)
			fakeTip := eth.HeaderBlockInfo(&types.Header{Number: big.NewInt(10_000)})
			mockEthClient.ExpectInfoByLabel(eth.Unsafe, fakeTip, nil)
			client, err := NewPrefetchingEthClient(mockEthClient, logger, prefetchingRange, 30*time.Second)
			require.NoError(t, err)
			defer client.Close()
			client.wg = new(sync.WaitGroup) // Initialize the WaitGroup for testing

			// set up a random block to get from the client
			randomness := rand.New(rand.NewSource(123))
			block, _ := randomRpcBlockAndReceipts(randomness, 2)
			const from = 1_000
			block.rpcHeader.Number = from
			rhdr := block.rpcHeader
			expectedTxs := block.Transactions
			expectedInfo, err := rhdr.Info(true, false)
			require.NoError(t, err)
			mockEthClient.ExpectInfoAndTxsByNumber(from, expectedInfo, expectedTxs, nil)

			// also set up a window of random blocks and receipts to prefetch
			windowEnd := (from + 1 + client.PrefetchingRange)
			for i := uint64(from) + 1; i < windowEnd; i++ {
				// set up different info per iteration
				fillerBlock, fillerReceipts := randomRpcBlockAndReceipts(randomness, 2)
				fillerBlock.rpcHeader.Number = hexutil.Uint64(i)
				fillerInfo, err := fillerBlock.rpcHeader.Info(true, false)
				require.NoError(t, err)
				mockEthClient.ExpectInfoAndTxsByNumber(i, fillerInfo, fillerBlock.Transactions, nil)
				mockEthClient.ExpectFetchReceipts(fillerBlock.Hash, fillerInfo, fillerReceipts, nil)
			}

			info, txs, err := client.InfoAndTxsByNumber(ctx, from)
			require.NoError(t, err)
			require.Equal(t, info, expectedInfo)
			require.Equal(t, txs, types.Transactions(expectedTxs))
			client.wg.Wait() // Wait for all goroutines to complete before asserting expectations
			mockEthClient.AssertExpectations(t)
		})
	}
}

func TestPrefetchingEthClient_updateRange(t *testing.T) {
	const (
		tip  = 10_000
		rnge = 10
	)
	logger := testlog.Logger(t, log.LvlDebug)

	type testCase struct {
		from        uint64
		start       uint64
		end         uint64
		shouldFetch bool
		tip         uint64 // optional tip update
	}
	tests := []struct {
		desc string
		ts   []testCase
	}{
		{
			desc: "3-hist",
			ts: []testCase{
				{
					from:        10,
					start:       11,
					end:         21,
					shouldFetch: true,
				},
				{
					from:        11,
					start:       21,
					end:         22,
					shouldFetch: true,
				},
				{
					from:        15,
					start:       22,
					end:         26,
					shouldFetch: true,
				},
			},
		},

		{
			desc: "equal",
			ts: []testCase{
				{
					from:        10,
					start:       11,
					end:         21,
					shouldFetch: true,
				},
				{
					from:        10,
					start:       21,
					end:         21,
					shouldFetch: false,
				},
			},
		},

		{
			desc: "hist-tip-hist",
			ts: []testCase{ // historical, then tip, then hist
				{
					from:        10,
					start:       11,
					end:         21,
					shouldFetch: true,
				},
				{
					from:        tip,
					start:       0,
					end:         0,
					shouldFetch: false,
					tip:         tip,
				},
				{
					from:        11,
					start:       21,
					end:         22,
					shouldFetch: true,
				},
			},
		},

		{
			desc: "tip-2-hist",
			ts: []testCase{
				{
					from:        tip,
					start:       0,
					end:         0,
					shouldFetch: false,
					tip:         tip,
				},
				{
					from:        10,
					start:       11,
					end:         21,
					shouldFetch: true,
				},
				{
					from:        11,
					start:       21,
					end:         22,
					shouldFetch: true,
				},
			},
		},

		{
			desc: "tip-update-2-hist",
			ts: []testCase{
				{
					from:        tip + 13,
					start:       0,
					end:         0,
					shouldFetch: false,
					tip:         tip + 13,
				},
				{
					from:        10,
					start:       11,
					end:         21,
					shouldFetch: true,
				},
				{
					from:        11,
					start:       21,
					end:         22,
					shouldFetch: true,
				},
			},
		},

		{
			desc: "hist-old-hist",
			ts: []testCase{
				{
					from:        100,
					start:       101,
					end:         111,
					shouldFetch: true,
				},
				{
					from:        10,
					start:       11,
					end:         21,
					shouldFetch: true,
				},
				{
					from:        105,
					start:       106,
					end:         116,
					shouldFetch: true,
				},
			},
		},

		{
			desc: "near-tip",
			ts: []testCase{
				{
					from:        tip - 100,
					start:       tip - 99,
					end:         tip - 89,
					shouldFetch: true,
					tip:         tip,
				},
				{
					from:        tip - 99,
					start:       tip - 89,
					end:         tip - 88,
					shouldFetch: true,
					tip:         tip,
				},
			},
		},

		{
			desc: "very-near-tip",
			ts: []testCase{
				{
					from:        tip - 5,
					start:       tip - 4,
					end:         tip,
					shouldFetch: true,
					tip:         tip,
				},
				{
					from:        tip - 4,
					start:       tip,
					end:         tip,
					shouldFetch: false,
					tip:         tip,
				},
			},
		},
	}

	for _, tts := range tests {
		t.Run(tts.desc, func(t *testing.T) {
			p := &PrefetchingEthClient{
				latestTip:        tip,
				PrefetchingRange: rnge,
				logger:           logger,
			}

			for i, tt := range tts.ts {
				start, end, sf := p.updateRange(tt.from)
				require.Equal(t, tt.start, start, "start mismatch (%d)", i)
				require.Equal(t, tt.end, end, "end mismatch (%d)", i)
				require.Equal(t, tt.shouldFetch, sf, "shouldFetch mismatch (%d)", i)
				if tt.tip != 0 {
					require.Equal(t, tt.tip, p.latestTip, "tip mismatch (%d)", i)
				}
			}
		})
	}
}
