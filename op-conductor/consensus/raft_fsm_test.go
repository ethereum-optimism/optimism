package consensus

import (
	"bytes"
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestUnsafeHeadData(t *testing.T) {
	t.Run("should marshal and unmarshal unsafe head data correctly", func(t *testing.T) {
		data := &unsafeHeadData{
			version: eth.BlockV1,
			payload: eth.ExecutionPayload{
				BlockNumber: hexutil.Uint64(1),
			},
		}

		var buf bytes.Buffer
		_, err := data.MarshalSSZ(&buf)
		require.NoError(t, err)

		var unmarshalled unsafeHeadData
		err = unmarshalled.UnmarshalSSZ(&buf)
		require.NoError(t, err)
		require.Equal(t, eth.BlockV1, unmarshalled.version)
		require.Equal(t, hexutil.Uint64(1), unmarshalled.payload.BlockNumber)
	})
}

func TestUnsafeHeadTracker(t *testing.T) {
	tracker := &unsafeHeadTracker{
		unsafeHead: unsafeHeadData{
			version: eth.BlockV1,
			payload: eth.ExecutionPayload{
				BlockNumber: hexutil.Uint64(1),
			},
		},
	}

	t.Run("Apply", func(t *testing.T) {
		unsafeHeadData := unsafeHeadData{
			version: eth.BlockV2,
			payload: eth.ExecutionPayload{
				BlockNumber: hexutil.Uint64(2),
				Withdrawals: &types.Withdrawals{},
			},
		}

		var buf bytes.Buffer
		_, err := unsafeHeadData.MarshalSSZ(&buf)
		require.NoError(t, err)

		l := raft.Log{Data: buf.Bytes()}
		require.Nil(t, tracker.Apply(&l))
		require.Equal(t, eth.BlockV2, tracker.unsafeHead.version)
		require.Equal(t, hexutil.Uint64(2), tracker.unsafeHead.payload.BlockNumber)
	})

	t.Run("Restore", func(t *testing.T) {
		data := unsafeHeadData{
			version: eth.BlockV1,
			payload: eth.ExecutionPayload{
				BlockNumber: hexutil.Uint64(2),
			},
		}
		mrc := NewMockReadCloser(data)
		err := tracker.Restore(mrc)
		require.NoError(t, err)
		require.Equal(t, eth.BlockV1, tracker.unsafeHead.version)
		require.Equal(t, hexutil.Uint64(2), tracker.unsafeHead.payload.BlockNumber)
	})
}

type mockReadCloser struct {
	currentPosition int
	data            unsafeHeadData
	buffer          []byte
}

func NewMockReadCloser(data unsafeHeadData) *mockReadCloser {
	mrc := &mockReadCloser{
		currentPosition: 0,
		data:            data,
		buffer:          make([]byte, 0),
	}

	var buf bytes.Buffer
	if _, err := data.MarshalSSZ(&buf); err != nil {
		return nil
	}
	mrc.buffer = buf.Bytes()

	return mrc
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	var end int
	if len(m.buffer)-m.currentPosition < len(p) {
		end = len(m.buffer)
		err = io.EOF
	} else {
		end = m.currentPosition + len(p)
		err = nil
	}

	copy(p, m.buffer[m.currentPosition:end])
	m.currentPosition = end
	return end, err
}

func (m *mockReadCloser) Close() error {
	return nil
}
