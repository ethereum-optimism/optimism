package types_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

func uint64Ptr(v uint64) *uint64 { return &v }

func gethReceipt() *gethtypes.Receipt {
	return &gethtypes.Receipt{
		Type:              gethtypes.DynamicFeeTxType,
		Status:            gethtypes.ReceiptStatusSuccessful,
		CumulativeGasUsed: 100_000,
		Logs: []*gethtypes.Log{
			{
				Address: common.HexToAddress("0x1111111111111111111111111111111111111111"),
				Topics:  []common.Hash{common.HexToHash("0x01")},
				Data:    []byte{0x42},
			},
		},
		TxHash:            common.HexToHash("0xaaaa"),
		GasUsed:           21_000,
		EffectiveGasPrice: big.NewInt(1_000_000_000),
		BlockHash:         common.HexToHash("0xbbbb"),
		BlockNumber:       big.NewInt(1234),
		TransactionIndex:  3,
	}
}

// gethReceiptAllOpFields sets every OP Stack extension field. A real receipt
// never has all of them at once (e.g. deposit fields exclude fee fields), but
// for wire-format testing the combination is irrelevant.
func gethReceiptAllOpFields() *gethtypes.Receipt {
	gr := gethReceipt()
	gr.DepositNonce = uint64Ptr(12)
	gr.DepositReceiptVersion = uint64Ptr(1)
	gr.L1GasPrice = big.NewInt(42_000_000_000)
	gr.L1BlobBaseFee = big.NewInt(123_456)
	gr.L1GasUsed = big.NewInt(2048)
	gr.L1Fee = big.NewInt(77_000)
	gr.FeeScalar = big.NewFloat(0.5)
	gr.L1BaseFeeScalar = uint64Ptr(5227)
	gr.L1BlobBaseFeeScalar = uint64Ptr(1014213)
	gr.OperatorFeeScalar = uint64Ptr(7)
	gr.OperatorFeeConstant = uint64Ptr(9000)
	gr.DAFootprintGasScalar = uint64Ptr(160)
	return gr
}

// TestReceiptUnmarshalJSONDifferential asserts that the receipt JSON produced
// by op-geth — the wire format of eth_getTransactionReceipt on an L2 endpoint —
// decodes into optypes.Receipt with all OP Stack extension fields intact.
// It will be removed in the final cutover, when the op-geth dependency is
// replaced with upstream go-ethereum.
func TestReceiptUnmarshalJSONDifferential(t *testing.T) {
	gr := gethReceiptAllOpFields()

	wire, err := json.Marshal(gr)
	require.NoError(t, err)

	var r optypes.Receipt
	require.NoError(t, json.Unmarshal(wire, &r))

	require.Equal(t, gr.DepositNonce, r.DepositNonce)
	require.Equal(t, gr.DepositReceiptVersion, r.DepositReceiptVersion)
	require.Equal(t, gr.L1GasPrice, r.L1GasPrice)
	require.Equal(t, gr.L1BlobBaseFee, r.L1BlobBaseFee)
	require.Equal(t, gr.L1GasUsed, r.L1GasUsed)
	require.Equal(t, gr.L1Fee, r.L1Fee)
	require.NotNil(t, r.FeeScalar)
	require.Zero(t, gr.FeeScalar.Cmp(r.FeeScalar))
	require.Equal(t, gr.L1BaseFeeScalar, r.L1BaseFeeScalar)
	require.Equal(t, gr.L1BlobBaseFeeScalar, r.L1BlobBaseFeeScalar)
	require.Equal(t, gr.OperatorFeeScalar, r.OperatorFeeScalar)
	require.Equal(t, gr.OperatorFeeConstant, r.OperatorFeeConstant)
	require.Equal(t, gr.DAFootprintGasScalar, r.DAFootprintGasScalar)

	// embedded standard fields survive
	require.Equal(t, gr.TxHash, r.TxHash)
	require.Equal(t, gr.Status, r.Status)
	require.Equal(t, gr.CumulativeGasUsed, r.CumulativeGasUsed)
	require.Equal(t, gr.GasUsed, r.GasUsed)
	require.Equal(t, gr.EffectiveGasPrice, r.EffectiveGasPrice)
	require.Equal(t, gr.BlockNumber, r.BlockNumber)
	require.Len(t, r.Logs, 1)
	require.Equal(t, gr.Logs[0].Address, r.Logs[0].Address)

	// re-encoding matches op-geth's encoding, modulo JSON key order
	reencoded, err := json.Marshal(&r)
	require.NoError(t, err)
	require.JSONEq(t, string(wire), string(reencoded))
}

