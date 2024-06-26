package consensus

import (
	"bytes"
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
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
		return nil, errors.Wrap(err, "failed to unmarshal execution payload envelope")
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
