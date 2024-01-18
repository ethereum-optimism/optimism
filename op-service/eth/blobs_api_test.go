package eth_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

// TestAPIGenesisResponse tests that json unmarshaling a json response from a
// eth/v1/beacon/genesis beacon node call into a APIGenesisResponse object
// fills all exising fields with the expected values, thereby confirming that
// APIGenesisResponse is compatible with the current beacon node API.
// This also confirms that the [sources.L1BeaconClient] correctly parses
// responses from a real beacon node.
func TestAPIGenesisResponse(t *testing.T) {
	var resp eth.APIGenesisResponse
	testBeaconAPIResponse(t, &resp, "eth_v1_beacon_genesis_goerli.json")
	require.NotZero(t, resp.Data.GenesisTime)
}

// TestAPIConfigResponse tests that json unmarshaling a json response from a
// eth/v1/config/spec beacon node call into a APIConfigResponse object
// fills all exising fields with the expected values, thereby confirming that
// APIGenesisResponse is compatible with the current beacon node API.
// This also confirms that the [sources.L1BeaconClient] correctly parses
// responses from a real beacon node.
func TestAPIConfigResponse(t *testing.T) {
	var resp eth.APIConfigResponse
	testBeaconAPIResponse(t, &resp, "eth_v1_config_spec_goerli.json")
	require.NotZero(t, resp.Data.SecondsPerSlot)
}

// TestAPIGetBlobSidecarsResponse tests that json unmarshaling a json response from a
// eth/v1/beacon/blob_sidecars/<X> beacon node call into a APIGetBlobSidecarsResponse object
// fills all exising fields with the expected values, thereby confirming that
// APIGenesisResponse is compatible with the current beacon node API.
// This also confirms that the [sources.L1BeaconClient] correctly parses
// responses from a real beacon node.
func TestAPIGetBlobSidecarsResponse(t *testing.T) {
	var resp eth.APIGetBlobSidecarsResponse
	testBeaconAPIResponse(t, &resp, "eth_v1_beacon_blob_sidecars_7422094_goerli.json")
	require.NotEmpty(t, resp.Data)
	require.NotZero(t, resp.Data[0].Blob)
	//require.NotZero(t, resp.Data[0].Index) // actually 0
	require.NotZero(t, resp.Data[0].KZGCommitment)
	require.NotZero(t, resp.Data[0].KZGProof)
	require.NotZero(t, resp.Data[0].SignedBlockHeader.Message.Slot)
	require.NotZero(t, resp.Data[0].SignedBlockHeader.Message.ParentRoot)
	require.NotZero(t, resp.Data[0].SignedBlockHeader.Message.BodyRoot)
	require.NotZero(t, resp.Data[0].SignedBlockHeader.Message.ProposerIndex)
	require.NotZero(t, resp.Data[0].SignedBlockHeader.Message.StateRoot)
}

// testBeaconAPIResponse tests that json-unmarshaling a Beacon node json response
// read from the provided testfile path into the provided response object works.
func testBeaconAPIResponse(t *testing.T, resp any, testfile string) {
	require := require.New(t)

	path := filepath.Join("testdata", testfile)
	jsonStr, err := os.ReadFile(path)
	require.NoError(err)

	// decode into the response type
	dec := json.NewDecoder(bytes.NewReader(jsonStr))
	require.NoError(dec.Decode(resp), "must decode test data")
}
