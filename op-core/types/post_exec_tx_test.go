package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

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

// TestPostExecTxHashGoldenVector pins the cross-client post-exec (0x7D) tx hash: op-geth and
// op-reth must agree. Both compute keccak256(0x7D || Data), the canonical EIP-2718 rule. Data is
// RLP(PostExecPayload{version:1, block:42, entries:[{index:3, gasRefund:7}]}) — op-alloy's
// build_post_exec_tx(42, [{3,7}]) — and opRethTxHash is the value op-alloy's
// post_exec_tx_hash_is_keccak_of_canonical_encoding pins for TxPostExec::tx_hash. Guards the
// op-geth Transaction.Hash fix against regression when the op-geth pin moves (the old op-geth
// returned keccak256(0x7D || RLP([Data])) here).
func TestPostExecTxHashGoldenVector(t *testing.T) {
	data := hexutil.MustDecode("0xc6012ac3c20307")
	const opRethTxHash = "0xcbc64a9665039744307b562634bf760d96f3bed235fecd9e09a1f1276a1823c7"

	gethHash := gethtypes.NewTx(&gethtypes.PostExecTx{Data: data}).Hash()
	require.Equal(t, opRethTxHash, gethHash.Hex(),
		"op-geth post-exec tx hash must match op-reth (op-alloy TxPostExec::tx_hash)")

	// The shared value is exactly keccak256 of the canonical encoding op-core produces.
	canonical, err := (&optypes.PostExecTx{Data: data}).MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, gethHash, crypto.Keccak256Hash(canonical))
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
