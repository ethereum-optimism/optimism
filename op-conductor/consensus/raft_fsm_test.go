package consensus

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type Bytes32 [32]byte

func createPayloadEnvelope(blockNum uint64) *eth.ExecutionPayloadEnvelope {
	hash := common.HexToHash("0x12345")
	one := hexutil.Uint64(1)
	return &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: &hash,
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber:   eth.Uint64Quantity(blockNum),
			BlockHash:     common.HexToHash("0x888"),
			Withdrawals:   &types.Withdrawals{{Index: 1, Validator: 2, Address: common.HexToAddress("0x123"), Amount: 3}},
			ExcessBlobGas: &one,
			BlobGasUsed:   &one,
		}}
}
func TestUnsafeHeadTracker(t *testing.T) {
	tracker := &unsafeHeadTracker{
		log:        testlog.Logger(t, log.LevelDebug),
		unsafeHead: createPayloadEnvelope(222),
	}

	t.Run("Apply", func(t *testing.T) {
		data := createPayloadEnvelope(333)

		var buf bytes.Buffer
		_, err := data.MarshalSSZ(&buf)
		require.NoError(t, err)

		l := raft.Log{Data: buf.Bytes()}
		require.Nil(t, tracker.Apply(&l))
		require.Equal(t, hexutil.Uint64(333), tracker.unsafeHead.ExecutionPayload.BlockNumber)
	})

	t.Run("Snapshot", func(t *testing.T) {
		snapshot, err := tracker.Snapshot()
		require.NoError(t, err)

		sink := new(raft.DiscardSnapshotSink)

		err = snapshot.Persist(sink)
		require.NoError(t, err)
	})

	t.Run("Restore", func(t *testing.T) {
		data := createPayloadEnvelope(333)

		mrc, err := NewMockReadCloser(data)
		require.NoError(t, err)
		err = tracker.Restore(mrc)
		require.NoError(t, err)
		require.Equal(t, hexutil.Uint64(333), tracker.unsafeHead.ExecutionPayload.BlockNumber)
	})
}

