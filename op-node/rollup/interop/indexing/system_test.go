package indexing

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func TestManagedMode_findLatestValidLocalUnsafe(t *testing.T) {
	tests := []struct {
		name           string
		l2Unsafe       uint64   // starting point (trusted valid)
		latestUnsafe   uint64   // current chain tip
		validBlocks    []uint64 // blocks with valid L1 origins
		expectedResult uint64
		expectedError  string
	}{
		{
			name:           "target_equals_latest",
			l2Unsafe:       100,
			latestUnsafe:   100,
			validBlocks:    []uint64{100},
			expectedResult: 100,
		},
		{
			name:           "all_blocks_valid",
			l2Unsafe:       100,
			latestUnsafe:   105,
			validBlocks:    []uint64{100, 101, 102, 103, 104, 105},
			expectedResult: 105,
		},
		{
			name:           "all_blocks_invalid",
			l2Unsafe:       100,
			latestUnsafe:   105,
			validBlocks:    []uint64{100}, // only l2Unsafe is valid
			expectedResult: 100,
		},
		{
			name:           "mixed_validity_case1",
			l2Unsafe:       100,
			latestUnsafe:   105,
			validBlocks:    []uint64{100, 101, 102}, // 103-105 invalid
			expectedResult: 102,
		},
		{
			name:           "single_block_ahead_valid",
			l2Unsafe:       100,
			latestUnsafe:   101,
			validBlocks:    []uint64{100, 101},
			expectedResult: 101,
		},
		{
			name:           "single_block_ahead_invalid",
			l2Unsafe:       100,
			latestUnsafe:   101,
			validBlocks:    []uint64{100}, // 101 invalid
			expectedResult: 100,
		},
		{
			name:           "target_not_at_100",
			l2Unsafe:       95,
			latestUnsafe:   100,
			validBlocks:    []uint64{95, 96, 97}, // 98-100 invalid
			expectedResult: 97,
		},
		{
			name:           "target_is_invalid",
			l2Unsafe:       100,
			latestUnsafe:   100,
			validBlocks:    []uint64{96, 97, 98, 99}, // 96-99 valid
			expectedResult: 99,
		},
		{
			name:           "target_is_larger_than_latest",
			l2Unsafe:       101,
			latestUnsafe:   100,
			validBlocks:    []uint64{96, 97, 98, 99}, // 96-99 valid
			expectedResult: 99,
		},
		{
			name:           "walkback_after_binary_search",
			l2Unsafe:       95,
			latestUnsafe:   105,
			validBlocks:    []uint64{92, 93}, // 92-93 valid
			expectedResult: 93,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := testlog.Logger(t, log.LevelDebug)

			mockL1 := &testutils.MockL1Source{}
			mockL2 := &testutils.MockL2Client{}

			var nilErr error

			// Setup l2Unsafe block
			l2UnsafeHash := common.HexToHash(fmt.Sprintf("0x%x", tt.l2Unsafe))
			l2UnsafeL1Origin := eth.BlockID{Hash: common.HexToHash("0xa"), Number: 10}
			l2UnsafeBlockRef := createL2BlockRef(tt.l2Unsafe, l2UnsafeHash.Hex(), l2UnsafeL1Origin)

			mockL2.ExpectL2BlockRefByHash(l2UnsafeHash, l2UnsafeBlockRef, nilErr)

			// Setup latest unsafe block
			latestHash := common.HexToHash(fmt.Sprintf("0x%x", tt.latestUnsafe))
			latestL1Origin := eth.BlockID{Hash: common.HexToHash("0xf"), Number: 10 + tt.latestUnsafe - tt.l2Unsafe}
			latestBlockRef := createL2BlockRef(tt.latestUnsafe, latestHash.Hex(), latestL1Origin)

			mockL2.ExpectL2BlockRefByLabel(eth.Unsafe, latestBlockRef, nilErr)

			// Setup blocks for binary search
			validBlocksMap := make(map[uint64]bool)
			for _, block := range tt.validBlocks {
				validBlocksMap[block] = true
			}

			// Setup specific expectations for each possible block
			for blockNum := tt.l2Unsafe - 10; blockNum <= tt.latestUnsafe; blockNum++ {
				l1OriginNum := 10 + blockNum - tt.l2Unsafe
				l1OriginHash := fmt.Sprintf("0x%x", l1OriginNum)
				l1Origin := eth.BlockID{Hash: common.HexToHash(l1OriginHash), Number: l1OriginNum}
				l2Block := createL2BlockRef(blockNum, fmt.Sprintf("0x%x", blockNum), l1Origin)

				logger.Info("Setting up block", "l2Block", l2Block, "l1Origin", l1Origin)

				mockL2.On("L2BlockRefByNumber", blockNum).Return(l2Block, &nilErr).Maybe()

				if validBlocksMap[blockNum] {
					// Valid: return matching hash
					mockL1.On("L1BlockRefByNumber", l1OriginNum).
						Return(createL1BlockRef(l1OriginNum, l1OriginHash), nilErr).Maybe()
				} else {
					// Invalid: return different hash (reorg)
					mockL1.On("L1BlockRefByNumber", l1OriginNum).
						Return(createL1BlockRef(l1OriginNum, fmt.Sprintf("0x%x", l1OriginNum+1000)), nilErr).Maybe()
				}
			}

			managedMode := &IndexingMode{
				log: logger,
				l1:  mockL1,
				l2:  mockL2,
			}

			result, err := managedMode.findLatestValidLocalUnsafe(ctx, eth.BlockID{
				Hash:   l2UnsafeHash,
				Number: tt.l2Unsafe,
			})

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result.Number)
			}
		})
	}
}

// Helper functions to create test data
func createL1BlockRef(number uint64, hash string) eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       common.HexToHash(hash),
		Number:     number,
		ParentHash: common.HexToHash("0x0"),
		Time:       1000000 + number*12, // 12 second block time
	}
}

func createL2BlockRef(number uint64, hash string, l1Origin eth.BlockID) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:           common.HexToHash(hash),
		Number:         number,
		ParentHash:     common.HexToHash("0x0"),
		Time:           1000000 + number*2, // 2 second block time
		L1Origin:       l1Origin,
		SequenceNumber: 0,
	}
}
