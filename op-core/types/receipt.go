package types

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Receipt extends go-ethereum's receipt with the OP Stack L1-cost and
// operator-fee fields that L2 endpoints include in eth_getTransactionReceipt
// responses. The JSON field names match op-geth's types.Receipt verbatim for
// wire compatibility; values are hex-encoded on the wire, hence the custom
// JSON methods.
type Receipt struct {
	types.Receipt
	L1GasPrice          *big.Int `json:"l1GasPrice,omitempty"`
	L1BlobBaseFee       *big.Int `json:"l1BlobBaseFee,omitempty"`
	L1BaseFeeScalar     *uint64  `json:"l1BaseFeeScalar,omitempty"`
	L1BlobBaseFeeScalar *uint64  `json:"l1BlobBaseFeeScalar,omitempty"`
	OperatorFeeScalar   *uint64  `json:"operatorFeeScalar,omitempty"`
	OperatorFeeConstant *uint64  `json:"operatorFeeConstant,omitempty"`
}

// receiptOpFields is the wire representation of the OP Stack receipt extensions.
type receiptOpFields struct {
	L1GasPrice          *hexutil.Big    `json:"l1GasPrice,omitempty"`
	L1BlobBaseFee       *hexutil.Big    `json:"l1BlobBaseFee,omitempty"`
	L1BaseFeeScalar     *hexutil.Uint64 `json:"l1BaseFeeScalar,omitempty"`
	L1BlobBaseFeeScalar *hexutil.Uint64 `json:"l1BlobBaseFeeScalar,omitempty"`
	OperatorFeeScalar   *hexutil.Uint64 `json:"operatorFeeScalar,omitempty"`
	OperatorFeeConstant *hexutil.Uint64 `json:"operatorFeeConstant,omitempty"`
}

// UnmarshalJSON decodes both the embedded go-ethereum receipt and the OP Stack
// extension fields from the same JSON object.
func (r *Receipt) UnmarshalJSON(input []byte) error {
	if err := json.Unmarshal(input, &r.Receipt); err != nil {
		return err
	}
	var dec receiptOpFields
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	r.L1GasPrice = (*big.Int)(dec.L1GasPrice)
	r.L1BlobBaseFee = (*big.Int)(dec.L1BlobBaseFee)
	r.L1BaseFeeScalar = (*uint64)(dec.L1BaseFeeScalar)
	r.L1BlobBaseFeeScalar = (*uint64)(dec.L1BlobBaseFeeScalar)
	r.OperatorFeeScalar = (*uint64)(dec.OperatorFeeScalar)
	r.OperatorFeeConstant = (*uint64)(dec.OperatorFeeConstant)
	return nil
}

// MarshalJSON encodes the embedded go-ethereum receipt and merges the OP Stack
// extension fields into the same JSON object, so the encoding round-trips.
func (r Receipt) MarshalJSON() ([]byte, error) {
	base, err := json.Marshal(&r.Receipt)
	if err != nil {
		return nil, err
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(base, &obj); err != nil {
		return nil, err
	}
	extra, err := json.Marshal(&receiptOpFields{
		L1GasPrice:          (*hexutil.Big)(r.L1GasPrice),
		L1BlobBaseFee:       (*hexutil.Big)(r.L1BlobBaseFee),
		L1BaseFeeScalar:     (*hexutil.Uint64)(r.L1BaseFeeScalar),
		L1BlobBaseFeeScalar: (*hexutil.Uint64)(r.L1BlobBaseFeeScalar),
		OperatorFeeScalar:   (*hexutil.Uint64)(r.OperatorFeeScalar),
		OperatorFeeConstant: (*hexutil.Uint64)(r.OperatorFeeConstant),
	})
	if err != nil {
		return nil, err
	}
	var extraObj map[string]json.RawMessage
	if err := json.Unmarshal(extra, &extraObj); err != nil {
		return nil, err
	}
	for k, v := range extraObj {
		obj[k] = v
	}
	return json.Marshal(obj)
}
