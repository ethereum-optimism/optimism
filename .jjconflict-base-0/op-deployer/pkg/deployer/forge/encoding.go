package forge

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func GoStructToABITuple(structType reflect.Type, tupleName string) (abi.Type, error) {
	var components []abi.ArgumentMarshaling

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		abiType, err := GoTypeToABIType(field.Type)
		if err != nil {
			return abi.Type{}, fmt.Errorf("unsupported field type %s: %w", field.Type, err)
		}

		components = append(components, abi.ArgumentMarshaling{
			Name: field.Name,
			Type: abiType,
		})
	}

	return abi.NewType("tuple", tupleName, components)
}

func GoTypeToABIType(goType reflect.Type) (string, error) {
	// handle pointers by dereferencing
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	// non-standard go types (need to catch these first)
	switch goType {
	case reflect.TypeOf(common.Address{}):
		return "address", nil
	case reflect.TypeOf(common.Hash{}):
		return "bytes32", nil
	case reflect.TypeOf(params.ProtocolVersion{}):
		return "bytes32", nil
	case reflect.TypeOf(big.NewInt(0)).Elem():
		return "uint256", nil
	}

	// standard go types
	switch goType.Kind() {
	case reflect.Slice:
		elemType := goType.Elem()

		// special case: []byte -> "bytes"
		if elemType.Kind() == reflect.Uint8 {
			return "bytes", nil
		}

		// recursive: []T -> "T[]"
		elemABI, err := GoTypeToABIType(elemType)
		if err != nil {
			return "", fmt.Errorf("unsupported slice element type: %w", err)
		}
		return elemABI + "[]", nil

	case reflect.Array:
		elemType := goType.Elem()
		arrayLen := goType.Len()

		// recursive: [N]T -> "T[N]"
		elemABI, err := GoTypeToABIType(elemType)
		if err != nil {
			return "", fmt.Errorf("unsupported array element type: %w", err)
		}
		return fmt.Sprintf("%s[%d]", elemABI, arrayLen), nil

	case reflect.String:
		return "string", nil
	case reflect.Bool:
		return "bool", nil
	case reflect.Uint8:
		return "uint8", nil
	case reflect.Uint16:
		return "uint16", nil
	case reflect.Uint32:
		return "uint32", nil
	case reflect.Uint64, reflect.Uint:
		return "uint64", nil
	case reflect.Int8:
		return "int8", nil
	case reflect.Int16:
		return "int16", nil
	case reflect.Int32:
		return "int32", nil
	case reflect.Int64, reflect.Int:
		return "int64", nil
	}

	return "", fmt.Errorf("unable to convert go type to abi type: %s", goType)
}

func ConvertAnonStructToTyped[T any](anonStruct interface{}) (T, error) {
	var result T

	srcVal := reflect.ValueOf(anonStruct)
	destVal := reflect.ValueOf(&result).Elem()

	// Ensure both are structs
	if srcVal.Kind() != reflect.Struct || destVal.Kind() != reflect.Struct {
		return result, fmt.Errorf("both source and destination must be structs")
	}

	// Check field count matches
	if srcVal.NumField() != destVal.NumField() {
		return result, fmt.Errorf("field count mismatch: source has %d, destination has %d",
			srcVal.NumField(), destVal.NumField())
	}

	// Copy fields by index (assumes same field order)
	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		destField := destVal.Field(i)

		if destField.CanSet() {
			destField.Set(srcField)
		}
	}

	return result, nil
}

type BytesScriptEncoder[T any] struct {
	TypeName string // e.g., "DeploySuperchainInput"
}

func (e *BytesScriptEncoder[T]) Encode(input T) ([]byte, error) {
	inputType, err := GoStructToABITuple(reflect.TypeOf(input), e.TypeName)
	if err != nil {
		return nil, fmt.Errorf("failed to create input type: %w", err)
	}

	args := abi.Arguments{{Type: inputType}}
	return args.Pack(input)
}

type BytesScriptDecoder[T any] struct {
	TypeName string // e.g., "DeploySuperchainOutput"
}

func (d *BytesScriptDecoder[T]) Decode(rawOutput []byte) (T, error) {
	var zero T
	outputType, err := GoStructToABITuple(reflect.TypeOf(zero), d.TypeName)
	if err != nil {
		return zero, fmt.Errorf("failed to create output type: %w", err)
	}

	args := abi.Arguments{{Type: outputType}}
	unpacked, err := args.Unpack(rawOutput)
	if err != nil {
		return zero, fmt.Errorf("failed to unpack output: %w", err)
	}

	if len(unpacked) != 1 {
		return zero, fmt.Errorf("expected 1 unpacked value, got %d", len(unpacked))
	}

	return ConvertAnonStructToTyped[T](unpacked[0])
}
