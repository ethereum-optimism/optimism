package eth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSuperRootResponseData_UnmarshalJSON(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		in := SuperRootResponseData{
			VerifiedRequiredL1: BlockID{Number: 7},
			Super:              NewSuperV1(9, ChainIDAndOutput{ChainID: ChainIDFromUInt64(1), Output: Bytes32{0xa1}}),
			SuperRoot:          Bytes32{0xdd},
		}
		data, err := json.Marshal(in)
		require.NoError(t, err)
		var out SuperRootResponseData
		require.NoError(t, json.Unmarshal(data, &out))
		require.Equal(t, in, out)
	})

	for name, input := range map[string]string{
		"MissingSuper": `{"super_root":"0x0000000000000000000000000000000000000000000000000000000000000000"}`,
		"NullSuper":    `{"super":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			var out SuperRootResponseData
			require.ErrorContains(t, json.Unmarshal([]byte(input), &out), "missing super")
		})
	}
}
