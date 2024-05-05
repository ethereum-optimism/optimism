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
	singularBatch := RandomSingularBatch(rng, 5000, big.NewInt(333))
	batchDataInput := NewBatchData(singularBatch)

	encodedBatch := &bytes.Buffer{}
	// Get the encoded data of the batch data
	batchDataInput.encodeTyped(encodedBatch)

	var testCases = []struct {
		name string
		algo func(buf *bytes.Buffer)
	}{{
		name: "zlib",
		algo: func(buf *bytes.Buffer) {
			writer := zlib.NewWriter(buf)
			writer.Write(encodedBatch.Bytes())
		},
	},
		{
			name: "brotli10",
			algo: func(buf *bytes.Buffer) {
				buf.WriteByte(ChannelVersionBrotli)
				writer := brotli.NewWriterLevel(buf, 10)
				writer.Write(encodedBatch.Bytes())
			},
		}, {
			name: "zstd",
			algo: func(buf *bytes.Buffer) {
				writer := zstd.NewWriter(buf)
				writer.Write(encodedBatch.Bytes())
			},
		}}

	for _, tc := range testCases {
		compressed := bytes.NewBuffer([]byte{})
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.algo(compressed)
			reader, err := BatchReader(bytes.NewReader(compressed.Bytes()))
			if tc.name == "zstd" {
				require.NotNil(t, err)
				return
			}
			require.Nil(t, err)

			// read the batch data
			batchData, err := reader()
			require.Nil(t, err)
			require.NotNil(t, batchData)
			require.Equal(t, batchDataInput, batchData)
		})
	}

}
