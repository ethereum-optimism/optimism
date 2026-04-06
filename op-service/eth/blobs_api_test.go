package eth_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type dataJson struct {
	Data map[string]any `json:"data"`
}

// TestAPIGenesisResponse tests that json unmarshalling a json response from a
// eth/v1/beacon/genesis beacon node call into a APIGenesisResponse object
// fills all existing fields with the expected values, thereby confirming that
// APIGenesisResponse is compatible with the current beacon node API.
// This also confirms that the [sources.L1BeaconClient] correctly parses
// responses from a real beacon node.
func TestAPIGenesisResponse(t *testing.T) {
	require := require.New(t)
	var resp eth.APIGenesisResponse
	require.Equal(1, reflect.TypeOf(resp.Data).NumField(), "APIGenesisResponse changed, adjust test")

	path := filepath.Join("testdata", "eth_v1_beacon_genesis_goerli.json")
	jsonStr, err := os.ReadFile(path)
	require.NoError(err)

	require.NoError(json.Unmarshal(jsonStr, &resp))
	require.NotZero(resp.Data.GenesisTime)

	jsonMap := &dataJson{Data: make(map[string]any)}
	require.NoError(json.Unmarshal(jsonStr, jsonMap))
	genesisTime, err := resp.Data.GenesisTime.MarshalText()
	require.NoError(err)
	require.Equal(jsonMap.Data["genesis_time"].(string), string(genesisTime))
}

// TestAPIConfigResponse tests that json unmarshalling a json response from a
// eth/v1/config/spec beacon node call into a APIConfigResponse object
// fills all existing fields with the expected values, thereby confirming that
// APIGenesisResponse is compatible with the current beacon node API.
// This also confirms that the [sources.L1BeaconClient] correctly parses
// responses from a real beacon node.
func TestAPIConfigResponse(t *testing.T) {
	require := require.New(t)
	var resp eth.APIConfigResponse
	require.Equal(1, reflect.TypeOf(resp.Data).NumField(), "APIConfigResponse changed, adjust test")

	path := filepath.Join("testdata", "eth_v1_config_spec_goerli.json")
	jsonStr, err := os.ReadFile(path)
	require.NoError(err)

	require.NoError(json.Unmarshal(jsonStr, &resp))
	require.NotZero(resp.Data.SecondsPerSlot)

	jsonMap := &dataJson{Data: make(map[string]any)}
	require.NoError(json.Unmarshal(jsonStr, jsonMap))
	secPerSlot, err := resp.Data.SecondsPerSlot.MarshalText()
	require.NoError(err)
	require.Equal(jsonMap.Data["SECONDS_PER_SLOT"].(string), string(secPerSlot))
}
