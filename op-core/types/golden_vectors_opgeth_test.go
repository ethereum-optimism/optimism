package types_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// This file checks golden vectors of golden_vectors_test.go that no differential test
// covers against op-geth. It only compiles against op-geth and goes away with it.

// opGethReceipt returns r as an op-geth receipt, which holds the OP Stack fields itself.
func opGethReceipt(r *receiptVector) *gethtypes.Receipt {
	return opGethReceiptOf(r.receipt.Receipt, r.receipt.DepositNonce, r.receipt.DepositReceiptVersion)
}

func opGethReceiptOf(gr gethtypes.Receipt, nonce, version *uint64) *gethtypes.Receipt {
	gr.DepositNonce = nonce
	gr.DepositReceiptVersion = version
	return &gr
}

func TestReceiptGoldenVectorsMatchOpGeth(t *testing.T) {
	for _, v := range receiptVectors() {
		t.Run(v.name, func(t *testing.T) {
			gr := opGethReceipt(&v)

			var preimage bytes.Buffer
			gethtypes.Receipts{gr}.EncodeIndex(0, &preimage)
			require.Equal(t, v.preimage, hexutil.Encode(preimage.Bytes()))

			binary, err := gr.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, v.binary, hexutil.Encode(binary))

			encoded, err := rlp.EncodeToBytes(gr)
			require.NoError(t, err)
			require.Equal(t, v.rlp, hexutil.Encode(encoded))
		})
	}
}

func TestReceiptsRootGoldenVectorMatchesOpGeth(t *testing.T) {
	var receipts gethtypes.Receipts
	for _, r := range mixedReceipts() {
		receipts = append(receipts, opGethReceiptOf(r.Receipt, r.DepositNonce, r.DepositReceiptVersion))
	}
	require.Equal(t, mixedReceiptsRoot, gethtypes.DeriveSha(receipts, trie.NewStackTrie(nil)).Hex())
}

func TestReceiptJSONGoldenVectorMatchesOpGeth(t *testing.T) {
	r := receiptAllOpFields()
	gr := opGethReceiptOf(r.Receipt, r.DepositNonce, r.DepositReceiptVersion)
	gr.L1GasPrice = r.L1GasPrice
	gr.L1BlobBaseFee = r.L1BlobBaseFee
	gr.L1GasUsed = r.L1GasUsed
	gr.L1Fee = r.L1Fee
	gr.FeeScalar = r.FeeScalar
	gr.L1BaseFeeScalar = r.L1BaseFeeScalar
	gr.L1BlobBaseFeeScalar = r.L1BlobBaseFeeScalar
	gr.OperatorFeeScalar = r.OperatorFeeScalar
	gr.OperatorFeeConstant = r.OperatorFeeConstant
	gr.DAFootprintGasScalar = r.DAFootprintGasScalar

	data, err := json.Marshal(gr)
	require.NoError(t, err)
	require.JSONEq(t, receiptAllOpFieldsJSON, string(data))
}

func TestDepositTxJSONWithoutIsSystemTxMatchesOpGeth(t *testing.T) {
	for _, tc := range depositTxTestCases() {
		if tc.tx.IsSystemTransaction {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			var tx gethtypes.Transaction
			require.NoError(t, tx.UnmarshalJSON(depositTxWithoutIsSystemTxJSON(t, tc)))
			raw, err := tx.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, tc.binary, hexutil.Encode(raw))
		})
	}
}
