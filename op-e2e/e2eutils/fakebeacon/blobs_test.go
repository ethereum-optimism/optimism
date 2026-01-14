package fakebeacon

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/blobstore"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestGetBlobs(t *testing.T) {
	t.Parallel()

	l := testlog.Logger(t, log.LevelInfo)

	// Create an in-memory blob store and a fake beacon that exposes the new blobs endpoint
	blobStore := blobstore.New()
	// set fuluTime in the past so the /eth/v1/beacon/blobs/ endpoint is enabled
	past := uint64(time.Now().Add(-time.Minute).Unix())
	beaconApi := NewBeacon(l, blobStore, uint64(0), uint64(12), &past)
	t.Cleanup(func() {
		_ = beaconApi.Close()
	})
	require.NoError(t, beaconApi.Start("127.0.0.1:0"))

	// Prepare a single blob payload to store
	var expected eth.Blob
	for i := range expected {
		expected[i] = 0xAB
	}

	// Prepare a fake commitment and proof (lengths must match expectations; contents can be arbitrary for the test)
	commitment := make([]byte, 48)
	for i := range commitment {
		commitment[i] = byte(i)
	}
	proof := make([]byte, 48)

	// Build an engine.BlobsBundle with one blob
	bundle := engine.BlobsBundle{
		Commitments: []hexutil.Bytes{hexutil.Bytes(commitment)},
		Proofs:      []hexutil.Bytes{hexutil.Bytes(proof)},
		Blobs:       []hexutil.Bytes{hexutil.Bytes(expected[:])},
	}

	// Store the bundle at slot 10
	slot := uint64(10)
	require.NoError(t, beaconApi.StoreBlobsBundle(slot, &bundle))

	// Query the beacon blobs endpoint for that slot (no versioned_hashes -> return all blobs)
	url := beaconApi.BeaconAddr() + "/eth/v1/beacon/blobs/10"
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResp eth.APIBeaconBlobsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&apiResp))
	require.Len(t, apiResp.Data, 1)

	// Verify returned blob matches what we stored
	require.Equal(t, expected, *apiResp.Data[0])
}

// TestGetBlobs_WithVersionedHashes verifies that the endpoint honors the
// versioned_hashes query parameter and returns only the blobs matching the
// supplied versioned hash(es).
func TestGetBlobs_WithVersionedHashes(t *testing.T) {
	t.Parallel()

	l := testlog.Logger(t, log.LevelInfo)

	blobStore := blobstore.New()
	past := uint64(time.Now().Add(-time.Minute).Unix())
	beaconApi := NewBeacon(l, blobStore, uint64(0), uint64(12), &past)
	t.Cleanup(func() {
		_ = beaconApi.Close()
	})
	require.NoError(t, beaconApi.Start("127.0.0.1:0"))

	// Create two distinct blobs and commitments
	var blob1, blob2 eth.Blob
	blob1[0] = 0x11
	blob2[0] = 0x22

	commit1 := make([]byte, 48)
	commit2 := make([]byte, 48)
	for i := range commit1 {
		commit1[i] = byte(i)
		commit2[i] = byte(i + 1)
	}
	proof := make([]byte, 48)

	bundle := engine.BlobsBundle{
		Commitments: []hexutil.Bytes{hexutil.Bytes(commit1), hexutil.Bytes(commit2)},
		Proofs:      []hexutil.Bytes{hexutil.Bytes(proof), hexutil.Bytes(proof)},
		Blobs:       []hexutil.Bytes{hexutil.Bytes(blob1[:]), hexutil.Bytes(blob2[:])},
	}

	slot := uint64(15)
	require.NoError(t, beaconApi.StoreBlobsBundle(slot, &bundle))

	// Compute the versioned hash for commit1; the endpoint should return only blob1 when queried with it.
	vh := eth.KZGToVersionedHash(kzg4844.Commitment(commit1))

	url := beaconApi.BeaconAddr() + "/eth/v1/beacon/blobs/15?versioned_hashes=" + vh.Hex()
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResp eth.APIBeaconBlobsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&apiResp))
	require.Len(t, apiResp.Data, 1)
	require.Equal(t, blob1, *apiResp.Data[0])
}
