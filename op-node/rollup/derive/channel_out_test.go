package derive

import (
	"bytes"
	"math/big"
	"testing"

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
		err := cout.AddBlock(block)
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
