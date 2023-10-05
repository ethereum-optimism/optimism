// Package safe contains types for working with Safe smart contract wallets. These are used to
// build batch transactions for the tx-builder app. The types are based on
// https://github.com/safe-global/safe-react-apps/blob/development/apps/tx-builder/src/typings/models.ts.
package safe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// Batch represents a Safe tx-builder transaction.
// SkipCalldata will skip adding the calldata to the BatchTransaction.
// This is useful for when using the Safe UI because it prefers using
// the raw calldata when both the calldata and ABIs with arguments are
// present.
type Batch struct {
	SkipCalldata bool               `json:"-"`
	Version      string             `json:"version"`
	ChainID      *big.Int           `json:"chainId"`
	CreatedAt    uint64             `json:"createdAt"`
	Meta         BatchMeta          `json:"meta"`
	Transactions []BatchTransaction `json:"transactions"`
}

// AddCall will add a call to the batch. After a series of calls are
// added to the batch, it can be serialized to JSON.
func (b *Batch) AddCall(to common.Address, value *big.Int, sig string, args []any, iface *abi.ABI) error {
	if iface == nil {
		return errors.New("abi cannot be nil")
	}
	// Attempt to pull out the signature from the top level methods.
	// The abi package uses normalization that we do not want to be
	// coupled to, so attempt to search for the raw name if the top
	// level name is not found to handle overloading more gracefully.
	method, ok := iface.Methods[sig]
	if !ok {
		for _, m := range iface.Methods {
			if m.RawName == sig || m.Sig == sig {
				method = m
				ok = true
			}
		}
	}
	if !ok {
		keys := maps.Keys(iface.Methods)
		methods := strings.Join(keys, ",")
		return fmt.Errorf("%s not found in abi, options are %s", sig, methods)
	}

	if len(args) != len(method.Inputs) {
		return fmt.Errorf("requires %d inputs but got %d for %s", len(method.Inputs), len(args), method.RawName)
	}

	contractMethod := ContractMethod{
		Name:    method.RawName,
		Payable: method.Payable,
	}

	inputValues := make(map[string]string)
	contractInputs := make([]ContractInput, 0)

	for i, input := range method.Inputs {
		contractInput, err := createContractInput(input, contractInputs)
		if err != nil {
			return err
		}
		contractMethod.Inputs = append(contractMethod.Inputs, contractInput...)

		str, err := stringifyArg(args[i])
		if err != nil {
			return err
		}
		inputValues[input.Name] = str
	}

	encoded, err := method.Inputs.PackValues(args)
	if err != nil {
		return err
	}
	data := make([]byte, len(method.ID)+len(encoded))
	copy(data, method.ID)
	copy(data[len(method.ID):], encoded)

	batchTransaction := BatchTransaction{
		To:          to,
		Value:       value,
		Method:      contractMethod,
		InputValues: inputValues,
	}

	if !b.SkipCalldata {
		batchTransaction.Data = data
	}

	b.Transactions = append(b.Transactions, batchTransaction)

	return nil
}

// Check will check the batch for errors
func (b *Batch) Check() error {
	for _, tx := range b.Transactions {
		if err := tx.Check(); err != nil {
			return err
		}
	}
	return nil
}

// bathcFileMarshaling is a helper type used for JSON marshaling.
type batchMarshaling struct {
	Version      string             `json:"version"`
	ChainID      string             `json:"chainId"`
	CreatedAt    uint64             `json:"createdAt"`
	Meta         BatchMeta          `json:"meta"`
	Transactions []BatchTransaction `json:"transactions"`
}

// MarshalJSON will marshal a Batch to JSON.
func (b *Batch) MarshalJSON() ([]byte, error) {
	batch := batchMarshaling{
		Version:      b.Version,
		CreatedAt:    b.CreatedAt,
		Meta:         b.Meta,
		Transactions: b.Transactions,
	}
	if b.ChainID != nil {
		batch.ChainID = b.ChainID.String()
	}
	return json.Marshal(batch)
}

// UnmarshalJSON will unmarshal a Batch from JSON.
func (b *Batch) UnmarshalJSON(data []byte) error {
	var bf batchMarshaling
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

// BatchMeta contains metadata about a Batch. Not all
// of the fields are required.
type BatchMeta struct {
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

// Check will check the batch transaction for errors.
// An error is defined by:
// - incorrectly encoded calldata
// - mismatch in number of arguments
// It does not currently work on structs, will return no error if a "tuple"
// is used as an argument. Need to find a generic way to work with structs.
func (bt *BatchTransaction) Check() error {
	if len(bt.Method.Inputs) != len(bt.InputValues) {
		return fmt.Errorf("expected %d inputs but got %d", len(bt.Method.Inputs), len(bt.InputValues))
	}

	if len(bt.Data) > 0 && bt.Method.Name != "fallback" {
		if len(bt.Data) < 4 {
			return fmt.Errorf("must have at least 4 bytes of calldata, got %d", len(bt.Data))
		}
		sig := bt.Signature()
		selector := crypto.Keccak256([]byte(sig))[0:4]
		if !bytes.Equal(bt.Data[0:4], selector) {
			return fmt.Errorf("data does not match signature")
		}

		// Check the calldata
		values := make([]any, len(bt.Method.Inputs))
		for i, input := range bt.Method.Inputs {
			value, ok := bt.InputValues[input.Name]
			if !ok {
				return fmt.Errorf("missing input %s", input.Name)
			}
			// Need to figure out better way to handle tuples in a generic way
			if input.Type == "tuple" {
				return nil
			}
			arg, err := unstringifyArg(value, input.Type)
			if err != nil {
				return err
			}
			values[i] = arg
		}

		calldata, err := bt.Arguments().PackValues(values)
		if err != nil {
			return err
		}
		if !bytes.Equal(bt.Data[4:], calldata) {
			return fmt.Errorf("calldata does not match inputs, expected %s, got %s", hexutil.Encode(bt.Data[4:]), hexutil.Encode(calldata))
		}
	}
	return nil
}

// Signature returns the function signature of the batch transaction.
func (bt *BatchTransaction) Signature() string {
	types := make([]string, len(bt.Method.Inputs))
	for i, input := range bt.Method.Inputs {
		types[i] = buildFunctionSignature(input)
	}
	return fmt.Sprintf("%s(%s)", bt.Method.Name, strings.Join(types, ","))
}

func (bt *BatchTransaction) Arguments() abi.Arguments {
	arguments := make(abi.Arguments, len(bt.Method.Inputs))
	for i, input := range bt.Method.Inputs {
		serialized, err := json.Marshal(input)
		if err != nil {
			panic(err)
		}
		var arg abi.Argument
		if err := json.Unmarshal(serialized, &arg); err != nil {
			panic(err)
		}
		arguments[i] = arg
	}
	return arguments
}

// UnmarshalJSON will unmarshal a BatchTransaction from JSON.
func (b *BatchTransaction) UnmarshalJSON(data []byte) error {
	var bt batchTransactionMarshaling
	if err := json.Unmarshal(data, &bt); err != nil {
		return err
	}
	b.To = common.HexToAddress(bt.To)
	b.Value = new(big.Int).SetUint64(bt.Value)
	if bt.Data != nil {
		b.Data = common.CopyBytes(*bt.Data)
	}
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
		data := hexutil.Bytes(b.Data)
		batch.Data = &data
	}
	return json.Marshal(batch)
}

// batchTransactionMarshaling is a helper type used for JSON marshaling.
type batchTransactionMarshaling struct {
	To          string            `json:"to"`
	Value       uint64            `json:"value,string"`
	Data        *hexutil.Bytes    `json:"data"`
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
