package bindings

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/holiman/uint256"
)

type ABIInt256 big.Int

var abiInt256Type = reflect.TypeFor[ABIInt256]()

var abiUint256Type = reflect.TypeFor[uint256.Int]()

func goStructTypeToABIType(t reflect.Type) (abi.Type, []abi.ArgumentMarshaling, error) {
	if t.Kind() != reflect.Struct {
		return abi.Type{}, nil, errors.New("input must be a struct type")
	}
	components := []abi.ArgumentMarshaling{}
	for i := range t.NumField() {
		field := t.Field(i)
		fieldTyp := field.Type
		fieldName := field.Name
		innerABIType, innerComponents, err := goTypeToABIType(fieldTyp)
		if err != nil {
			return abi.Type{}, nil, fmt.Errorf("field %s: %w", fieldName, err)
		}
		innerElemTyp := innerABIType.String()
		switch innerABIType.T {
		case abi.TupleTy:
			innerElemTyp = "tuple"
		case abi.ArrayTy:
			if innerComponents != nil {
				idx := strings.LastIndex(innerElemTyp, "[")
				if idx == -1 {
					panic(fmt.Sprintf("malformed type: %s", innerElemTyp))
				}
				innerElemTyp = "tuple" + innerElemTyp[idx:]
			}
		case abi.SliceTy:
			innerElemTyp = "tuple[]"
		}
		components = append(components, abi.ArgumentMarshaling{
			Name: fieldName, Type: innerElemTyp, Components: innerComponents,
		})
	}
	tuple, err := abi.NewType("tuple", "", components)
	if err != nil {
		return abi.Type{}, nil, fmt.Errorf("failed to construct tuple: %w", err)
	}
	return tuple, components, nil
}

func goTypeToABIType(typ reflect.Type) (abi.Type, []abi.ArgumentMarshaling, error) {
	switch typ.Kind() {
	case reflect.Int, reflect.Uint:
		return abi.Type{}, nil, fmt.Errorf("ints must have explicit size, type not valid: %s", typ)
	case reflect.Bool, reflect.String, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		abiType, err := abi.NewType(strings.ToLower(typ.Kind().String()), "", nil)
		return abiType, nil, err
	case reflect.Array:
		if typ.AssignableTo(abiUint256Type) {
			abiType, err := abi.NewType("uint256", "", nil)
			return abiType, nil, err
		}
		if typ.Elem().Kind() == reflect.Uint8 {
			if typ.Len() == 20 && typ.Name() == "Address" {
				abiType, err := abi.NewType("address", "", nil)
				return abiType, nil, err
			}
			if typ.Len() > 32 {
				return abi.Type{}, nil, fmt.Errorf("byte array too large: %d", typ.Len())
			}
			abiType, err := abi.NewType(fmt.Sprintf("bytes%d", typ.Len()), "", nil)
			return abiType, nil, err
		}
		innerABIType, innerComponents, err := goTypeToABIType(typ.Elem())
		if err != nil {
			return abi.Type{}, nil, fmt.Errorf("unrecognized slice-elem type: %w", err)
		}
		elemType := innerABIType.String()
		if innerABIType.TupleType != nil {
			elemType = "tuple"
		}
		abiType, err := abi.NewType(fmt.Sprintf("%s[%d]", elemType, typ.Len()), "", innerComponents)
		return abiType, innerComponents, err
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			abiType, err := abi.NewType("bytes", "", nil)
			return abiType, nil, err
		}
		innerABIType, innerComponents, err := goTypeToABIType(typ.Elem())
		if err != nil {
			return abi.Type{}, nil, fmt.Errorf("unrecognized slice-elem type: %w", err)
		}
		elemType := innerABIType.String()
		if innerABIType.TupleType != nil {
			elemType = "tuple"
		}
		abiType, err := abi.NewType(fmt.Sprintf("%s[]", elemType), "", innerComponents)
		return abiType, innerComponents, err
	case reflect.Struct:
		if typ.AssignableTo(abiInt256Type) {
			abiType, err := abi.NewType("int256", "", nil)
			return abiType, nil, err
		}
		if typ.ConvertibleTo(reflect.TypeFor[big.Int]()) {
			abiType, err := abi.NewType("uint256", "", nil)
			return abiType, nil, err
		}
		abiType, components, err := goStructTypeToABIType(typ)
		if err != nil {
			return abi.Type{}, nil, fmt.Errorf("struct conversion failure: type %s: %w", typ, err)
		}
		return abiType, components, nil
	case reflect.Pointer:
		abiType, components, err := goTypeToABIType(typ.Elem())
		if err != nil {
			return abi.Type{}, nil, fmt.Errorf("unrecognized pointer-elem type: %w", err)
		}
		return abiType, components, nil
	default:
		return abi.Type{}, nil, fmt.Errorf("unrecognized typ: %s", typ)
	}
}