func TestReceiptUnmarshalJSONWithoutOpFields(t *testing.T) {
	// an L1 receipt has none of the OP Stack extension fields
	wire, err := json.Marshal(gethReceipt())
	require.NoError(t, err)

	var r optypes.Receipt
	require.NoError(t, json.Unmarshal(wire, &r))

	require.Nil(t, r.DepositNonce)
	require.Nil(t, r.DepositReceiptVersion)
	require.Nil(t, r.L1GasPrice)
	require.Nil(t, r.L1BlobBaseFee)
	require.Nil(t, r.L1GasUsed)
	require.Nil(t, r.L1Fee)
	require.Nil(t, r.FeeScalar)
	require.Nil(t, r.L1BaseFeeScalar)
	require.Nil(t, r.L1BlobBaseFeeScalar)
	require.Nil(t, r.OperatorFeeScalar)
	require.Nil(t, r.OperatorFeeConstant)
	require.Nil(t, r.DAFootprintGasScalar)
	require.Equal(t, common.HexToHash("0xaaaa"), r.TxHash)

	reencoded, err := json.Marshal(&r)
	require.NoError(t, err)
	require.JSONEq(t, string(wire), string(reencoded))
}

func TestReceiptUnmarshalJSONNullOpFields(t *testing.T) {
	wire, err := json.Marshal(gethReceipt())
	require.NoError(t, err)
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(wire, &obj))
	for _, k := range []string{"l1GasPrice", "l1Fee", "l1FeeScalar", "l1BaseFeeScalar", "operatorFeeScalar", "depositNonce"} {
		obj[k] = json.RawMessage("null")
	}
	wire, err = json.Marshal(obj)
	require.NoError(t, err)

	var r optypes.Receipt
	require.NoError(t, json.Unmarshal(wire, &r))
	require.Nil(t, r.L1GasPrice)
	require.Nil(t, r.L1Fee)
	require.Nil(t, r.FeeScalar)
	require.Nil(t, r.L1BaseFeeScalar)
	require.Nil(t, r.OperatorFeeScalar)
	require.Nil(t, r.DepositNonce)
}

func TestReceiptMarshalJSONRoundTrip(t *testing.T) {
	r := optypes.Receipt{
		Receipt:               *gethReceipt(),
		DepositNonce:          uint64Ptr(12),
		DepositReceiptVersion: uint64Ptr(1),
		L1GasPrice:            big.NewInt(1_000_000_007),
		L1BlobBaseFee:         big.NewInt(1),
		L1GasUsed:             big.NewInt(4096),
		L1Fee:                 big.NewInt(55_555),
		FeeScalar:             big.NewFloat(0.25),
		L1BaseFeeScalar:       uint64Ptr(11),
		L1BlobBaseFeeScalar:   uint64Ptr(0),
		OperatorFeeScalar:     uint64Ptr(123),
		OperatorFeeConstant:   uint64Ptr(456),
		DAFootprintGasScalar:  uint64Ptr(160),
	}

	wire, err := json.Marshal(&r)
	require.NoError(t, err)

	var decoded optypes.Receipt
	require.NoError(t, json.Unmarshal(wire, &decoded))

	require.Equal(t, r.DepositNonce, decoded.DepositNonce)
	require.Equal(t, r.DepositReceiptVersion, decoded.DepositReceiptVersion)
	require.Equal(t, r.L1GasPrice, decoded.L1GasPrice)
	require.Equal(t, r.L1BlobBaseFee, decoded.L1BlobBaseFee)
	require.Equal(t, r.L1GasUsed, decoded.L1GasUsed)
	require.Equal(t, r.L1Fee, decoded.L1Fee)
	require.NotNil(t, decoded.FeeScalar)
	require.Zero(t, r.FeeScalar.Cmp(decoded.FeeScalar))
	require.Equal(t, r.L1BaseFeeScalar, decoded.L1BaseFeeScalar)
	require.Equal(t, r.L1BlobBaseFeeScalar, decoded.L1BlobBaseFeeScalar)
	require.Equal(t, r.OperatorFeeScalar, decoded.OperatorFeeScalar)
	require.Equal(t, r.OperatorFeeConstant, decoded.OperatorFeeConstant)
	require.Equal(t, r.DAFootprintGasScalar, decoded.DAFootprintGasScalar)
	require.Equal(t, r.TxHash, decoded.TxHash)
	require.Equal(t, r.GasUsed, decoded.GasUsed)
}
