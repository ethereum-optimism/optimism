package consensus

import (
	"bytes"
	"fmt"
	"io"
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

func TestUnsafeHeadTrackerRestore(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)

	t.Run("empty snapshot resets unsafe head", func(t *testing.T) {
		tracker := &unsafeHeadTracker{log: logger, unsafeHead: createPayloadEnvelope(222)}
		require.NoError(t, tracker.Restore(io.NopCloser(bytes.NewReader(nil))))
		require.Nil(t, tracker.UnsafeHead())
	})

	// Guards that only the exact zero-length case is accepted; short non-empty inputs must still fail.
	t.Run("truncated snapshot is rejected", func(t *testing.T) {
		var buf bytes.Buffer
		_, err := createPayloadEnvelope(333).MarshalSSZ(&buf)
		require.NoError(t, err)

		for _, size := range []int{1, 31, 32, 40, buf.Len() - 1} {
			tracker := &unsafeHeadTracker{log: logger, unsafeHead: createPayloadEnvelope(222)}
			err := tracker.Restore(io.NopCloser(bytes.NewReader(buf.Bytes()[:size])))
			require.Error(t, err, "snapshot truncated to %d bytes must be rejected", size)
			require.Equal(t, hexutil.Uint64(222), tracker.UnsafeHead().ExecutionPayload.BlockNumber)
		}
	})
}

// TestUnsafeHeadTrackerSnapshotRoundTrip persists and restores snapshots through the file snapshot store that
// op-conductor uses in production, so the zero-length snapshot path is exercised end to end.
func TestUnsafeHeadTrackerSnapshotRoundTrip(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)

	cases := []struct {
		name       string
		unsafeHead *eth.ExecutionPayloadEnvelope
	}{
		{name: "nil unsafe head", unsafeHead: nil},
		{name: "committed unsafe head", unsafeHead: createPayloadEnvelope(333)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, err := raft.NewFileSnapshotStore(t.TempDir(), 1, io.Discard)
			require.NoError(t, err)

			source := &unsafeHeadTracker{log: logger, unsafeHead: tc.unsafeHead}
			snap, err := source.Snapshot()
			require.NoError(t, err)
			sink, err := store.Create(raft.SnapshotVersionMax, 1, 1, raft.Configuration{}, 1, nil)
			require.NoError(t, err)
			require.NoError(t, snap.Persist(sink))

			metas, err := store.List()
			require.NoError(t, err)
			require.Len(t, metas, 1)
			_, rc, err := store.Open(metas[0].ID)
			require.NoError(t, err)

			restored := &unsafeHeadTracker{log: logger, unsafeHead: createPayloadEnvelope(222)}
			require.NoError(t, restored.Restore(rc))
			if tc.unsafeHead == nil {
				require.Nil(t, restored.UnsafeHead())
				return
			}
			require.Equal(t, tc.unsafeHead.ExecutionPayload.BlockNumber, restored.UnsafeHead().ExecutionPayload.BlockNumber)
			require.Equal(t, tc.unsafeHead.ExecutionPayload.BlockHash, restored.UnsafeHead().ExecutionPayload.BlockHash)
		})
	}
}
