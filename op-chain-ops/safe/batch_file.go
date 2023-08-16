// Package safe contains types for working with Safe smart contract wallets. These are used to
// build batch transactions for the tx-builder app. The types are based on
// https://github.com/safe-global/safe-react-apps/blob/development/apps/tx-builder/src/typings/models.ts.
package safe

import (
	"encoding/json"
	"fmt"
	"golang.org/x/exp/maps"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Batch represents a Safe tx-builder transaction.
type Batch struct {
	Version      string             `json:"version"`
	ChainID      *big.Int           `json:"chainId"`
	CreatedAt    uint64             `json:"createdAt"`
	Meta         BatchMeta          `json:"meta"`
	Transactions []BatchTransaction `json:"transactions"`
}

func (b *Batch) AddCall(to common.Address, value *big.Int, sig string, args []any, iface abi.ABI) error {
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

	// TODO: refactor this to be recursive
	// it should return a ContractInput and take an Input in
	for i, input := range method.Inputs {
		inputType, err := stringifyType(input.Type)
		if err != nil {
			return err
		}

		internalType := input.Type.String()
		if inputType == "tuple" {
			internalType = input.Type.TupleRawName
		}

		components := make([]ContractInput, len(input.Type.TupleElems))
		for j, elem := range input.Type.TupleElems {
			str := input.Type.TupleRawNames[j]
			components[j] = ContractInput{
				InternalType: elem.String(),
				Name:         str,
				Type:         elem.String(),
			}
		}

		contractInput := ContractInput{
			InternalType: internalType,
			Name:         input.Name,
			Type:         inputType,
			Components:   components,
		}
		contractMethod.Inputs = append(contractMethod.Inputs, contractInput)

		str, err := stringifyArg(args[i])
		if err != nil {
			return err
		}
		inputValues[input.Name] = str
	}

	batchTransaction := BatchTransaction{
		To:          to,
		Value:       value,
		Method:      contractMethod,
		InputValues: inputValues,
	}

	b.Transactions = append(b.Transactions, batchTransaction)

	return nil
}

func stringifyType(t abi.Type) (string, error) {
	switch t.T {
	case abi.TupleTy:
		return "tuple", nil
	case abi.BoolTy:
		return "bool", nil
	case abi.AddressTy:
		return "address", nil
	case abi.UintTy:
		return t.String(), nil
	case abi.IntTy:
		return t.String(), nil
	default:
		return "", fmt.Errorf("unknown type: %T", t.T)
	}
}

func stringifyArg(argument any) (string, error) {
	switch arg := argument.(type) {
	case common.Address:
		return arg.String(), nil
	case *common.Address:
		return arg.String(), nil
	case *big.Int:
		return arg.String(), nil
	case big.Int:
		return arg.String(), nil
	case bool:
		if arg {
			return "true", nil
		}
		return "false", nil
	case int64:
		return strconv.FormatInt(arg, 10), nil
	case int32:
		return strconv.FormatInt(int64(arg), 10), nil
	case int16:
		return strconv.FormatInt(int64(arg), 10), nil
	case int8:
		return strconv.FormatInt(int64(arg), 10), nil
	case int:
		return strconv.FormatInt(int64(arg), 10), nil
	case uint64:
		return strconv.FormatUint(uint64(arg), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(arg), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(arg), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(arg), 10), nil
	case uint:
		return strconv.FormatUint(uint64(arg), 10), nil
	case []byte:
		return hexutil.Encode(arg), nil
	case []any:
		ret := make([]string, len(arg))
		for i, v := range arg {
			str, err := stringifyArg(v)
			if err != nil {
				return "", err
			}
			ret[i] = str
		}
		return "[" + strings.Join(ret, ",") + "]", nil
	default:
		return "", fmt.Errorf("unknown type as argument: %T", arg)
	}
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
