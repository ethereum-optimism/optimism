package addresses

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestL1Contracts_NoShadowedFields gives every address in L1Contracts a distinct value and
// checks that all of them survive a JSON round-trip. encoding/json drops a field that another
// embedded field of the same name shadows, so this test fails on any untagged name collision.
func TestL1Contracts_NoShadowedFields(t *testing.T) {
	var want L1Contracts
	n := setDistinctAddresses(t, reflect.ValueOf(&want).Elem(), 1)
	require.Greater(t, n, uint64(1))

	data, err := json.Marshal(want)
	require.NoError(t, err)
	var got L1Contracts
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

// setDistinctAddresses sets every common.Address leaf of v, recursing into nested structs, to
// a unique value starting at next. It returns the next unused value.
func setDistinctAddresses(t *testing.T, v reflect.Value, next uint64) uint64 {
	t.Helper()
	addrType := reflect.TypeFor[common.Address]()
	for i := range v.NumField() {
		f := v.Field(i)
		switch {
		case f.Type() == addrType:
			f.Set(reflect.ValueOf(common.BigToAddress(new(big.Int).SetUint64(next))))
			next++
		case f.Kind() == reflect.Struct:
			next = setDistinctAddresses(t, f, next)
		default:
			t.Fatalf("unexpected field %s of kind %s", v.Type().Field(i).Name, f.Kind())
		}
	}
	return next
}
