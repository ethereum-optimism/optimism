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

// unstringifyArg converts a string to a Go type.
func unstringifyArg(arg string, typ string) (any, error) {
	switch typ {
	case "address":
		return common.HexToAddress(arg), nil
	case "bool":
		return strconv.ParseBool(arg)
	case "uint8":
		val, err := strconv.ParseUint(arg, 10, 8)
		return uint8(val), err
	case "uint16":
		val, err := strconv.ParseUint(arg, 10, 16)
		return uint16(val), err
	case "uint32":
		val, err := strconv.ParseUint(arg, 10, 32)
		return uint32(val), err
	case "uint64":
		val, err := strconv.ParseUint(arg, 10, 64)
		return val, err
	case "int8":
		val, err := strconv.ParseInt(arg, 10, 8)
		return val, err
	case "int16":
		val, err := strconv.ParseInt(arg, 10, 16)
		return val, err
	case "int32":
		val, err := strconv.ParseInt(arg, 10, 32)
		return val, err
	case "int64":
		val, err := strconv.ParseInt(arg, 10, 64)
		return val, err
	case "uint256", "int256":
		val, ok := new(big.Int).SetString(arg, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse %s as big.Int", arg)
		}
		return val, nil
	case "string":
		return arg, nil
	case "bytes":
		return hexutil.Decode(arg)
	default:
		return nil, fmt.Errorf("unknown type: %s", typ)
	}
}

// createContractInput converts an abi.Argument to one or more ContractInputs.
func createContractInput(input abi.Argument, inputs []ContractInput) ([]ContractInput, error) {
	inputType, err := stringifyType(input.Type)
	if err != nil {
		return nil, err
	}

	internalType := input.Type.String()
	if input.Type.T == abi.TupleTy {
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
