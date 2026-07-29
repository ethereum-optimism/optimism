package jsonutil

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type LazySortedJsonMapTestCase[K comparable, V any] struct {
	Object LazySortedJsonMap[K, V]
	Json   string
}

func (tc *LazySortedJsonMapTestCase[K, V]) Run(t *testing.T) {
	t.Run("Marshal", func(t *testing.T) {
		got, err := json.Marshal(tc.Object)
		require.NoError(t, err)
		require.Equal(t, tc.Json, string(got), "json output must match")
	})
	t.Run("Unmarshal", func(t *testing.T) {
		var dest LazySortedJsonMap[K, V]
		err := json.Unmarshal([]byte(tc.Json), &dest)
		require.NoError(t, err)
		require.Equal(t, len(tc.Object), len(dest), "lengths match")
		for k, v := range tc.Object {
			require.Equal(t, v, dest[k], "values of %q match", k)
		}
	})
}

func TestLazySortedJsonMap(t *testing.T) {
	testCases := []interface{ Run(t *testing.T) }{
		&LazySortedJsonMapTestCase[string, int]{Object: LazySortedJsonMap[string, int]{}, Json: `{}`},
		&LazySortedJsonMapTestCase[string, int]{Object: LazySortedJsonMap[string, int]{"a": 1, "c": 2, "b": 3}, Json: `{"a":1,"b":3,"c":2}`},
		&LazySortedJsonMapTestCase[common.Address, int]{Object: LazySortedJsonMap[common.Address, int]{
			common.HexToAddress("0x4100000000000000000000000000000000000000"): 123,
			common.HexToAddress("0x4200000000000000000000000000000000000000"): 100,
			common.HexToAddress("0x4200000000000000000000000000000000000001"): 100,
		},
			Json: `{"0x4100000000000000000000000000000000000000":123,` +
				`"0x4200000000000000000000000000000000000000":100,` +
				`"0x4200000000000000000000000000000000000001":100}`},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d", i), tc.Run)
	}
}
