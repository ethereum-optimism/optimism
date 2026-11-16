package types_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// postExecTxJSONVectors are post-exec transactions with their RPC JSON encoding, as
// op-geth encodes them.
var postExecTxJSONVectors = []struct {
	data []byte
	json string
}{
	{
		data: []byte{0x01},
		json: `{"type":"0x7d","nonce":null,"to":null,"gas":"0x0","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0x01","v":null,"r":null,"s":null,"hash":"0x12b72a1cf763f3084b38506a32995d2bfcb83b3b7ea0045f0d367ca21cc870fd"}`,
	},
	{
		data: []byte{0xc2, 0x80, 0x80},
		json: `{"type":"0x7d","nonce":null,"to":null,"gas":"0x0","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0xc28080","v":null,"r":null,"s":null,"hash":"0x9b7c909256b87c342f4bbe84bc7d42bacf14448124e00f8bb8962149c19b898f"}`,
	},
	{
		data: make([]byte, 64),
		json: `{"type":"0x7d","nonce":null,"to":null,"gas":"0x0","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0x` + strings.Repeat("00", 64) + `","v":null,"r":null,"s":null,"hash":"0xca1f045b3a509d5e59170a201e15c8be7450c832831e339c9596faf08d1df0db"}`,
	},
}

func TestPostExecTxJSONGoldenVectors(t *testing.T) {
	for _, v := range postExecTxJSONVectors {
		data, err := json.Marshal(&optypes.PostExecTx{Data: v.data})
		require.NoError(t, err)
		require.JSONEq(t, v.json, string(data))

		var decoded optypes.PostExecTx
		require.NoError(t, json.Unmarshal([]byte(v.json), &decoded))
		require.Equal(t, v.data, decoded.Data)
	}
}

// zeroBloomRLP is the RLP encoding of an empty logs bloom.
var zeroBloomRLP = "b90100" + strings.Repeat("00", 256)

type depositReceiptVector struct {
	name    string
	receipt *optypes.Receipt
	// preimage is the receipt's encoding in the receipts trie (Receipts.EncodeIndex).
	preimage string
	// binary is the receipt's MarshalBinary encoding, which includes the nonce whenever
	// it is set.
	binary string
	// rlp is the receipt's RLP encoding: binary as an RLP string.
	rlp string
}

// depositReceiptVectors are deposit receipts in each consensus shape, with their encodings
// as op-geth computes them: before Regolith a deposit receipt has no nonce, from Regolith it
// has a nonce that the receipts trie leaves out, and from Canyon it has a nonce and a
// receipt version, both in the receipts trie.
func depositReceiptVectors() []depositReceiptVector {
	mk := func(status uint64, nonce, version *uint64) *optypes.Receipt {
		return &optypes.Receipt{
			Receipt: gethtypes.Receipt{
				Type:              optypes.DepositTxType,
				Status:            status,
				CumulativeGasUsed: 42_000,
				Logs:              []*gethtypes.Log{},
			},
			DepositNonce:          nonce,
			DepositReceiptVersion: version,
		}
	}
	return []depositReceiptVector{
		{
			name:     "pre-Regolith",
			receipt:  mk(gethtypes.ReceiptStatusSuccessful, nil, nil),
			preimage: "0x7ef901080182a410" + zeroBloomRLP + "c0",
			binary:   "0x7ef901080182a410" + zeroBloomRLP + "c0",
			rlp:      "0xb9010c7ef901080182a410" + zeroBloomRLP + "c0",
		},
		{
			name:     "pre-Canyon",
			receipt:  mk(gethtypes.ReceiptStatusSuccessful, uint64Ptr(7), nil),
			preimage: "0x7ef901080182a410" + zeroBloomRLP + "c0",
			binary:   "0x7ef901090182a410" + zeroBloomRLP + "c007",
			rlp:      "0xb9010d7ef901090182a410" + zeroBloomRLP + "c007",
		},
		{
			name:     "post-Canyon",
			receipt:  mk(gethtypes.ReceiptStatusFailed, uint64Ptr(8), uint64Ptr(optypes.CanyonDepositReceiptVersion)),
			preimage: "0x7ef9010a8082a410" + zeroBloomRLP + "c00801",
			binary:   "0x7ef9010a8082a410" + zeroBloomRLP + "c00801",
			rlp:      "0xb9010e7ef9010a8082a410" + zeroBloomRLP + "c00801",
		},
	}
}

// TestDepositReceiptGoldenVectors pins the consensus encodings of deposit receipts, and
// that the binary and RLP encodings decode back to the receipt.
func TestDepositReceiptGoldenVectors(t *testing.T) {
	for _, v := range depositReceiptVectors() {
		t.Run(v.name, func(t *testing.T) {
			var preimage bytes.Buffer
			optypes.Receipts{v.receipt}.EncodeIndex(0, &preimage)
			require.Equal(t, v.preimage, hexutil.Encode(preimage.Bytes()))

			binary, err := v.receipt.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, v.binary, hexutil.Encode(binary))

			encoded, err := rlp.EncodeToBytes(v.receipt)
			require.NoError(t, err)
			require.Equal(t, v.rlp, hexutil.Encode(encoded))

			var fromBinary optypes.Receipt
			require.NoError(t, fromBinary.UnmarshalBinary(hexutil.MustDecode(v.binary)))
			var fromRLP optypes.Receipt
			require.NoError(t, rlp.DecodeBytes(hexutil.MustDecode(v.rlp), &fromRLP))
			for _, got := range []*optypes.Receipt{&fromBinary, &fromRLP} {
				require.Equal(t, v.receipt.Type, got.Type)
				require.Equal(t, v.receipt.Status, got.Status)
				require.Equal(t, v.receipt.CumulativeGasUsed, got.CumulativeGasUsed)
				require.Equal(t, v.receipt.DepositNonce, got.DepositNonce)
				require.Equal(t, v.receipt.DepositReceiptVersion, got.DepositReceiptVersion)
			}
		})
	}
}

