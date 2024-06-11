package attributes

import (
	"context"
	"io"
	"math/big"
	"math/rand" // nosemgrep
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func TestAttributesHandler(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	refA := testutils.RandomBlockRef(rng)

	aL1Info := &testutils.MockBlockInfo{
		InfoParentHash:  refA.ParentHash,
		InfoNum:         refA.Number,
		InfoTime:        refA.Time,
		InfoHash:        refA.Hash,
		InfoBaseFee:     big.NewInt(1),
		InfoBlobBaseFee: big.NewInt(1),
		InfoReceiptRoot: types.EmptyRootHash,
		InfoRoot:        testutils.RandomHash(rng),
		InfoGasUsed:     rng.Uint64(),
	}

	refA0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}
	refA0Alt := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}

	gasLimit := eth.Uint64Quantity(20_000_000)
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{42},
				Overhead:    [32]byte{31: 123},
				Scalar:      [32]byte{0: 0, 31: 42},
				GasLimit:    20_000_000,
			},
		},
		BlockTime:     1,
		SeqWindowSize: 2,
		RegolithTime:  new(uint64),
		CanyonTime:    new(uint64),
		EcotoneTime:   new(uint64),
	}

	a1L1Info, err := derive.L1InfoDepositBytes(cfg, cfg.Genesis.SystemConfig, 1, aL1Info, refA0.Time+cfg.BlockTime)
	require.NoError(t, err)
	parentBeaconBlockRoot := testutils.RandomHash(rng)
	payloadA1 := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:    refA0.Hash,
		FeeRecipient:  common.Address{},
		StateRoot:     eth.Bytes32{},
		ReceiptsRoot:  eth.Bytes32{},
		LogsBloom:     eth.Bytes256{},
		PrevRandao:    eth.Bytes32{},
		BlockNumber:   eth.Uint64Quantity(refA0.Number + 1),
		GasLimit:      gasLimit,
		GasUsed:       0,
		Timestamp:     eth.Uint64Quantity(refA0.Time + cfg.BlockTime),
		ExtraData:     nil,
		BaseFeePerGas: eth.Uint256Quantity(*uint256.NewInt(7)),
		BlockHash:     common.Hash{},
		Transactions:  []eth.Data{a1L1Info},
	}, ParentBeaconBlockRoot: &parentBeaconBlockRoot}
	// fix up the block-hash
	payloadA1.ExecutionPayload.BlockHash, _ = payloadA1.CheckBlockHash()

	attrA1 := &derive.AttributesWithParent{
		Attributes: &eth.PayloadAttributes{
			Timestamp:             payloadA1.ExecutionPayload.Timestamp,
			PrevRandao:            payloadA1.ExecutionPayload.PrevRandao,
			SuggestedFeeRecipient: payloadA1.ExecutionPayload.FeeRecipient,
			Withdrawals:           payloadA1.ExecutionPayload.Withdrawals,
			ParentBeaconBlockRoot: payloadA1.ParentBeaconBlockRoot,
			Transactions:          []eth.Data{a1L1Info},
			NoTxPool:              false,
			GasLimit:              &payloadA1.ExecutionPayload.GasLimit,
		},
		Parent:       refA0,
		IsLastInSpan: true,
	}
	refA1, err := derive.PayloadToBlockRef(cfg, payloadA1.ExecutionPayload)
	require.NoError(t, err)

	payloadA1Alt := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:    refA0.Hash,
		FeeRecipient:  common.Address{0xde, 0xea}, // change of the alternative payload
		StateRoot:     eth.Bytes32{},
		ReceiptsRoot:  eth.Bytes32{},
		LogsBloom:     eth.Bytes256{},
		PrevRandao:    eth.Bytes32{},
		BlockNumber:   eth.Uint64Quantity(refA0.Number + 1),
		GasLimit:      gasLimit,
		GasUsed:       0,
		Timestamp:     eth.Uint64Quantity(refA0.Time + cfg.BlockTime),
		ExtraData:     nil,
		BaseFeePerGas: eth.Uint256Quantity(*uint256.NewInt(7)),
		BlockHash:     common.Hash{},
		Transactions:  []eth.Data{a1L1Info},
	}, ParentBeaconBlockRoot: &parentBeaconBlockRoot}
	// fix up the block-hash
	payloadA1Alt.ExecutionPayload.BlockHash, _ = payloadA1Alt.CheckBlockHash()

	attrA1Alt := &derive.AttributesWithParent{
		Attributes: &eth.PayloadAttributes{
			Timestamp:             payloadA1Alt.ExecutionPayload.Timestamp,
			PrevRandao:            payloadA1Alt.ExecutionPayload.PrevRandao,
			SuggestedFeeRecipient: payloadA1Alt.ExecutionPayload.FeeRecipient,
			Withdrawals:           payloadA1Alt.ExecutionPayload.Withdrawals,
			ParentBeaconBlockRoot: payloadA1Alt.ParentBeaconBlockRoot,
			Transactions:          []eth.Data{a1L1Info},
			NoTxPool:              false,
			GasLimit:              &payloadA1Alt.ExecutionPayload.GasLimit,
		},
		Parent:       refA0,
		IsLastInSpan: true,
	}

	refA1Alt, err := derive.PayloadToBlockRef(cfg, payloadA1Alt.ExecutionPayload)
	require.NoError(t, err)

	refA2 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA1.Number + 1,
		ParentHash:     refA1.Hash,
		Time:           refA1.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}

	a2L1Info, err := derive.L1InfoDepositBytes(cfg, cfg.Genesis.SystemConfig, refA2.SequenceNumber, aL1Info, refA2.Time)
	require.NoError(t, err)
	attrA2 := &derive.AttributesWithParent{
		Attributes: &eth.PayloadAttributes{
			Timestamp:             eth.Uint64Quantity(refA2.Time),
			PrevRandao:            eth.Bytes32{},
			SuggestedFeeRecipient: common.Address{},
			Withdrawals:           nil,
			ParentBeaconBlockRoot: &common.Hash{},
			Transactions:          []eth.Data{a2L1Info},
			NoTxPool:              false,
			GasLimit:              &gasLimit,
		},
		Parent:       refA1,
		IsLastInSpan: true,
	}

	t.Run("drop stale attributes", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelInfo)
		eng := &testutils.MockEngine{}
		ec := derive.NewEngineController(eng, logger, metrics.NoopMetrics, cfg, sync.CLSync)
		ah := NewAttributesHandler(logger, cfg, ec, eng)
		defer eng.AssertExpectations(t)

		ec.SetPendingSafeL2Head(refA1Alt)
		ah.SetAttributes(attrA1)
		require.True(t, ah.HasAttributes())
		require.NoError(t, ah.Proceed(context.Background()), "drop stale attributes")
		require.False(t, ah.HasAttributes())
	})

	t.Run("pending gets reorged", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelInfo)
		eng := &testutils.MockEngine{}
		ec := derive.NewEngineController(eng, logger, metrics.NoopMetrics, cfg, sync.CLSync)
		ah := NewAttributesHandler(logger, cfg, ec, eng)
		defer eng.AssertExpectations(t)

		ec.SetPendingSafeL2Head(refA0Alt)
		ah.SetAttributes(attrA1)
		require.True(t, ah.HasAttributes())
		require.ErrorIs(t, ah.Proceed(context.Background()), derive.ErrReset, "A1 does not fit on A0Alt")
		require.True(t, ah.HasAttributes(), "detected reorg does not clear state, reset is required")
	})

	t.Run("pending older than unsafe", func(t *testing.T) {
		t.Run("consolidation fails", func(t *testing.T) {
			logger := testlog.Logger(t, log.LevelInfo)
			eng := &testutils.MockEngine{}
			ec := derive.NewEngineController(eng, logger, metrics.NoopMetrics, cfg, sync.CLSync)
			ah := NewAttributesHandler(logger, cfg, ec, eng)

			ec.SetUnsafeHead(refA1)
			ec.SetSafeHead(refA0)
			ec.SetFinalizedHead(refA0)
			ec.SetPendingSafeL2Head(refA0)

			defer eng.AssertExpectations(t)

			// Call during consolidation.
			// The payloadA1 is going to get reorged out in favor of attrA1Alt (turns into payloadA1Alt)
			eng.ExpectPayloadByNumber(refA1.Number, payloadA1, nil)

			// attrA1Alt does not match block A1, so will cause force-reorg.
			{
				eng.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
					HeadBlockHash:      payloadA1Alt.ExecutionPayload.ParentHash, // reorg
					SafeBlockHash:      refA0.Hash,
					FinalizedBlockHash: refA0.Hash,
				}, attrA1Alt.Attributes, &eth.ForkchoiceUpdatedResult{
					PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid},
					PayloadID:     &eth.PayloadID{1, 2, 3},
				}, nil) // to build the block
				eng.ExpectGetPayload(eth.PayloadID{1, 2, 3}, payloadA1Alt, nil)
				eng.ExpectNewPayload(payloadA1Alt.ExecutionPayload, payloadA1Alt.ParentBeaconBlockRoot,
					&eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil) // to persist the block
				eng.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
					HeadBlockHash:      payloadA1Alt.ExecutionPayload.BlockHash,
					SafeBlockHash:      payloadA1Alt.ExecutionPayload.BlockHash,
					FinalizedBlockHash: refA0.Hash,
				}, nil, &eth.ForkchoiceUpdatedResult{
					PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid},
					PayloadID:     nil,
				}, nil) // to make it canonical
			}

			ah.SetAttributes(attrA1Alt)

			require.True(t, ah.HasAttributes())
			require.NoError(t, ah.Proceed(context.Background()), "fail consolidation, perform force reorg")
			require.False(t, ah.HasAttributes())

			require.Equal(t, refA1Alt.Hash, payloadA1Alt.ExecutionPayload.BlockHash, "hash")
			t.Log("ref A1: ", refA1.Hash)
			t.Log("ref A0: ", refA0.Hash)
			t.Log("ref alt: ", refA1Alt.Hash)
			require.Equal(t, refA1Alt, ec.UnsafeL2Head(), "unsafe head reorg complete")
			require.Equal(t, refA1Alt, ec.SafeL2Head(), "safe head reorg complete and updated")
		})
		t.Run("consolidation passes", func(t *testing.T) {
			fn := func(t *testing.T, lastInSpan bool) {
				logger := testlog.Logger(t, log.LevelInfo)
				eng := &testutils.MockEngine{}
				ec := derive.NewEngineController(eng, logger, metrics.NoopMetrics, cfg, sync.CLSync)
				ah := NewAttributesHandler(logger, cfg, ec, eng)

				ec.SetUnsafeHead(refA1)
				ec.SetSafeHead(refA0)
				ec.SetFinalizedHead(refA0)
				ec.SetPendingSafeL2Head(refA0)

				defer eng.AssertExpectations(t)

				// Call during consolidation.
				eng.ExpectPayloadByNumber(refA1.Number, payloadA1, nil)

				expectedSafeHash := refA0.Hash
				if lastInSpan { // if last in span, then it becomes safe
					expectedSafeHash = refA1.Hash
				}
				eng.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
					HeadBlockHash:      refA1.Hash,
					SafeBlockHash:      expectedSafeHash,
					FinalizedBlockHash: refA0.Hash,
				}, nil, &eth.ForkchoiceUpdatedResult{
					PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid},
					PayloadID:     nil,
				}, nil)

				attr := &derive.AttributesWithParent{
					Attributes:   attrA1.Attributes, // attributes will match, passing consolidation
					Parent:       attrA1.Parent,
					IsLastInSpan: lastInSpan,
				}
				ah.SetAttributes(attr)

				require.True(t, ah.HasAttributes())
				require.NoError(t, ah.Proceed(context.Background()), "consolidate")
				require.False(t, ah.HasAttributes())
				require.NoError(t, ec.TryUpdateEngine(context.Background()), "update to handle safe bump (lastinspan case)")
				if lastInSpan {
					require.Equal(t, refA1, ec.SafeL2Head(), "last in span becomes safe instantaneously")
				} else {
					require.Equal(t, refA1, ec.PendingSafeL2Head(), "pending as safe")
					require.Equal(t, refA0, ec.SafeL2Head(), "A1 not yet safe")
				}
			}
			t.Run("is last span", func(t *testing.T) {
				fn(t, true)
			})

			t.Run("is not last span", func(t *testing.T) {
				fn(t, false)
			})
		})
	})

	t.Run("pending equals unsafe", func(t *testing.T) {
		// no consolidation to do, just force next attributes on tip of chain

		logger := testlog.Logger(t, log.LevelInfo)
		eng := &testutils.MockEngine{}
		ec := derive.NewEngineController(eng, logger, metrics.NoopMetrics, cfg, sync.CLSync)
		ah := NewAttributesHandler(logger, cfg, ec, eng)

		ec.SetUnsafeHead(refA0)
		ec.SetSafeHead(refA0)
		ec.SetFinalizedHead(refA0)
		ec.SetPendingSafeL2Head(refA0)

		defer eng.AssertExpectations(t)

		// sanity check test setup
		require.True(t, attrA1Alt.IsLastInSpan, "must be last in span for attributes to become safe")

		// process attrA1Alt on top
		{
			eng.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
				HeadBlockHash:      payloadA1Alt.ExecutionPayload.ParentHash, // reorg
				SafeBlockHash:      refA0.Hash,
				FinalizedBlockHash: refA0.Hash,
			}, attrA1Alt.Attributes, &eth.ForkchoiceUpdatedResult{
				PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid},
				PayloadID:     &eth.PayloadID{1, 2, 3},
			}, nil) // to build the block
			eng.ExpectGetPayload(eth.PayloadID{1, 2, 3}, payloadA1Alt, nil)
			eng.ExpectNewPayload(payloadA1Alt.ExecutionPayload, payloadA1Alt.ParentBeaconBlockRoot,
				&eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil) // to persist the block
			eng.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
				HeadBlockHash:      payloadA1Alt.ExecutionPayload.BlockHash,
				SafeBlockHash:      payloadA1Alt.ExecutionPayload.BlockHash, // it becomes safe
				FinalizedBlockHash: refA0.Hash,
			}, nil, &eth.ForkchoiceUpdatedResult{
				PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid},
				PayloadID:     nil,
			}, nil) // to make it canonical
		}

		ah.SetAttributes(attrA1Alt)

		require.True(t, ah.HasAttributes())
		require.NoError(t, ah.Proceed(context.Background()), "insert new block")
		require.False(t, ah.HasAttributes())

		require.Equal(t, refA1Alt, ec.SafeL2Head(), "processing complete")
	})

	t.Run("pending ahead of unsafe", func(t *testing.T) {
		// Legacy test case: if attributes fit on top of the pending safe block as expected,
		// but if the unsafe block is older, then we can recover by updating the unsafe head.

		logger := testlog.Logger(t, log.LevelInfo)
		eng := &testutils.MockEngine{}
		ec := derive.NewEngineController(eng, logger, metrics.NoopMetrics, cfg, sync.CLSync)
		ah := NewAttributesHandler(logger, cfg, ec, eng)

		ec.SetUnsafeHead(refA0)
		ec.SetSafeHead(refA0)
		ec.SetFinalizedHead(refA0)
		ec.SetPendingSafeL2Head(refA1)

		defer eng.AssertExpectations(t)

		ah.SetAttributes(attrA2)

		require.True(t, ah.HasAttributes())
		require.NoError(t, ah.Proceed(context.Background()), "detect unsafe - pending safe inconsistency")
		require.True(t, ah.HasAttributes(), "still need the attributes, after unsafe head is corrected")

		require.Equal(t, refA0, ec.SafeL2Head(), "still same safe head")
		require.Equal(t, refA1, ec.PendingSafeL2Head(), "still same pending safe head")
		require.Equal(t, refA1, ec.UnsafeL2Head(), "updated unsafe head")
	})

	t.Run("no attributes", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelInfo)
		eng := &testutils.MockEngine{}
		ec := derive.NewEngineController(eng, logger, metrics.NoopMetrics, cfg, sync.CLSync)
		ah := NewAttributesHandler(logger, cfg, ec, eng)
		defer eng.AssertExpectations(t)

		require.Equal(t, ah.Proceed(context.Background()), io.EOF, "no attributes to process")
	})

}
