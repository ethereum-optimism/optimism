package derive

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestChannelOutAddBlock(t *testing.T) {
	cout, err := NewChannelOut()
	require.NoError(t, err)

	t.Run("returns err if first tx is not an l1info tx", func(t *testing.T) {
		header := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(100)}
		block := types.NewBlockWithHeader(header).WithBody(
			[]*types.Transaction{
				types.NewTx(&types.DynamicFeeTx{}),
			},
			nil,
		)
		_, err := cout.AddBlock(block)
		require.Error(t, err)
		require.Equal(t, ErrNotDepositTx, err)
	})
}

// TestRLPByteLimit ensures that stream encoder is properly limiting the length.
// It will decode the input if `len(input) <= inputLimit`.
func TestRLPByteLimit(t *testing.T) {
	// Should succeed if `len(input) == inputLimit`
	enc := []byte("\x8bhello world") // RLP encoding of the string "hello world"
	in := bytes.NewBuffer(enc)
	var out string
	stream := rlp.NewStream(in, 12)
	err := stream.Decode(&out)
	require.Nil(t, err)
	require.Equal(t, out, "hello world")

	// Should fail if the `inputLimit = len(input) - 1`
	enc = []byte("\x8bhello world") // RLP encoding of the string "hello world"
	in = bytes.NewBuffer(enc)
	var out2 string
	stream = rlp.NewStream(in, 11)
	err = stream.Decode(&out2)
	require.Equal(t, err, rlp.ErrValueTooLarge)
	require.Equal(t, out2, "")
}

func TestForceCloseTxData(t *testing.T) {
	id := [16]byte{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef}
	tests := []struct {
		frames []Frame
		errors bool
		output string
	}{
		{
			frames: []Frame{},
			errors: true,
			output: "",
		},
		{
			frames: []Frame{Frame{FrameNumber: 0, IsLast: false}, Frame{ID: id, FrameNumber: 1, IsLast: true}},
			errors: true,
			output: "",
		},
		{
			frames: []Frame{Frame{ID: id, FrameNumber: 0, IsLast: false}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000001",
		},
		{
			frames: []Frame{Frame{ID: id, FrameNumber: 0, IsLast: true}},
			errors: false,
			output: "00",
		},
		{
			frames: []Frame{Frame{ID: id, FrameNumber: 1, IsLast: false}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000001",
		},
		{
			frames: []Frame{Frame{ID: id, FrameNumber: 1, IsLast: true}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000000",
		},
		{
			frames: []Frame{Frame{ID: id, FrameNumber: 2, IsLast: true}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000000deadbeefdeadbeefdeadbeefdeadbeef00010000000000",
		},
		{
			frames: []Frame{Frame{ID: id, FrameNumber: 1, IsLast: false}, Frame{ID: id, FrameNumber: 3, IsLast: true}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000000deadbeefdeadbeefdeadbeefdeadbeef00020000000000",
		},
		{
			frames: []Frame{Frame{ID: id, FrameNumber: 1, IsLast: false}, Frame{ID: id, FrameNumber: 3, IsLast: true}, Frame{ID: id, FrameNumber: 5, IsLast: true}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000000deadbeefdeadbeefdeadbeefdeadbeef00020000000000",
		},
	}

	for i, test := range tests {
		out, err := ForceCloseTxData(test.frames)
		if test.errors {
			require.NotNil(t, err, "Should error on tc %v", i)
			require.Nil(t, out, "Should return no value in tc %v", i)
		} else {
			require.NoError(t, err, "Should not error on tc %v", i)
			require.Equal(t, common.FromHex(test.output), out, "Should match output tc %v", i)
		}
	}
}
