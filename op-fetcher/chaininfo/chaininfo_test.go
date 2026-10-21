package chaininfo

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestConfigJSON pins the JSON encoding of Config, which superchain-registry tooling
// consumes.
func TestConfigJSON(t *testing.T) {
	var out Info
	setDistinctAddresses(reflect.ValueOf(&out).Elem(), new(byte))
	out.Permissioned = true
	out.Permissionless = true
	out.RespectedGameType = 1

	got, err := json.MarshalIndent(out.Config(), "", "  ")
	require.NoError(t, err)
	want, err := os.ReadFile("testdata/chain-config.json")
	require.NoError(t, err)
	require.Equal(t, string(want), string(got)+"\n")

	out.RespectedGameType = math.MaxUint32
	cfg := out.Config()
	require.Nil(t, cfg.FaultProofStatus, "no fault proofs")
	got, err = json.Marshal(cfg)
	require.NoError(t, err)
	require.NotContains(t, string(got), "faultProofs")
}

// setDistinctAddresses sets every address field of v, in field order, to a distinct address.
func setDistinctAddresses(v reflect.Value, next *byte) {
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		switch {
		case f.Type() == reflect.TypeOf(common.Address{}):
			*next++
			f.Set(reflect.ValueOf(common.Address{19: *next}))
		case f.Kind() == reflect.Struct:
			setDistinctAddresses(f, next)
		}
	}
}
