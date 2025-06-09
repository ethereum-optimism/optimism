package bindings

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

//nolint:unused
type TestSimpleStructA struct {
	a *big.Int
	b []byte
	c common.Address
}

//nolint:unused
type TestSimpleStructB struct {
	a [3]byte
	b [32]byte
	c *uint256.Int
}

//nolint:unused
type TestNestedStruct struct {
	a TestSimpleStructA
	b TestSimpleStructB
	c [3]TestSimpleStructA
}

//nolint:unused
type TestComplexStruct struct {
	a TestSimpleStructB
	b []TestNestedStruct
	c TestSimpleStructA
	d *big.Int
	e TestSimpleStructB
	f TestSimpleStructA
	g [5]TestNestedStruct
	h []byte
	i [5]byte
}

//nolint:unused
type TestNestedStructVarLen struct {
	a []TestNestedStruct
}

//nolint:unused
type TestNestedStructFixLen struct {
	a [7]TestNestedStruct
}

//nolint:unused
type TestRecursiveStruct struct {
	a TestNestedStruct
}

//nolint:unused
type TestRecursiveStruct2 struct {
	a TestRecursiveStruct
}

//nolint:unused
type TestRecursiveStruct3 struct {
	a TestRecursiveStruct2
}

func TestTypeConversion(t *testing.T) {
	type testCase struct {
		value    any
		want     string
		testName string
	}

	tests := []testCase{
		{
			value:    eth.ETH{},
			want:     "uint256",
			testName: "eth.ETH",
		},
		{
			value:    eth.ChainID{},
			want:     "uint256",
			testName: "eth.ChainID",
		},
		{
			value:    common.Address{},
			want:     "address",
			testName: "address (value)",
		},
		{
			value:    &common.Address{},
			want:     "address",
			testName: "address (pointer)",
		},
		{
			value:    big.NewInt(0),
			want:     "uint256",
			testName: "big.NewInt",
		},
		{
			value:    []byte{0x13},
			want:     "bytes",
			testName: "bytes",
		},
		{
			value:    [3]byte{0x13, 0x33, 0x37},
			want:     "bytes3",
			testName: "fixed size bytes",
		},
		{
			value:    TestSimpleStructA{},
			want:     "(uint256,bytes,address)",
			testName: "SimpleStructA (value)",
		},
		{
			value:    &TestSimpleStructA{},
			want:     "(uint256,bytes,address)",
			testName: "SimpleStructA (pointer)",
		},
		{
			value:    TestSimpleStructB{},
			want:     "(bytes3,bytes32,uint256)",
			testName: "SimpleStructB",
		},
		{
			value:    TestNestedStruct{},
			want:     "((uint256,bytes,address),(bytes3,bytes32,uint256),(uint256,bytes,address)[3])",
			testName: "NestedStruct",
		},
		{
			value:    TestNestedStructVarLen{},
			want:     "(((uint256,bytes,address),(bytes3,bytes32,uint256),(uint256,bytes,address)[3])[])",
			testName: "TestNestedStructVarLen",
		},
		{
			value:    TestNestedStructFixLen{},
			want:     "(((uint256,bytes,address),(bytes3,bytes32,uint256),(uint256,bytes,address)[3])[7])",
			testName: "TestNestedStructFixLen",
		},
		{
			value:    TestRecursiveStruct2{},
			want:     "((((uint256,bytes,address),(bytes3,bytes32,uint256),(uint256,bytes,address)[3])))",
			testName: "RecursiveStruct2",
		},
		{
			value:    TestRecursiveStruct3{},
			want:     "(((((uint256,bytes,address),(bytes3,bytes32,uint256),(uint256,bytes,address)[3]))))",
			testName: "RecursiveStruct3",
		},
		{
			value:    &TestRecursiveStruct3{},
			want:     "(((((uint256,bytes,address),(bytes3,bytes32,uint256),(uint256,bytes,address)[3]))))",
			testName: "RecursiveStruct3 (pointer)",
		},
		{
			value:    TestComplexStruct{},
			want:     "((bytes3,bytes32,uint256),((uint256,bytes,address),(bytes3,bytes32,uint256),(uint256,bytes,address)[3])[],(uint256,bytes,address),uint256,(bytes3,bytes32,uint256),(uint256,bytes,address),((uint256,bytes,address),(bytes3,bytes32,uint256),(uint256,bytes,address)[3])[5],bytes,bytes5)",
			testName: "ComplexStruct",
		},
		{
			value:    ABIIdentifier{},
			want:     "(address,uint256,uint256,uint256,uint256)",
			testName: "supervisor Identifier",
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {
			abiTargetType := CustomTypeToGoType(reflect.TypeOf(tc.value))
			typ, _, err := goTypeToABIType(abiTargetType)
			require.NoError(t, err)
			require.Equal(t, tc.want, typ.String())
		})
	}
}
