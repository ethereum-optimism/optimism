package types_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// This file checks the golden vectors of golden_vectors_test.go against op-geth. It only
// compiles against op-geth and goes away with it.

// opGethReceipt returns r as an op-geth receipt, which holds the OP Stack fields itself.
func opGethReceipt(r *optypes.Receipt) *gethtypes.Receipt {
	gr := r.Receipt
	gr.DepositNonce = r.DepositNonce
	gr.DepositReceiptVersion = r.DepositReceiptVersion
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
	return &gr
}

func TestDepositReceiptGoldenVectorsMatchOpGeth(t *testing.T) {
	for _, v := range depositReceiptVectors() {
		t.Run(v.name, func(t *testing.T) {
			gr := opGethReceipt(v.receipt)

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

func TestReceiptJSONGoldenVectorMatchesOpGeth(t *testing.T) {
	data, err := json.Marshal(opGethReceipt(receiptAllOpFields()))
	require.NoError(t, err)
	require.JSONEq(t, receiptAllOpFieldsJSON, string(data))
}
