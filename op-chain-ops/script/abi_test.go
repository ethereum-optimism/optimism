package script

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func die[O any](value O, err error) O {
	if err != nil {
		panic(err)
	}

	return value
}

func TestMatchTypes(t *testing.T) {
	type matchTypesTest struct {
		abiType abi.Type
		goType  reflect.Type
		err     string
	}

	type StructWithPrimitiveFields struct {
		AddressField common.Address
		BoolField    bool
		UintField    *big.Int
	}

	type StructWithPrimitiveFieldsWrapper struct {
		Nested StructWithPrimitiveFields
	}

	structWithPrimitiveFieldsMarshalling := []abi.ArgumentMarshaling{{Name: "addressField", Type: "address"}, {Name: "boolField", Type: "bool"}, {Name: "uintField", Type: "uint256"}}

	matchTypesTests := []matchTypesTest{
		{
			abiType: die(abi.NewType("uint256", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(new(big.Int)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("uint128", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(new(big.Int)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("uint64", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new(uint64)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("uint8", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new(uint8)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("string", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new(string)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("bool", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new(bool)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("bytes", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new([]byte)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("bytes", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new([32]byte)),
			err:     `ABI type bytes (represented by []uint8) is not assignable to Go type [32]uint8`,
		},
		{
			abiType: die(abi.NewType("bytes32", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new([32]byte)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("bytes32", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new([]byte)),
			err:     `ABI type bytes32 (represented by [32]uint8) is not assignable to Go type []uint8`,
		},
		{
			abiType: die(abi.NewType("bytes32", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new([64]byte)),
			err:     `ABI type bytes32 (represented by [32]uint8) is not assignable to Go type [64]uint8`,
		},
		{
			abiType: die(abi.NewType("address", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new(common.Address)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("address", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new([]byte)),
			err:     `ABI type address (represented by common.Address) is not assignable to Go type []uint8`,
		},
		{
			abiType: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new(struct{})),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("tuple", "", structWithPrimitiveFieldsMarshalling)),
			goType:  reflect.TypeOf(*new(StructWithPrimitiveFields)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "boolField", Type: "bool"}, {Name: "addressField", Type: "address"}, {Name: "uintField", Type: "uint256"}})),
			goType:  reflect.TypeOf(*new(StructWithPrimitiveFields)),
			err:     `ABI type (bool,address,uint256) (represented by struct { BoolField bool "json:\"boolField\""; AddressField common.Address "json:\"addressField\""; UintField *big.Int "json:\"uintField\"" }) is not assignable to Go type script.StructWithPrimitiveFields: ABI field name BoolField at index 0 does not match Go field name AddressField. Please make sure to match the Go structs with Solidity structs`,
		},
		{
			abiType: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "addressField", Type: "bool"}, {Name: "boolField", Type: "bool"}, {Name: "uintField", Type: "uint256"}})),
			goType:  reflect.TypeOf(*new(StructWithPrimitiveFields)),
			err:     `ABI type (bool,bool,uint256) (represented by struct { AddressField bool "json:\"addressField\""; BoolField bool "json:\"boolField\""; UintField *big.Int "json:\"uintField\"" }) is not assignable to Go type script.StructWithPrimitiveFields: ABI field AddressField does not match Go field AddressField: ABI type bool (represented by bool) is not assignable to Go type common.Address`,
		},
		{
			abiType: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "nested", Type: "tuple", Components: structWithPrimitiveFieldsMarshalling}})),
			goType:  reflect.TypeOf(*new(StructWithPrimitiveFieldsWrapper)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "nested", Type: "tuple", Components: []abi.ArgumentMarshaling{{Name: "addressField", Type: "bool"}, {Name: "boolField", Type: "bool"}, {Name: "uintField", Type: "uint256"}}}})),
			goType:  reflect.TypeOf(*new(StructWithPrimitiveFieldsWrapper)),
			err:     `ABI type ((bool,bool,uint256)) (represented by struct { Nested struct { AddressField bool "json:\"addressField\""; BoolField bool "json:\"boolField\""; UintField *big.Int "json:\"uintField\"" } "json:\"nested\"" }) is not assignable to Go type script.StructWithPrimitiveFieldsWrapper: ABI field Nested does not match Go field Nested: ABI type (bool,bool,uint256) (represented by struct { AddressField bool "json:\"addressField\""; BoolField bool "json:\"boolField\""; UintField *big.Int "json:\"uintField\"" }) is not assignable to Go type script.StructWithPrimitiveFields: ABI field AddressField does not match Go field AddressField: ABI type bool (represented by bool) is not assignable to Go type common.Address`,
		},
		{
			abiType: die(abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{})),
			goType:  reflect.TypeOf(*new([]struct{})),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("tuple[]", "", structWithPrimitiveFieldsMarshalling)),
			goType:  reflect.TypeOf(*new([]StructWithPrimitiveFields)),
			err:     ``,
		},
		{
			abiType: die(abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{{Name: "boolField", Type: "bool"}, {Name: "addressField", Type: "address"}, {Name: "uintField", Type: "uint256"}})),
			goType:  reflect.TypeOf(*new([]StructWithPrimitiveFields)),
			err:     `ABI type (bool,address,uint256)[] (represented by []struct { BoolField bool "json:\"boolField\""; AddressField common.Address "json:\"addressField\""; UintField *big.Int "json:\"uintField\"" }) is not assignable to Go type []script.StructWithPrimitiveFields: ABI type (bool,address,uint256) (represented by struct { BoolField bool "json:\"boolField\""; AddressField common.Address "json:\"addressField\""; UintField *big.Int "json:\"uintField\"" }) is not assignable to Go type script.StructWithPrimitiveFields: ABI field name BoolField at index 0 does not match Go field name AddressField. Please make sure to match the Go structs with Solidity structs`,
		},
		{
			abiType: die(abi.NewType("tuple[2]", "", structWithPrimitiveFieldsMarshalling)),
			goType:  reflect.TypeOf(*new([3]StructWithPrimitiveFields)),
			err:     `ABI type (address,bool,uint256)[2] (represented by [2]struct { AddressField common.Address "json:\"addressField\""; BoolField bool "json:\"boolField\""; UintField *big.Int "json:\"uintField\"" }) is not assignable to Go type [3]script.StructWithPrimitiveFields: expected an array of length 2, got length 3`,
		},
	}

	for _, test := range matchTypesTests {
		t.Run(fmt.Sprintf("%s <-> %s", test.abiType, test.goType), func(t *testing.T) {
			err := matchTypes(test.abiType, test.goType)

			if test.err == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.err)
			}
		})
	}
}

