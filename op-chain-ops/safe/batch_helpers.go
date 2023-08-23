package safe

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// stringifyArg converts a Go type to a string that is representable by ABI.
// To do so, this function must be recursive to handle nested tuples.
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
		typ := reflect.TypeOf(argument)
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}

		if typ.Kind() == reflect.Struct {
			v := reflect.ValueOf(argument)
			numField := v.NumField()
			ret := make([]string, numField)

			for i := 0; i < numField; i++ {
				val := v.Field(i).Interface()
				str, err := stringifyArg(val)
				if err != nil {
					return "", err
				}
				ret[i] = str
			}

			return "[" + strings.Join(ret, ",") + "]", nil
		}

		return "", fmt.Errorf("unknown type as argument: %T", arg)
	}
}

// countArgs will recursively count the number of arguments in an abi.Argument.
func countArgs(total *int, input abi.Argument) error {
	for i, elem := range input.Type.TupleElems {
		e := *elem
		*total++
		arg := abi.Argument{
			Name: input.Type.TupleRawNames[i],
			Type: e,
		}
		return countArgs(total, arg)
	}
	return nil
}

// createContractInput converts an abi.Argument to one or more ContractInputs.
func createContractInput(input abi.Argument, inputs []ContractInput) ([]ContractInput, error) {
	inputType, err := stringifyType(input.Type)
	if err != nil {
		return nil, err
	}

	// TODO: could probably do better than string comparison?
	internalType := input.Type.String()
	if inputType == "tuple" {
		internalType = input.Type.TupleRawName
	}

	components := make([]ContractInput, 0)
	for i, elem := range input.Type.TupleElems {
		e := *elem
		arg := abi.Argument{
			Name: input.Type.TupleRawNames[i],
			Type: e,
		}
		component, err := createContractInput(arg, inputs)
		if err != nil {
			return nil, err
		}
		components = append(components, component...)
	}

	contractInput := ContractInput{
		InternalType: internalType,
		Name:         input.Name,
		Type:         inputType,
		Components:   components,
	}

	inputs = append(inputs, contractInput)

	return inputs, nil
}

// stringifyType turns an abi.Type into a string
func stringifyType(t abi.Type) (string, error) {
	switch t.T {
	case abi.TupleTy:
		return "tuple", nil
	case abi.BoolTy:
		return t.String(), nil
	case abi.AddressTy:
		return t.String(), nil
	case abi.UintTy:
		return t.String(), nil
	case abi.IntTy:
		return t.String(), nil
	case abi.StringTy:
		return t.String(), nil
	case abi.BytesTy:
		return t.String(), nil
	default:
		return "", fmt.Errorf("unknown type: %d", t.T)
	}
}

// buildFunctionSignature builds a function signature from a ContractInput.
// It is recursive to handle tuples.
func buildFunctionSignature(input ContractInput) string {
	if input.Type == "tuple" {
		types := make([]string, len(input.Components))
		for i, component := range input.Components {
			types[i] = buildFunctionSignature(component)
		}
		return fmt.Sprintf("(%s)", strings.Join(types, ","))
	}
	return input.InternalType
}
