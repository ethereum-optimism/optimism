package engine_controller

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestEngineController_RewindToTimestamp(t *testing.T) {
	t.Parallel()

	// Helper to create a test payload for a given block
	createPayload := func(blockNum uint64, parentHash common.Hash, blockTime uint64) *eth.ExecutionPayloadEnvelope {
		blockHash := common.Hash{byte(blockNum)}
		return &eth.ExecutionPayloadEnvelope{
			ExecutionPayload: &eth.ExecutionPayload{
				ParentHash:   parentHash,
				BlockNumber:  eth.Uint64Quantity(blockNum),
				Timestamp:    eth.Uint64Quantity(blockTime),
				BlockHash:    blockHash,
				FeeRecipient: common.Address{0x01},
			},
		}
	}

	// Helper to create rollup config
	createRollupConfig := func() *rollup.Config {
		return &rollup.Config{
			Genesis:   rollup.Genesis{L2: eth.BlockID{Number: 0}, L2Time: 1000},
			BlockTime: 2,
			L2ChainID: big.NewInt(420),
		}
	}

	t.Run("successful rewind", func(t *testing.T) {
		t.Parallel()

		// Setup: chain is at block 10, we want to rewind to block 5
		// Block 5 is at timestamp 1000 + 5*2 = 1010
		targetTimestamp := uint64(1010)
		targetBlockNum := uint64(5)
		parentHash := common.Hash{0x04} // parent of block 5

		targetRef := eth.L2BlockRef{
			Number:     targetBlockNum,
			Hash:       common.Hash{byte(targetBlockNum)},
			ParentHash: parentHash,
			Time:       targetTimestamp,
		}

		l2 := &mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetBlockNum: targetRef,
			},
			// Initial state before rewind
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Safe:      {Number: 10, Hash: common.Hash{0x0a}},
				eth.Finalized: {Number: 8, Hash: common.Hash{0x08}},
			},
			// State after FCU completes - verification reads these values
			refsByLabelAfterFCU: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Unsafe:    targetRef,
				eth.Safe:      targetRef, // clamped to target (min of 10 and 5)
				eth.Finalized: targetRef, // clamped to target (min of 8 and 5)
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetBlockNum: createPayload(targetBlockNum, parentHash, targetTimestamp),
			},
		}

		ec := &simpleEngineController{l2: l2, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), targetTimestamp)
		require.NoError(t, err)

		// Verify NewPayload was called (for synthetic block)
		require.Equal(t, 1, l2.newPayloadCalls, "NewPayload should be called once for synthetic block")
		require.NotNil(t, l2.lastNewPayload)
		// The synthetic payload should have modified fee recipient
		require.Equal(t, common.MaxAddress, l2.lastNewPayload.FeeRecipient, "Synthetic payload should have modified fee recipient")

		// Verify ForkchoiceUpdate was called twice (once for synthetic, once for target)
		require.Equal(t, 2, l2.fcuCalls, "ForkchoiceUpdate should be called twice")
	})

	t.Run("returns error when engine client is nil", func(t *testing.T) {
		t.Parallel()

		ec := &simpleEngineController{l2: nil, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), 1010)
		require.ErrorIs(t, err, ErrNoEngineClient)
	})

	t.Run("returns error when rollup config is nil", func(t *testing.T) {
		t.Parallel()

		l2 := &mockL2{}
		ec := &simpleEngineController{l2: l2, rollup: nil, log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), 1010)
		require.ErrorIs(t, err, ErrRewindTargetBlockNotFound)
	})

	t.Run("returns error when target block not found", func(t *testing.T) {
		t.Parallel()

		l2 := &mockL2{
			refErr: ethereum.NotFound,
		}

		ec := &simpleEngineController{l2: l2, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), 1010)
		require.ErrorIs(t, err, ErrRewindTargetBlockNotFound)
	})

	t.Run("returns error when NewPayload fails", func(t *testing.T) {
		t.Parallel()

		targetTimestamp := uint64(1010)
		targetBlockNum := uint64(5)
		parentHash := common.Hash{0x04}

		targetRef := eth.L2BlockRef{
			Number:     targetBlockNum,
			Hash:       common.Hash{byte(targetBlockNum)},
			ParentHash: parentHash,
			Time:       targetTimestamp,
		}

		l2 := &mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetBlockNum: targetRef,
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetBlockNum: createPayload(targetBlockNum, parentHash, targetTimestamp),
			},
			newPayloadErr: errors.New("engine unavailable"),
		}

		ec := &simpleEngineController{l2: l2, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), targetTimestamp)
		require.ErrorIs(t, err, ErrRewindInsertSyntheticFailed)
	})

	t.Run("returns error when NewPayload rejects synthetic block", func(t *testing.T) {
		t.Parallel()

		targetTimestamp := uint64(1010)
		targetBlockNum := uint64(5)
		parentHash := common.Hash{0x04}

		targetRef := eth.L2BlockRef{
			Number:     targetBlockNum,
			Hash:       common.Hash{byte(targetBlockNum)},
			ParentHash: parentHash,
			Time:       targetTimestamp,
		}

		l2 := &mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetBlockNum: targetRef,
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetBlockNum: createPayload(targetBlockNum, parentHash, targetTimestamp),
			},
			newPayloadStatus: &eth.PayloadStatusV1{Status: eth.ExecutionInvalid},
		}

		ec := &simpleEngineController{l2: l2, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), targetTimestamp)
		require.ErrorIs(t, err, ErrRewindSyntheticPayloadRejected)
	})

	t.Run("returns error when FCU to synthetic block fails", func(t *testing.T) {
		t.Parallel()

		targetTimestamp := uint64(1010)
		targetBlockNum := uint64(5)
		parentHash := common.Hash{0x04}

		targetRef := eth.L2BlockRef{
			Number:     targetBlockNum,
			Hash:       common.Hash{byte(targetBlockNum)},
			ParentHash: parentHash,
			Time:       targetTimestamp,
		}

		l2 := &mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetBlockNum: targetRef,
			},
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Safe:      {Number: 10, Hash: common.Hash{0x0a}},
				eth.Finalized: {Number: 8, Hash: common.Hash{0x08}},
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetBlockNum: createPayload(targetBlockNum, parentHash, targetTimestamp),
			},
			fcuErr: errors.New("FCU failed"),
		}

		ec := &simpleEngineController{l2: l2, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), targetTimestamp)
		require.ErrorIs(t, err, ErrRewindFCUSyntheticFailed)
	})

	t.Run("returns error when FCU is rejected by engine", func(t *testing.T) {
		t.Parallel()

		targetTimestamp := uint64(1010)
		targetBlockNum := uint64(5)
		parentHash := common.Hash{0x04}

		targetRef := eth.L2BlockRef{
			Number:     targetBlockNum,
			Hash:       common.Hash{byte(targetBlockNum)},
			ParentHash: parentHash,
			Time:       targetTimestamp,
		}

		l2 := &mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetBlockNum: targetRef,
			},
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Safe:      {Number: 10, Hash: common.Hash{0x0a}},
				eth.Finalized: {Number: 8, Hash: common.Hash{0x08}},
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetBlockNum: createPayload(targetBlockNum, parentHash, targetTimestamp),
			},
			fcuResult: &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}},
		}

		ec := &simpleEngineController{l2: l2, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), targetTimestamp)
		require.ErrorIs(t, err, ErrRewindFCURejected)
	})

	t.Run("returns error when unsafe block hash mismatches after rewind", func(t *testing.T) {
		t.Parallel()

		targetTimestamp := uint64(1010)
		targetBlockNum := uint64(5)
		parentHash := common.Hash{0x04}

		targetRef := eth.L2BlockRef{
			Number:     targetBlockNum,
			Hash:       common.Hash{byte(targetBlockNum)},
			ParentHash: parentHash,
			Time:       targetTimestamp,
		}

		l2 := &mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetBlockNum: targetRef,
			},
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Safe:      {Number: 10, Hash: common.Hash{0x0a}},
				eth.Finalized: {Number: 8, Hash: common.Hash{0x08}},
			},
			// After FCU, unsafe has correct number but wrong hash
			refsByLabelAfterFCU: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Unsafe:    {Number: targetBlockNum, Hash: common.Hash{0xff}}, // wrong hash
				eth.Safe:      targetRef,
				eth.Finalized: targetRef,
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetBlockNum: createPayload(targetBlockNum, parentHash, targetTimestamp),
			},
		}

		ec := &simpleEngineController{l2: l2, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), targetTimestamp)
		require.ErrorIs(t, err, ErrRewindVerificationFailed)
		require.ErrorContains(t, err, "unexpected unsafe block hash")
	})

	t.Run("returns error when safe block hash mismatches after rewind", func(t *testing.T) {
		t.Parallel()

		targetTimestamp := uint64(1010)
		targetBlockNum := uint64(5)
		parentHash := common.Hash{0x04}

		targetRef := eth.L2BlockRef{
			Number:     targetBlockNum,
			Hash:       common.Hash{byte(targetBlockNum)},
			ParentHash: parentHash,
			Time:       targetTimestamp,
		}

		l2 := &mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetBlockNum: targetRef,
			},
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Safe:      {Number: 10, Hash: common.Hash{0x0a}},
				eth.Finalized: {Number: 8, Hash: common.Hash{0x08}},
			},
			// After FCU, safe has correct number but wrong hash
			refsByLabelAfterFCU: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Unsafe:    targetRef,
				eth.Safe:      {Number: targetBlockNum, Hash: common.Hash{0xff}}, // wrong hash
				eth.Finalized: targetRef,
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetBlockNum: createPayload(targetBlockNum, parentHash, targetTimestamp),
			},
		}

		ec := &simpleEngineController{l2: l2, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), targetTimestamp)
		require.ErrorIs(t, err, ErrRewindVerificationFailed)
		require.ErrorContains(t, err, "unexpected safe block hash")
	})

	t.Run("returns error when finalized block hash mismatches after rewind", func(t *testing.T) {
		t.Parallel()

		targetTimestamp := uint64(1010)
		targetBlockNum := uint64(5)
		parentHash := common.Hash{0x04}

		targetRef := eth.L2BlockRef{
			Number:     targetBlockNum,
			Hash:       common.Hash{byte(targetBlockNum)},
			ParentHash: parentHash,
			Time:       targetTimestamp,
		}

		l2 := &mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetBlockNum: targetRef,
			},
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Safe:      {Number: 10, Hash: common.Hash{0x0a}},
				eth.Finalized: {Number: 8, Hash: common.Hash{0x08}},
			},
			// After FCU, finalized has correct number but wrong hash
			refsByLabelAfterFCU: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Unsafe:    targetRef,
				eth.Safe:      targetRef,
				eth.Finalized: {Number: targetBlockNum, Hash: common.Hash{0xff}}, // wrong hash
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetBlockNum: createPayload(targetBlockNum, parentHash, targetTimestamp),
			},
		}

		ec := &simpleEngineController{l2: l2, rollup: createRollupConfig(), log: gethlog.New()}

		err := ec.RewindToTimestamp(context.Background(), targetTimestamp)
		require.ErrorIs(t, err, ErrRewindVerificationFailed)
		require.ErrorContains(t, err, "unexpected finalized block hash")
	})

	t.Run("returns error when timestamp to block conversion fails", func(t *testing.T) {
		t.Parallel()

		// Use a rollup config where the timestamp would result in a negative block number
		// or any other error from blockNumberAtTimestamp
		rcfg := &rollup.Config{
			Genesis:   rollup.Genesis{L2: eth.BlockID{Number: 0}, L2Time: 2000}, // genesis at time 2000
			BlockTime: 2,
			L2ChainID: big.NewInt(420),
		}

		l2 := &mockL2{}

		ec := &simpleEngineController{l2: l2, rollup: rcfg, log: gethlog.New()}

		// Request timestamp before genesis (1000 < 2000), which should fail
		err := ec.RewindToTimestamp(context.Background(), 1000)
		require.ErrorIs(t, err, ErrRewindTimestampToBlockConversion)
	})
}