func TestMatchArguments(t *testing.T) {
	type matchArgumentsTest struct {
		abiArguments abi.Arguments
		goTypes      []reflect.Type
		err          string
	}

	matchArgumentsTests := []matchArgumentsTest{
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("uint256", "", []abi.ArgumentMarshaling{})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(new(big.Int))},
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("uint256[]", "", []abi.ArgumentMarshaling{})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new([]*big.Int))},
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("uint256[2]", "", []abi.ArgumentMarshaling{})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new([2]*big.Int))},
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("uint256[]", "", []abi.ArgumentMarshaling{})),
				},
			},
			goTypes: []reflect.Type{},
			err:     `ABI arguments don't match Go types: ABI has 1 arguments, Go has 0`,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("uint256[]", "", []abi.ArgumentMarshaling{})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new(big.Int))},
			err:     `ABI argument  at index 0 doesn't match Go type: ABI type uint256[] (represented by []*big.Int) is not assignable to Go type big.Int`,
		},
		{
			abiArguments: abi.Arguments{},
			goTypes:      []reflect.Type{reflect.TypeOf(*new([]*big.Int))},
			err:          `ABI arguments don't match Go types: ABI has 0 arguments, Go has 1`,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("uint256[2]", "", []abi.ArgumentMarshaling{})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new([2]*big.Int))},
			err:     ``,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("uint256[2]", "", []abi.ArgumentMarshaling{})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new([2]*big.Int))},
			err:     ``,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("uint256[2]", "", []abi.ArgumentMarshaling{})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new([]*big.Int))},
			err:     `ABI argument  at index 0 doesn't match Go type: ABI type uint256[2] (represented by [2]*big.Int) is not assignable to Go type []*big.Int`,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("uint256[2]", "", []abi.ArgumentMarshaling{})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new([2]*string))},
			err:     `ABI argument  at index 0 doesn't match Go type: ABI type uint256[2] (represented by [2]*big.Int) is not assignable to Go type [2]*string: ABI type uint256 (represented by *big.Int) is not assignable to Go type *string`,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "field", Type: "uint256"}, {Name: "otherField", Type: "address"}})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new(struct {
				FieldRenamed      *big.Int       `abi:"field"`
				OtherFieldRenamed common.Address `abi:"otherField"`
			}))},
			err: ``,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "field", Type: "uint256"}, {Name: "otherField", Type: "address"}})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new(struct {
				Field      *big.Int       `abi:"otherField"`
				OtherField common.Address `abi:"field"`
			}))},
			err: `ABI argument  at index 0 doesn't match Go type: ABI type (uint256,address) (represented by struct { Field *big.Int "json:\"field\""; OtherField common.Address "json:\"otherField\"" }) is not assignable to Go type struct { Field *big.Int "abi:\"otherField\""; OtherField common.Address "abi:\"field\"" }: ABI field name Field at index 0 does not match Go field name Field. Please make sure to match the Go structs with Solidity structs`,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "field", Type: "address"}})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new(uint8))},
			err:     `ABI argument  at index 0 doesn't match Go type: ABI type (address) (represented by struct { Field common.Address "json:\"field\"" }) is not assignable to Go type uint8`,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "field", Type: "address"}})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new(struct{ Field *big.Int }))},
			err:     `ABI argument  at index 0 doesn't match Go type: ABI type (address) (represented by struct { Field common.Address "json:\"field\"" }) is not assignable to Go type struct { Field *big.Int }: ABI field Field does not match Go field Field: ABI type address (represented by common.Address) is not assignable to Go type *big.Int`,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "field", Type: "address"}, {Name: "otherField", Type: "uint256"}})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new(struct{ Field common.Address }))},
			err:     `ABI argument  at index 0 doesn't match Go type: ABI type (address,uint256) (represented by struct { Field common.Address "json:\"field\""; OtherField *big.Int "json:\"otherField\"" }) is not assignable to Go type struct { Field common.Address }: the number of struct fields doesn't match: ABI type has 2, Go type has 1`,
		},
		{
			abiArguments: abi.Arguments{
				{
					Name: "",
					Type: die(abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "field", Type: "address"}, {Name: "otherField", Type: "uint256"}})),
				},
			},
			goTypes: []reflect.Type{reflect.TypeOf(*new(struct {
				OtherField *big.Int
				Field      common.Address
			}))},
			err: `ABI argument  at index 0 doesn't match Go type: ABI type (address,uint256) (represented by struct { Field common.Address "json:\"field\""; OtherField *big.Int "json:\"otherField\"" }) is not assignable to Go type struct { OtherField *big.Int; Field common.Address }: ABI field name Field at index 0 does not match Go field name OtherField. Please make sure to match the Go structs with Solidity structs`,
		},
	}

	for _, test := range matchArgumentsTests {
		t.Run(fmt.Sprintf("%v <-> %v", test.abiArguments, test.goTypes), func(t *testing.T) {
			err := matchArguments(test.abiArguments, test.goTypes...)

			if test.err == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.err)
			}
		})
	}
}
