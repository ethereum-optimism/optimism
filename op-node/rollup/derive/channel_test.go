package derive

import (
	"testing"

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
