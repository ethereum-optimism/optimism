package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Transaction struct {
	To             string  `json:"to"`
	Value          string  `json:"value"`
	Data           *string `json:"data"`
	ContractMethod struct {
		Inputs []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"inputs"`
		Name string `json:"name"`
	} `json:"contractMethod"`
	ContractInputsValues map[string]string `json:"contractInputsValues"`
}

type Bundle struct {
	Transactions []Transaction `json:"transactions"`
}

// EncodedTransaction represents the ABI-encoded transaction
type EncodedTransaction struct {
	Target   common.Address
	Value    *big.Int
	Calldata []byte
}

func EncodeBundleTransactions() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go EncodeBundleTransactions <path_to_json_file>")
		os.Exit(1)
	}

	jsonFilePath := os.Args[2]

	// Read the JSON file
	jsonData, err := os.ReadFile(jsonFilePath)
	if err != nil {
		fmt.Printf("Error reading JSON file: %v\n", err)
		os.Exit(1)
	}

	// Parse the JSON data
	var bundle Bundle
	err = json.Unmarshal(jsonData, &bundle)
	if err != nil {
		fmt.Printf("Error parsing JSON data: %v\n", err)
		os.Exit(1)
	}

	// Encode the transactions
	encodedTransactions := make([]EncodedTransaction, len(bundle.Transactions))
	for i, tx := range bundle.Transactions {
		target := common.HexToAddress(tx.To)
		value, _ := new(big.Int).SetString(tx.Value, 10)

		if tx.Data != nil {
			fmt.Println("Data MUST be null")
			os.Exit(1)
		}

		calldata, err := encodeMethodCall(tx.ContractMethod, tx.ContractInputsValues)
		if err != nil {
			fmt.Printf("Error encoding method call: %v\n", err)
			os.Exit(1)
		}
		encodedTransactions[i] = EncodedTransaction{
			Target:   target,
			Value:    value,
			Calldata: calldata,
		}
	}

	// Define the array type for EncodedTransaction
	arrayType, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "target", Type: "address"},
		{Name: "value", Type: "uint256"},
		{Name: "calldata", Type: "bytes"},
	})
	if err != nil {
		fmt.Printf("Error creating array type: %v\n", err)
		os.Exit(1)
	}

	// ABI encode the array of encoded transactions
	encodedData, err := abi.Arguments{{Type: arrayType}}.Pack(encodedTransactions)
	if err != nil {
		fmt.Printf("Error encoding transactions: %v\n", err)
		os.Exit(1)
	}

	// Print the encoded data
	fmt.Print(hexutil.Encode(encodedData))
}

func encodeMethodCall(method struct {
	Inputs []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"inputs"`
	Name string `json:"name"`
}, inputValues map[string]string) ([]byte, error) {
	// Create the inputs for NewMethod
	inputs := make(abi.Arguments, len(method.Inputs))
	for i, input := range method.Inputs {
		argType, err := abi.NewType(input.Type, "", nil)
		if err != nil {
			return nil, fmt.Errorf("error creating ABI type for %s: %v", input.Type, err)
		}
		inputs[i] = abi.Argument{Name: input.Name, Type: argType}
	}

	// Create the Method using NewMethod
	abiMethod := abi.NewMethod(
		method.Name,     // name
		method.Name,     // rawName
		abi.Function,    // funType
		"",              // mutability (default to empty string)
		false,           // isConst
		false,           // isPayable
		inputs,          // inputs
		abi.Arguments{}, // outputs (empty for this case)
	)

	// Create the ABI definition with the new method
	abiDef := abi.ABI{
		Methods: map[string]abi.Method{
			method.Name: abiMethod,
		},
	}

	// Prepare the input values
	args := make([]interface{}, len(method.Inputs))
	for i, input := range method.Inputs {
		convertedValue, err := convertValue(input.Type, inputValues[input.Name])
		if err != nil {
			return nil, err
		}
		args[i] = convertedValue
	}

	// Encode the method call
	encodedData, err := abiDef.Pack(method.Name, args...)
	if err != nil {
		return nil, fmt.Errorf("error encoding method call: %v", err)
	}

	return encodedData, nil
}

func convertValue(typ string, value string) (interface{}, error) {
	switch typ {
	case "address":
		return common.HexToAddress(value), nil
	case "uint256":
		bigInt, _ := new(big.Int).SetString(value, 10)
		return bigInt, nil
	case "uint32":
		intValue, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse uint32: %v", err)
		}
		return uint32(intValue), nil
	case "bytes32":
		return common.HexToHash(value), nil
	case "bytes":
		return common.FromHex(value), nil
	default:
		return nil, errors.New("unsupported type")
	}
}
