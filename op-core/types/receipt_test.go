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
