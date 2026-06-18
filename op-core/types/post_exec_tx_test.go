package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	gethtypes "github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

func postExecTxTestCases() []optypes.PostExecTx {
	return []optypes.PostExecTx{
		{Data: []byte{0x01}},
		{Data: []byte{0xde, 0xad, 0xbe, 0xef}},
		{Data: make([]byte, 1024)},
	}
}

// TestPostExecTxMarshalBinaryDifferential asserts that PostExecTx.MarshalBinary
// is byte-for-byte identical to op-geth's encoding of the same transaction.
// It will be removed in the final cutover, when the op-geth dependency is
// replaced with upstream go-ethereum.
func TestPostExecTxMarshalBinaryDifferential(t *testing.T) {
	for _, tx := range postExecTxTestCases() {
		ours, err := tx.MarshalBinary()
		require.NoError(t, err)

		theirs, err := gethtypes.NewTx(&gethtypes.PostExecTx{Data: tx.Data}).MarshalBinary()
		require.NoError(t, err)

		require.Equal(t, theirs, ours)
	}
}

func TestUnmarshalPostExecTxRoundTrip(t *testing.T) {
	for _, tx := range postExecTxTestCases() {
		raw, err := tx.MarshalBinary()
		require.NoError(t, err)

		decoded, err := optypes.UnmarshalPostExecTx(raw)
		require.NoError(t, err)
		require.Equal(t, tx.Data, decoded.Data)
	}
}

func TestUnmarshalPostExecTxErrors(t *testing.T) {
	_, err := optypes.UnmarshalPostExecTx(nil)
	require.ErrorContains(t, err, "empty")

	_, err = optypes.UnmarshalPostExecTx([]byte{optypes.DepositTxType, 0x01})
	require.ErrorContains(t, err, "type byte")

	// op-geth rejects a bare type byte without payload, and so do we
	var gethTx gethtypes.Transaction
	require.ErrorContains(t, gethTx.UnmarshalBinary([]byte{optypes.PostExecTxType}), "too short")
	_, err = optypes.UnmarshalPostExecTx([]byte{optypes.PostExecTxType})
	require.ErrorContains(t, err, "payload is empty")
}

// TestPostExecTxEmptyData pins the marshal/unmarshal asymmetry for an empty
// payload: both implementations produce the bare type byte, and neither
// accepts it back.
func TestPostExecTxEmptyData(t *testing.T) {
	bareTypeByte := []byte{optypes.PostExecTxType}

	ours, err := (&optypes.PostExecTx{Data: nil}).MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, bareTypeByte, ours)

	theirs, err := gethtypes.NewTx(&gethtypes.PostExecTx{Data: nil}).MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, bareTypeByte, theirs)

	var gethTx gethtypes.Transaction
	require.Error(t, gethTx.UnmarshalBinary(bareTypeByte))
	_, err = optypes.UnmarshalPostExecTx(bareTypeByte)
	require.Error(t, err)
}

func TestIsPostExecTx(t *testing.T) {
	raw, err := gethtypes.NewTx(&gethtypes.PostExecTx{Data: []byte{0x42}}).MarshalBinary()
	require.NoError(t, err)
	tx := new(gethtypes.Transaction)
	require.NoError(t, tx.UnmarshalBinary(raw))

	require.True(t, optypes.IsPostExecTx(tx))
	require.False(t, optypes.IsDepositTx(tx))

	depositTx := depositTxTestCases()[0].gethTx()
	require.False(t, optypes.IsPostExecTx(depositTx))
}
