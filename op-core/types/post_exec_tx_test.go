package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
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

// TestPostExecTxHashGoldenVector pins the cross-client post-exec (0x7D) tx hash: the Go
// encoding and op-reth must agree on keccak256(0x7D || Data), the canonical EIP-2718 rule. data
// is op-alloy's build_post_exec_tx(42, [{3,7}]); opRethTxHash is the value op-alloy pins for
// TxPostExec::tx_hash.
func TestPostExecTxHashGoldenVector(t *testing.T) {
	data := hexutil.MustDecode("0xc6012ac3c20307")
	const opRethTxHash = "0xcbc64a9665039744307b562634bf760d96f3bed235fecd9e09a1f1276a1823c7"

	// The shared value is exactly keccak256 of the canonical encoding op-core produces.
	canonical, err := (&optypes.PostExecTx{Data: data}).MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, opRethTxHash, crypto.Keccak256Hash(canonical).Hex(),
		"post-exec tx hash must match op-reth (op-alloy TxPostExec::tx_hash)")
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

	_, err = optypes.UnmarshalPostExecTx([]byte{optypes.PostExecTxType})
	require.ErrorContains(t, err, "payload is empty")
}

// TestPostExecTxEmptyData pins the marshal/unmarshal asymmetry for an empty
// payload: marshaling produces the bare type byte, which does not decode.
func TestPostExecTxEmptyData(t *testing.T) {
	bareTypeByte := []byte{optypes.PostExecTxType}

	ours, err := (&optypes.PostExecTx{Data: nil}).MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, bareTypeByte, ours)

	_, err = optypes.UnmarshalPostExecTx(bareTypeByte)
	require.Error(t, err)
}
