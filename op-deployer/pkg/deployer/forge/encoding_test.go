package forge

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

// TestGoStructToABITuple_AbiTypeTag verifies that the `abiType` field tag
// overrides the auto-derived ABI type. This is the only way to encode a
// uint128 (or other widths without a native Go primitive).
func TestGoStructToABITuple_AbiTypeTag(t *testing.T) {
	type withTag struct {
		A *big.Int `abi:"a" abiType:"uint128"`
	}
	tup, err := GoStructToABITuple(reflect.TypeOf(withTag{}), "withTag")
	require.NoError(t, err)
	require.Equal(t, "(uint128)", tup.String())
}

// TestGoStructToABITuple_NestedStruct verifies that nested struct fields are
// encoded as nested tuples and that field-level tags inside the nested struct
// are honored.
func TestGoStructToABITuple_NestedStruct(t *testing.T) {
	type inner struct {
		X uint32   `abi:"x"`
		Y *big.Int `abi:"y" abiType:"uint128"`
	}
	type outer struct {
		Top   uint64 `abi:"top"`
		Inner inner  `abi:"inner"`
	}
	tup, err := GoStructToABITuple(reflect.TypeOf(outer{}), "outer")
	require.NoError(t, err)
	require.Equal(t, "(uint64,(uint32,uint128))", tup.String())
}

// TestBytesScriptEncoder_NestedAndUint128 verifies that the BytesScriptEncoder
// can pack a struct that mixes a nested tuple and a uint128 field.
func TestBytesScriptEncoder_NestedAndUint128(t *testing.T) {
	type inner struct {
		Limit uint32   `abi:"limit"`
		Cap   *big.Int `abi:"cap" abiType:"uint128"`
	}
	type outer struct {
		Gas   uint64 `abi:"gas"`
		Inner inner  `abi:"inner"`
	}
	enc := &BytesScriptEncoder[outer]{TypeName: "Outer"}
	cap128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	packed, err := enc.Encode(outer{
		Gas:   5_000_000,
		Inner: inner{Limit: 4_000_000, Cap: cap128},
	})
	require.NoError(t, err)

	// Round-trip via abi.Arguments to confirm shape.
	tup, err := GoStructToABITuple(reflect.TypeOf(outer{}), "Outer")
	require.NoError(t, err)
	args := abi.Arguments{{Type: tup}}
	unpacked, err := args.Unpack(packed)
	require.NoError(t, err)
	require.Len(t, unpacked, 1)
}
