package eth

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalOutput_UnknownVersion(t *testing.T) {
	_, err := UnmarshalOutput([]byte{0: 0xA, 32: 0xA})
	require.ErrorIs(t, err, ErrInvalidOutputVersion)
}

func TestUnmarshalOutput_TooShortForVersion(t *testing.T) {
	_, err := UnmarshalOutput([]byte{0xA})
	require.ErrorIs(t, err, ErrInvalidOutput)
}

func TestOutputV0Codec(t *testing.T) {
	output := OutputV0{
		StateRoot:                Bytes32{1, 2, 3},
		MessagePasserStorageRoot: Bytes32{4, 5, 6},
		BlockHash:                common.Hash{7, 8, 9},
	}
	marshaled := output.Marshal()
	unmarshaled, err := UnmarshalOutput(marshaled)
	require.NoError(t, err)
	unmarshaledV0 := unmarshaled.(*OutputV0)
	require.Equal(t, output, *unmarshaledV0)

	_, err = UnmarshalOutput([]byte{64: 0xA})
	require.ErrorIs(t, err, ErrInvalidOutput)
}

func TestOutputJsonMarshal(t *testing.T) {
	output := OutputV0{
		StateRoot:                Bytes32{1, 2, 3},
		MessagePasserStorageRoot: Bytes32{4, 5, 6},
		BlockHash:                common.Hash{7, 8, 9},
	}
	jsonOutput, _ := json.Marshal(output)
	expectedJson, err := os.ReadFile("testdata/output_v0.json")
	require.NoError(t, err)
	require.JSONEq(t, string(expectedJson), string(jsonOutput))

	var unmarshaled OutputV0
	require.NoError(t, json.Unmarshal([]byte(expectedJson), &unmarshaled))
	require.Equal(t, output, unmarshaled)
}