// receiptAllOpFields is gethReceipt with every OP Stack extension field set. A real
// receipt never has all of them at once, but the wire format does not depend on that.
func receiptAllOpFields() *optypes.Receipt {
	return &optypes.Receipt{
		Receipt:               *gethReceipt(),
		DepositNonce:          uint64Ptr(12),
		DepositReceiptVersion: uint64Ptr(1),
		L1GasPrice:            big.NewInt(42_000_000_000),
		L1BlobBaseFee:         big.NewInt(123_456),
		L1GasUsed:             big.NewInt(2048),
		L1Fee:                 big.NewInt(77_000),
		FeeScalar:             big.NewFloat(0.5),
		L1BaseFeeScalar:       uint64Ptr(5227),
		L1BlobBaseFeeScalar:   uint64Ptr(1014213),
		OperatorFeeScalar:     uint64Ptr(7),
		OperatorFeeConstant:   uint64Ptr(9000),
		DAFootprintGasScalar:  uint64Ptr(160),
	}
}

// receiptAllOpFieldsJSON is the eth_getTransactionReceipt encoding of receiptAllOpFields,
// as op-geth encodes it.
var receiptAllOpFieldsJSON = `{"type":"0x2","root":"0x","status":"0x1","cumulativeGasUsed":"0x186a0","logsBloom":"0x` + strings.Repeat("00", 256) + `",` +
	`"logs":[{"address":"0x1111111111111111111111111111111111111111","topics":["0x0000000000000000000000000000000000000000000000000000000000000001"],"data":"0x42","blockNumber":"0x0","transactionHash":"0x0000000000000000000000000000000000000000000000000000000000000000","transactionIndex":"0x0","blockHash":"0x0000000000000000000000000000000000000000000000000000000000000000","blockTimestamp":"0x0","logIndex":"0x0","removed":false}],` +
	`"transactionHash":"0x000000000000000000000000000000000000000000000000000000000000aaaa","contractAddress":"0x0000000000000000000000000000000000000000","gasUsed":"0x5208","effectiveGasPrice":"0x3b9aca00",` +
	`"depositNonce":"0xc","depositReceiptVersion":"0x1","blockHash":"0x000000000000000000000000000000000000000000000000000000000000bbbb","blockNumber":"0x4d2","transactionIndex":"0x3",` +
	`"l1GasPrice":"0x9c7652400","l1BlobBaseFee":"0x1e240","l1GasUsed":"0x800","l1Fee":"0x12cc8","l1FeeScalar":"0.5","l1BaseFeeScalar":"0x146b","l1BlobBaseFeeScalar":"0xf79c5",` +
	`"operatorFeeScalar":"0x7","operatorFeeConstant":"0x2328","daFootprintGasScalar":"0xa0"}`

// TestReceiptJSONGoldenVector pins the JSON encoding of a receipt with every OP Stack
// extension field, and that it decodes to the same fields.
func TestReceiptJSONGoldenVector(t *testing.T) {
	want := receiptAllOpFields()
	data, err := json.Marshal(want)
	require.NoError(t, err)
	require.JSONEq(t, receiptAllOpFieldsJSON, string(data))

	var got optypes.Receipt
	require.NoError(t, json.Unmarshal([]byte(receiptAllOpFieldsJSON), &got))
	require.Equal(t, want.DepositNonce, got.DepositNonce)
	require.Equal(t, want.DepositReceiptVersion, got.DepositReceiptVersion)
	require.Equal(t, want.L1GasPrice, got.L1GasPrice)
	require.Equal(t, want.L1BlobBaseFee, got.L1BlobBaseFee)
	require.Equal(t, want.L1GasUsed, got.L1GasUsed)
	require.Equal(t, want.L1Fee, got.L1Fee)
	require.NotNil(t, got.FeeScalar)
	require.Zero(t, want.FeeScalar.Cmp(got.FeeScalar))
	require.Equal(t, want.L1BaseFeeScalar, got.L1BaseFeeScalar)
	require.Equal(t, want.L1BlobBaseFeeScalar, got.L1BlobBaseFeeScalar)
	require.Equal(t, want.OperatorFeeScalar, got.OperatorFeeScalar)
	require.Equal(t, want.OperatorFeeConstant, got.OperatorFeeConstant)
	require.Equal(t, want.DAFootprintGasScalar, got.DAFootprintGasScalar)
	require.Equal(t, want.TxHash, got.TxHash)
	require.Equal(t, want.GasUsed, got.GasUsed)
}
