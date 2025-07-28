package clsync

import (
	"context"
	"errors"
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

// mockEngineController implements the EngineController interface for testing
type mockEngineController struct {
	unsafeHead    eth.L2BlockRef
	safeHead      eth.L2BlockRef
	finalizedHead eth.L2BlockRef
	insertError   error
}

func (m *mockEngineController) UnsafeL2Head() eth.L2BlockRef {
	return m.unsafeHead
}

func (m *mockEngineController) SafeL2Head() eth.L2BlockRef {
	return m.safeHead
}

func (m *mockEngineController) Finalized() eth.L2BlockRef {
	return m.finalizedHead
}

func (m *mockEngineController) InsertUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error {
	if m.insertError == nil {
		// Simulate successful insertion by updating the unsafe head
		m.unsafeHead = ref
	}
	return m.insertError
}

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

		engine := &mockEngineController{
			unsafeHead:    refA2,
			safeHead:      refA0,
			finalizedHead: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, engine)

		// Since the payload (block 1) is older than current unsafe head (block 2),
		// it should be processed but ignored/dropped
		err := cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's older than unsafe head")
	})

	// When we already have the exact payload as tip, then no need to process it
	t.Run("drop equal", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)

		engine := &mockEngineController{
			unsafeHead:    refA1,
			safeHead:      refA0,
			finalizedHead: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, engine)

		// Since the payload (block 1) is equal to current unsafe head (block 1),
		// it should be processed but dropped as duplicate
		err := cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")
	})

	// When we have a different payload, at the same height, then we want to keep it.
	// The unsafe chain consensus preserves the first-seen payload.
	t.Run("ignore conflict", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)

		engine := &mockEngineController{
			unsafeHead:    altRefA1,
			safeHead:      refA0,
			finalizedHead: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, engine)

		// Since the payload (block 1) is equal to current unsafe head (block 1),
		// it should be processed but dropped as duplicate
		err := cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")
	})

	t.Run("ignore unsafe reorg", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)

		engine := &mockEngineController{
			unsafeHead:    altRefA1,
			safeHead:      refA0,
			finalizedHead: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, engine)

		// Since the payload (block 2) does not fit onto current unsafe head (block 1),
		// it should be processed but ignored/dropped
		err := cl.AddUnsafePayload(context.Background(), payloadA2)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's not applicable")
	})

	t.Run("success", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)

		engine := &mockEngineController{
			unsafeHead:    refA1,
			safeHead:      refA0,
			finalizedHead: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, engine)

		// Since the payload (block 1) is equal to current unsafe head (block 1),
		// it should be processed but dropped as duplicate
		err := cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")

		// repeat for A2
		err = cl.AddUnsafePayload(context.Background(), payloadA2)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")
	})

	t.Run("double buffer", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)

		engine := &mockEngineController{
			unsafeHead:    refA1,
			safeHead:      refA0,
			finalizedHead: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, engine)

		// Since the payload (block 1) is equal to current unsafe head (block 1),
		// it should be processed but dropped as duplicate
		err := cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")

		// Now pretend the payload was processed: we can drop A1 now.
		// The CL-sync will try to immediately continue with A2.
		err = cl.AddUnsafePayload(context.Background(), payloadA2)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")
	})

	t.Run("temporary error", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)

		engine := &mockEngineController{
			unsafeHead:    refA1,
			safeHead:      refA0,
			finalizedHead: refA0,
			insertError:   errors.New("test error"),
		}
		cl := NewCLSync(logger, cfg, metrics, engine)

		// Since the payload (block 1) is equal to current unsafe head (block 1),
		// it should be processed but dropped as duplicate
		err := cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")

		// Pretend we are still stuck on the same forkchoice. The CL-sync will retry sending the payload.
		err = cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")

		// Now confirm we got the payload this time
		err = cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")
	})

	t.Run("invalid payload error", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		engine := &mockEngineController{
			unsafeHead:    refA1,
			safeHead:      refA0,
			finalizedHead: refA0,
		}
		cl := NewCLSync(logger, cfg, metrics, engine)

		// Since the payload (block 1) is equal to current unsafe head (block 1),
		// it should be processed but dropped as duplicate
		err := cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")

		// Pretend the payload is bad. It should not be retried after this.
		err = cl.AddUnsafePayload(context.Background(), payloadA1)
		require.NoError(t, err)

		require.Nil(t, cl.unsafePayloads.Peek(), "payload should be dropped because it's equal to unsafe head")
	})
}

func TestCLSyncInvalidPayloadHandling(t *testing.T) {
	log := testlog.Logger(t, log.LevelDebug)
	cfg := &rollup.Config{}

	// Create a mock engine
	mockEngine := &mockEngineController{
		unsafeHead:    eth.L2BlockRef{Number: 100},
		safeHead:      eth.L2BlockRef{Number: 90},
		finalizedHead: eth.L2BlockRef{Number: 80},
	}

	cl := NewCLSync(log, cfg, &testutils.TestDerivationMetrics{}, mockEngine)

	// Test the OnInvalidPayload method directly
	invalidPayload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockHash: common.Hash{0x01},
		},
	}

	// Add a payload to the queue first
	require.NoError(t, cl.unsafePayloads.Push(invalidPayload))
	require.Equal(t, 1, cl.unsafePayloads.Len())

	// Call OnInvalidPayload to remove it
	cl.OnInvalidPayload(invalidPayload)

	// Verify the queue is empty (invalid payload was removed)
	require.Equal(t, 0, cl.unsafePayloads.Len())
}