// TestUnsafeHeadTrackerRetainsPayloads covers the payload history every member keeps so its
// conductor can bring its own op-node up to the consensus unsafe head across a multi-block gap.
func TestUnsafeHeadTrackerRetainsPayloads(t *testing.T) {
	// Blocks are chained by hash so a retained payload really is the child of its predecessor.
	// branch distinguishes competing blocks at the same height.
	hashOf := func(blockNum uint64, branch byte) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(blockNum<<8 | uint64(branch)))
	}
	payloadOn := func(blockNum uint64, branch, parentBranch byte) *eth.ExecutionPayloadEnvelope {
		payload := createPayloadEnvelope(blockNum)
		payload.ExecutionPayload.BlockHash = hashOf(blockNum, branch)
		payload.ExecutionPayload.ParentHash = hashOf(blockNum-1, parentBranch)
		return payload
	}
	applyOn := func(t *testing.T, tracker *unsafeHeadTracker, blockNum uint64, branch, parentBranch byte) {
		var buf bytes.Buffer
		_, err := payloadOn(blockNum, branch, parentBranch).MarshalSSZ(&buf)
		require.NoError(t, err)
		require.Nil(t, tracker.Apply(&raft.Log{Data: buf.Bytes()}))
	}
	apply := func(t *testing.T, tracker *unsafeHeadTracker, blockNum uint64) {
		applyOn(t, tracker, blockNum, 0, 0)
	}
	numbers := func(payloads []*eth.ExecutionPayloadEnvelope) []uint64 {
		nums := make([]uint64, len(payloads))
		for i, payload := range payloads {
			nums[i] = uint64(payload.ExecutionPayload.BlockNumber)
		}
		return nums
	}
	newTracker := func(t *testing.T) *unsafeHeadTracker {
		return NewUnsafeHeadTracker(testlog.Logger(t, log.LevelDebug))
	}

	t.Run("returns the contiguous suffix after a block number", func(t *testing.T) {
		tracker := newTracker(t)
		for num := uint64(10); num <= 14; num++ {
			apply(t, tracker, num)
		}

		require.Equal(t, []uint64{13, 14}, numbers(tracker.UnsafePayloadsAfter(12)))
		require.Equal(t, []uint64{11, 12, 13, 14}, numbers(tracker.UnsafePayloadsAfter(10)))
		require.Empty(t, tracker.UnsafePayloadsAfter(14), "nothing is missing at the head")
		for _, payload := range tracker.UnsafePayloadsAfter(10) {
			require.Equal(t, hashOf(uint64(payload.ExecutionPayload.BlockNumber)-1, 0), payload.ExecutionPayload.ParentHash,
				"replayed payloads must chain")
		}
	})

	t.Run("returns nothing when the history does not reach back", func(t *testing.T) {
		tracker := newTracker(t)
		apply(t, tracker, 14)

		require.Empty(t, tracker.UnsafePayloadsAfter(12), "block 13 was never applied here")
		require.Equal(t, []uint64{14}, numbers(tracker.UnsafePayloadsAfter(13)))
	})

	t.Run("drops the history on a discontinuity", func(t *testing.T) {
		tracker := newTracker(t)
		apply(t, tracker, 10)
		apply(t, tracker, 11)
		apply(t, tracker, 20)

		require.Empty(t, tracker.UnsafePayloadsAfter(10), "history must not span the skipped range")
		require.Equal(t, []uint64{20}, numbers(tracker.UnsafePayloadsAfter(19)))
	})

	t.Run("bounds the retained history", func(t *testing.T) {
		tracker := newTracker(t)
		last := uint64(maxRetainedUnsafePayloads * 3)
		for num := uint64(1); num <= last; num++ {
			apply(t, tracker, num)
		}

		require.Len(t, tracker.retained, maxRetainedUnsafePayloads)
		require.Len(t, tracker.UnsafePayloadsAfter(last-maxRetainedUnsafePayloads), maxRetainedUnsafePayloads)
		require.Empty(t, tracker.UnsafePayloadsAfter(last-maxRetainedUnsafePayloads-1))
	})

	t.Run("ignores payloads that do not advance the head", func(t *testing.T) {
		tracker := newTracker(t)
		apply(t, tracker, 10)
		apply(t, tracker, 11)
		apply(t, tracker, 11)

		require.Equal(t, []uint64{11}, numbers(tracker.UnsafePayloadsAfter(10)))
	})

	// Apply discards a competing block at a height it has already seen, so the next height on the
	// new branch is number-contiguous with the retained tail while chaining onto the block that
	// was discarded. Replaying that range would hand op-node payloads it cannot insert, so the
	// history has to be reset instead.
	t.Run("drops the history when the chain switches branch", func(t *testing.T) {
		tracker := newTracker(t)
		apply(t, tracker, 10)
		apply(t, tracker, 11)
		applyOn(t, tracker, 11, 1, 0)
		applyOn(t, tracker, 12, 1, 1)

		require.Equal(t, hashOf(12, 1), tracker.UnsafeHead().ExecutionPayload.BlockHash)
		require.Empty(t, tracker.UnsafePayloadsAfter(9), "history must not span the branch change")
		require.Empty(t, tracker.UnsafePayloadsAfter(10))
		require.Equal(t, []uint64{12}, numbers(tracker.UnsafePayloadsAfter(11)))
	})

	t.Run("bounds the retained history by size", func(t *testing.T) {
		tracker := newTracker(t)
		// One shared backing array: only the reported length matters, and a payload big enough to
		// make the byte bound bite is too big to allocate once per block here.
		calldata := make([]byte, maxRetainedUnsafePayloadsMemory/4)
		for num := uint64(10); num <= 15; num++ {
			payload := payloadOn(num, 0, 0)
			payload.ExecutionPayload.Transactions = []eth.Data{calldata}
			tracker.retain(payload)
		}

		require.LessOrEqual(t, tracker.retainedSize, uint64(maxRetainedUnsafePayloadsMemory))
		require.Less(t, len(tracker.retained), maxRetainedUnsafePayloads,
			"the size bound must bite long before the count bound on calldata-heavy blocks")
		require.NotEmpty(t, tracker.retained, "the newest payload is always retained")
	})

	t.Run("clears the history on restore", func(t *testing.T) {
		tracker := newTracker(t)
		apply(t, tracker, 10)
		apply(t, tracker, 11)

		mrc, err := NewMockReadCloser(createPayloadEnvelope(11))
		require.NoError(t, err)
		require.NoError(t, tracker.Restore(mrc))

		require.Empty(t, tracker.retained)
		require.Empty(t, tracker.UnsafePayloadsAfter(10))
	})
}

type mockReadCloser struct {
	currentPosition int
	data            *eth.ExecutionPayloadEnvelope
	buffer          []byte
}

func NewMockReadCloser(data *eth.ExecutionPayloadEnvelope) (*mockReadCloser, error) {
	mrc := &mockReadCloser{
		currentPosition: 0,
		data:            data,
		buffer:          make([]byte, 0),
	}

	var buf bytes.Buffer
	if _, err := data.MarshalSSZ(&buf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution payload envelope: %w", err)
	}
	mrc.buffer = buf.Bytes()

	return mrc, nil
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.currentPosition >= len(m.buffer) {
		return 0, io.EOF
	}

	end := m.currentPosition + len(p)
	if end > len(m.buffer) {
		end = len(m.buffer)
		err = io.EOF
	}
	n = copy(p, m.buffer[m.currentPosition:end])
	m.currentPosition = end
	return n, err
}

func (m *mockReadCloser) Close() error {
	return nil
}
