package derive

import (
	"bytes"
	"compress/zlib"
	"math/big"
	"math/rand"
	"testing"

	"github.com/DataDog/zstd"
	"github.com/andybalholm/brotli"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

type frameValidityTC struct {
	name      string
	frames    []Frame
	shouldErr []bool
	sizes     []uint64
}

func (tc *frameValidityTC) Run(t *testing.T) {
	id := [16]byte{0xff}
	block := eth.L1BlockRef{}
	ch := NewChannel(id, block)

	if len(tc.frames) != len(tc.shouldErr) || len(tc.frames) != len(tc.sizes) {
		t.Errorf("lengths should be the same. frames: %d, shouldErr: %d, sizes: %d", len(tc.frames), len(tc.shouldErr), len(tc.sizes))
	}

	for i, frame := range tc.frames {
		err := ch.AddFrame(frame, block)
		if tc.shouldErr[i] {
			require.NotNil(t, err)
		} else {
			require.Nil(t, err)
		}
		require.Equal(t, tc.sizes[i], ch.Size())
	}
}

// TestFrameValidity inserts a list of frames into the channel. It checks if an error
// should be returned by `AddFrame` as well as checking the size of the channel.
func TestFrameValidity(t *testing.T) {
	id := [16]byte{0xff}
	testCases := []frameValidityTC{
		{
			name:      "wrong channel",
			frames:    []Frame{{ID: [16]byte{0xee}}},
			shouldErr: []bool{true},
			sizes:     []uint64{0},
		},
		{
			name: "double close",
			frames: []Frame{
				{ID: id, FrameNumber: 2, IsLast: true, Data: []byte("four")},
				{ID: id, FrameNumber: 1, IsLast: true}},
			shouldErr: []bool{false, true},
			sizes:     []uint64{204, 204},
		},
		{
			name: "duplicate frame",
			frames: []Frame{
				{ID: id, FrameNumber: 2, Data: []byte("four")},
				{ID: id, FrameNumber: 2, Data: []byte("seven__")}},
			shouldErr: []bool{false, true},
			sizes:     []uint64{204, 204},
		},
		{
			name: "duplicate closing frames",
			frames: []Frame{
				{ID: id, FrameNumber: 2, IsLast: true, Data: []byte("four")},
				{ID: id, FrameNumber: 2, IsLast: true, Data: []byte("seven__")}},
			shouldErr: []bool{false, true},
			sizes:     []uint64{204, 204},
		},
		{
			name: "frame past closing",
			frames: []Frame{
				{ID: id, FrameNumber: 2, IsLast: true, Data: []byte("four")},
				{ID: id, FrameNumber: 10, Data: []byte("seven__")}},
			shouldErr: []bool{false, true},
			sizes:     []uint64{204, 204},
		},
		{
			name: "prune after close frame",
			frames: []Frame{
				{ID: id, FrameNumber: 10, IsLast: false, Data: []byte("seven__")},
				{ID: id, FrameNumber: 2, IsLast: true, Data: []byte("four")}},
			shouldErr: []bool{false, false},
			sizes:     []uint64{207, 204},
		},
		{
			name: "multiple valid frames",
			frames: []Frame{
				{ID: id, FrameNumber: 10, Data: []byte("seven__")},
				{ID: id, FrameNumber: 2, Data: []byte("four")}},
			shouldErr: []bool{false, false},
			sizes:     []uint64{207, 411},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.Run)
	}
}

func TestBatchReader(t *testing.T) {
	// Get batch data
	rng := rand.New(rand.NewSource(0x543331))
	singularBatch := RandomSingularBatch(rng, 20, big.NewInt(333))
	batchDataInput := NewBatchData(singularBatch)

	encodedBatch := &bytes.Buffer{}
	err := batchDataInput.EncodeRLP(encodedBatch)
	require.NoError(t, err)

	var testCases = []struct {
		name      string
		algo      func(buf *bytes.Buffer, t *testing.T)
		isFjord   bool
		expectErr bool
	}{
		{
			name: "zlib-post-fjord",
			algo: func(buf *bytes.Buffer, t *testing.T) {
				writer := zlib.NewWriter(buf)
				_, err := writer.Write(encodedBatch.Bytes())
				require.NoError(t, err)
				writer.Close()
			},
			isFjord: true,
		},
		{
			name: "zlib-pre-fjord",
			algo: func(buf *bytes.Buffer, t *testing.T) {
				writer := zlib.NewWriter(buf)
				_, err := writer.Write(encodedBatch.Bytes())
				require.NoError(t, err)
				writer.Close()
			},
			isFjord: false,
		},
		{
			name: "brotli9-post-fjord",
			algo: func(buf *bytes.Buffer, t *testing.T) {
				buf.WriteByte(ChannelVersionBrotli)
				writer := brotli.NewWriterLevel(buf, 9)
				_, err := writer.Write(encodedBatch.Bytes())
				require.NoError(t, err)
				writer.Close()
			},
			isFjord: true,
		},
		{
			name: "brotli9-pre-fjord",
			algo: func(buf *bytes.Buffer, t *testing.T) {
				buf.WriteByte(ChannelVersionBrotli)
				writer := brotli.NewWriterLevel(buf, 9)
				_, err := writer.Write(encodedBatch.Bytes())
				require.NoError(t, err)
				writer.Close()
			},
			isFjord:   false,
			expectErr: true, // expect an error because brotli is not supported before Fjord
		},
		{
			name: "brotli10-post-fjord",
			algo: func(buf *bytes.Buffer, t *testing.T) {
				buf.WriteByte(ChannelVersionBrotli)
				writer := brotli.NewWriterLevel(buf, 10)
				_, err := writer.Write(encodedBatch.Bytes())
				require.NoError(t, err)
				writer.Close()
			},
			isFjord: true,
		},
		{
			name: "brotli11-post-fjord",
			algo: func(buf *bytes.Buffer, t *testing.T) {
				buf.WriteByte(ChannelVersionBrotli)
				writer := brotli.NewWriterLevel(buf, 11)
				_, err := writer.Write(encodedBatch.Bytes())
				require.NoError(t, err)
				writer.Close()
			},
			isFjord: true,
		},
		{
			name: "zstd-post-fjord",
			algo: func(buf *bytes.Buffer, t *testing.T) {
				writer := zstd.NewWriter(buf)
				_, err := writer.Write(encodedBatch.Bytes())
				require.NoError(t, err)
				writer.Close()
			},
			expectErr: true,
			isFjord:   true,
		}}

	for _, tc := range testCases {
		compressed := new(bytes.Buffer)
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.algo(compressed, t)
			reader, err := BatchReader(bytes.NewReader(compressed.Bytes()), 120000, tc.isFjord)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// read the batch data
			batchData, err := reader()
			require.NoError(t, err)
			require.NotNil(t, batchData)
			require.Equal(t, batchDataInput, batchData)
		})
	}
}
