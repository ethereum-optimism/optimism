// Package safe contains types for working with Safe smart contract wallets. These are used to
// build batch transactions for the tx-builder app. The types are based on
// https://github.com/safe-global/safe-react-apps/blob/development/apps/tx-builder/src/typings/models.ts.
package safe

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BatchFile represents a Safe tx-builder transaction.
type BatchFile struct {
	Version      string             `json:"version"`
	ChainID      *big.Int           `json:"chainId"`
	CreatedAt    uint64             `json:"createdAt"`
	Meta         BatchFileMeta      `json:"meta"`
	Transactions []BatchTransaction `json:"transactions"`
}

// bathcFileMarshaling is a helper type used for JSON marshaling.
type batchFileMarshaling struct {
	Version      string             `json:"version"`
	ChainID      string             `json:"chainId"`
	CreatedAt    uint64             `json:"createdAt"`
	Meta         BatchFileMeta      `json:"meta"`
	Transactions []BatchTransaction `json:"transactions"`
}

// MarshalJSON will marshal a BatchFile to JSON.
func (b *BatchFile) MarshalJSON() ([]byte, error) {
	return json.Marshal(batchFileMarshaling{
		Version:      b.Version,
		ChainID:      b.ChainID.String(),
		CreatedAt:    b.CreatedAt,
		Meta:         b.Meta,
		Transactions: b.Transactions,
	})
}

// UnmarshalJSON will unmarshal a BatchFile from JSON.
func (b *BatchFile) UnmarshalJSON(data []byte) error {
	var bf batchFileMarshaling
	if err := json.Unmarshal(data, &bf); err != nil {
		return err
	}
	b.Version = bf.Version
	chainId, ok := new(big.Int).SetString(bf.ChainID, 10)
	if !ok {
		return fmt.Errorf("cannot set chainId to %s", bf.ChainID)
	}
	b.ChainID = chainId
	b.CreatedAt = bf.CreatedAt
	b.Meta = bf.Meta
	b.Transactions = bf.Transactions
	return nil
}

// BatchFileMeta contains metadata about a BatchFile. Not all
// of the fields are required.
type BatchFileMeta struct {
	TxBuilderVersion        string `json:"txBuilderVersion,omitempty"`
	Checksum                string `json:"checksum,omitempty"`
	CreatedFromSafeAddress  string `json:"createdFromSafeAddress"`
	CreatedFromOwnerAddress string `json:"createdFromOwnerAddress"`
	Name                    string `json:"name"`
	Description             string `json:"description"`
}

// BatchTransaction represents a single call in a tx-builder transaction.
type BatchTransaction struct {
	To          common.Address    `json:"to"`
	Value       *big.Int          `json:"value"`
	Data        []byte            `json:"data"`
	Method      ContractMethod    `json:"contractMethod"`
	InputValues map[string]string `json:"contractInputsValues"`
}

// UnmarshalJSON will unmarshal a BatchTransaction from JSON.
func (b *BatchTransaction) UnmarshalJSON(data []byte) error {
	var bt batchTransactionMarshaling
	if err := json.Unmarshal(data, &bt); err != nil {
		return err
	}
	b.To = common.HexToAddress(bt.To)
	b.Value = new(big.Int).SetUint64(bt.Value)
	b.Method = bt.Method
	b.InputValues = bt.InputValues
	return nil
}

// MarshalJSON will marshal a BatchTransaction to JSON.
func (b *BatchTransaction) MarshalJSON() ([]byte, error) {
	batch := batchTransactionMarshaling{
		To:          b.To.Hex(),
		Value:       b.Value.Uint64(),
		Method:      b.Method,
		InputValues: b.InputValues,
	}
	if len(b.Data) != 0 {
		hex := hexutil.Encode(b.Data)
		batch.Data = &hex
	}
	return json.Marshal(batch)
}

// batchTransactionMarshaling is a helper type used for JSON marshaling.
type batchTransactionMarshaling struct {
	To          string            `json:"to"`
	Value       uint64            `json:"value,string"`
	Data        *string           `json:"data"`
	Method      ContractMethod    `json:"contractMethod"`
	InputValues map[string]string `json:"contractInputsValues"`
}

// ContractMethod represents a method call in a tx-builder transaction.
type ContractMethod struct {
	Inputs  []ContractInput `json:"inputs"`
	Name    string          `json:"name"`
	Payable bool            `json:"payable"`
}

// ContractInput represents an input to a contract method.
type ContractInput struct {
	InternalType string          `json:"internalType"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	Components   []ContractInput `json:"components,omitempty"`
}
