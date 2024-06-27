package clsync

import (
	"context"
	"errors"
	"io"
	"math/big"
	"math/rand" // nosemgrep
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type fakeEngine struct {
	unsafe, safe, finalized eth.L2BlockRef

	err error
}

func (f *fakeEngine) Finalized() eth.L2BlockRef {
	return f.finalized
}

func (f *fakeEngine) UnsafeL2Head() eth.L2BlockRef {
	return f.unsafe
}

func (f *fakeEngine) SafeL2Head() eth.L2BlockRef {
	return f.safe
}

func (f *fakeEngine) InsertUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error {
	if f.err != nil {
		return f.err
	}
	f.unsafe = ref
	return nil
}

var _ Engine = (*fakeEngine)(nil)

func TestCLSync(t *testing.T) {
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
	gasLimit := eth.Uint64Quantity(20_000_000)
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{42},
				Overhead:    [32]byte{123},
				Scalar:      [32]byte{42},
				GasLimit:    20_000_000,
			},
		},
		BlockTime:     1,
		SeqWindowSize: 2,
	}

	refA1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA0.Number + 1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}

	altRefA1 := refA1
	altRefA1.Hash = testutils.RandomHash(rng)

	refA2 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA1.Number + 1,
		ParentHash:     refA1.Hash,
		Time:           refA1.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 2,
	}

	a1L1Info, err := derive.L1InfoDepositBytes(cfg, cfg.Genesis.SystemConfig, refA1.SequenceNumber, aL1Info, refA1.Time)
	require.NoError(t, err)
	payloadA1 := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:    refA1.ParentHash,
		FeeRecipient:  common.Address{},
		StateRoot:     eth.Bytes32{},
		ReceiptsRoot:  eth.Bytes32{},
		LogsBloom:     eth.Bytes256{},
		PrevRandao:    eth.Bytes32{},
		BlockNumber:   eth.Uint64Quantity(refA1.Number),
		GasLimit:      gasLimit,
		GasUsed:       0,
		Timestamp:     eth.Uint64Quantity(refA1.Time),
		ExtraData:     nil,
		BaseFeePerGas: eth.Uint256Quantity(*uint256.NewInt(7)),
		BlockHash:     refA1.Hash,
		Transactions:  []eth.Data{a1L1Info},
	}}
	a2L1Info, err := derive.L1InfoDepositBytes(cfg, cfg.Genesis.SystemConfig, refA2.SequenceNumber, aL1Info, refA2.Time)
	require.NoError(t, err)
	payloadA2 := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:    refA2.ParentHash,
		FeeRecipient:  common.Address{},
		StateRoot:     eth.Bytes32{},
		ReceiptsRoot:  eth.Bytes32{},
		LogsBloom:     eth.Bytes256{},
		PrevRandao:    eth.Bytes32{},
		BlockNumber:   eth.Uint64Quantity(refA2.Number),
		GasLimit:      gasLimit,
		GasUsed:       0,
		Timestamp:     eth.Uint64Quantity(refA2.Time),
		ExtraData:     nil,
		BaseFeePerGas: eth.Uint256Quantity(*uint256.NewInt(7)),
		BlockHash:     refA2.Hash,
		Transactions:  []eth.Data{a2L1Info},
	}}

	metrics := &testutils.TestDerivationMetrics{}

	// When a previously received unsafe block is older than the tip of the chain, we want to drop it.
	t.Run("drop old", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		eng := &fakeEngine{
			unsafe:    refA2,
			safe:      refA0,
			finalized: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, eng)

		cl.AddUnsafePayload(payloadA1)
		require.NoError(t, cl.Proceed(context.Background()))

		require.Nil(t, cl.unsafePayloads.Peek(), "pop because too old")
		require.Equal(t, refA2, eng.unsafe, "keep unsafe head")
	})

	// When we already have the exact payload as tip, then no need to process it
	t.Run("drop equal", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		eng := &fakeEngine{
			unsafe:    refA1,
			safe:      refA0,
			finalized: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, eng)

		cl.AddUnsafePayload(payloadA1)
		require.NoError(t, cl.Proceed(context.Background()))

		require.Nil(t, cl.unsafePayloads.Peek(), "pop because seen")
		require.Equal(t, refA1, eng.unsafe, "keep unsafe head")
	})

	// When we have a different payload, at the same height, then we want to keep it.
	// The unsafe chain consensus preserves the first-seen payload.
	t.Run("ignore conflict", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		eng := &fakeEngine{
			unsafe:    altRefA1,
			safe:      refA0,
			finalized: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, eng)

		cl.AddUnsafePayload(payloadA1)
		require.NoError(t, cl.Proceed(context.Background()))

		require.Nil(t, cl.unsafePayloads.Peek(), "pop because alternative")
		require.Equal(t, altRefA1, eng.unsafe, "keep unsafe head")
	})

	t.Run("ignore unsafe reorg", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		eng := &fakeEngine{
			unsafe:    altRefA1,
			safe:      refA0,
			finalized: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, eng)

		cl.AddUnsafePayload(payloadA2)
		require.ErrorIs(t, cl.Proceed(context.Background()), io.EOF, "payload2 does not fit onto alt1, thus retrieve next input from L1")

		require.Nil(t, cl.unsafePayloads.Peek(), "pop because not applicable")
		require.Equal(t, altRefA1, eng.unsafe, "keep unsafe head")
	})

	t.Run("success", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		eng := &fakeEngine{
			unsafe:    refA0,
			safe:      refA0,
			finalized: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, eng)

		require.ErrorIs(t, cl.Proceed(context.Background()), io.EOF, "nothing to process yet")
		require.Nil(t, cl.unsafePayloads.Peek(), "no payloads yet")

		cl.AddUnsafePayload(payloadA1)
		lowest := cl.LowestQueuedUnsafeBlock()
		require.Equal(t, refA1, lowest, "expecting A1 next")
		require.NoError(t, cl.Proceed(context.Background()))
		require.Nil(t, cl.unsafePayloads.Peek(), "pop because applied")
		require.Equal(t, refA1, eng.unsafe, "new unsafe head")

		cl.AddUnsafePayload(payloadA2)
		lowest = cl.LowestQueuedUnsafeBlock()
		require.Equal(t, refA2, lowest, "expecting A2 next")
		require.NoError(t, cl.Proceed(context.Background()))
		require.Nil(t, cl.unsafePayloads.Peek(), "pop because applied")
		require.Equal(t, refA2, eng.unsafe, "new unsafe head")
	})

	t.Run("double buffer", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		eng := &fakeEngine{
			unsafe:    refA0,
			safe:      refA0,
			finalized: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, eng)

		cl.AddUnsafePayload(payloadA1)
		cl.AddUnsafePayload(payloadA2)

		lowest := cl.LowestQueuedUnsafeBlock()
		require.Equal(t, refA1, lowest, "expecting A1 next")

		require.NoError(t, cl.Proceed(context.Background()))
		require.NotNil(t, cl.unsafePayloads.Peek(), "next is ready")
		require.Equal(t, refA1, eng.unsafe, "new unsafe head")
		require.NoError(t, cl.Proceed(context.Background()))
		require.Nil(t, cl.unsafePayloads.Peek(), "done")
		require.Equal(t, refA2, eng.unsafe, "new unsafe head")
	})

	t.Run("temporary error", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		eng := &fakeEngine{
			unsafe:    refA0,
			safe:      refA0,
			finalized: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, eng)

		testErr := derive.NewTemporaryError(errors.New("test error"))
		eng.err = testErr
		cl.AddUnsafePayload(payloadA1)
		require.ErrorIs(t, cl.Proceed(context.Background()), testErr)
		require.Equal(t, refA0, eng.unsafe, "old unsafe head after error")
		require.NotNil(t, cl.unsafePayloads.Peek(), "no pop because temporary error")

		eng.err = nil
		require.NoError(t, cl.Proceed(context.Background()))
		require.Equal(t, refA1, eng.unsafe, "new unsafe head after resolved error")
		require.Nil(t, cl.unsafePayloads.Peek(), "pop because valid")
	})

	t.Run("invalid payload error", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		eng := &fakeEngine{
			unsafe:    refA0,
			safe:      refA0,
			finalized: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, eng)

		testErr := errors.New("test error")
		eng.err = testErr
		cl.AddUnsafePayload(payloadA1)
		require.ErrorIs(t, cl.Proceed(context.Background()), testErr)
		require.Equal(t, refA0, eng.unsafe, "old unsafe head after error")
		require.Nil(t, cl.unsafePayloads.Peek(), "pop because invalid")
	})
}
