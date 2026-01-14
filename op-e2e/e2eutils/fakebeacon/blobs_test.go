package fakebeacon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/blobstore"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestBlobsEndpoints consolidates multiple checks of the /eth/v1/beacon/blobs/ endpoint
// into subtests that share a single setup. The subtests cover:
//   - returning all blobs for a slot (no query params)
//   - returning blobs filtered by a single versioned_hash
//   - returning blobs filtered by multiple versioned_hashes (both should be returned)
func TestBlobsEndpoints(t *testing.T) {
	t.Parallel()

	l := testlog.Logger(t, log.LevelInfo)

	// shared setup: in-memory blob store
	blobStore := blobstore.New()
	beaconApi := NewBeacon(l, blobStore, uint64(0), uint64(12))
	t.Cleanup(func() { _ = beaconApi.Close() })
	require.NoError(t, beaconApi.Start("127.0.0.1:0"))

	// Prepare bundles for different slots used in subtests.

	// Slot 10: single blob (pattern used by first subtest)
	var blobSlot10 eth.Blob
	for i := range blobSlot10 {
		blobSlot10[i] = 0xAB
	}
	commit10 := make([]byte, 48)
	for i := range commit10 {
		commit10[i] = byte(i)
	}
	proof10 := make([]byte, 48)
	bundle10 := engine.BlobsBundle{
		Commitments: []hexutil.Bytes{hexutil.Bytes(commit10)},
		Proofs:      []hexutil.Bytes{hexutil.Bytes(proof10)},
		Blobs:       []hexutil.Bytes{hexutil.Bytes(blobSlot10[:])},
	}
	slot10 := uint64(10)
	require.NoError(t, beaconApi.StoreBlobsBundle(slot10, &bundle10))

	// Slot 20: single blob, we'll query by its versioned hash
	var blobSlot20 eth.Blob
	blobSlot20[0] = 0x42
	commit20 := make([]byte, 48)
	for i := range commit20 {
		commit20[i] = byte(100 + i)
	}
	proof20 := make([]byte, 48)
	bundle20 := engine.BlobsBundle{
		Commitments: []hexutil.Bytes{hexutil.Bytes(commit20)},
		Proofs:      []hexutil.Bytes{hexutil.Bytes(proof20)},
		Blobs:       []hexutil.Bytes{hexutil.Bytes(blobSlot20[:])},
	}
	slot20 := uint64(20)
	require.NoError(t, beaconApi.StoreBlobsBundle(slot20, &bundle20))

	// Slot 15: two blobs; used to test multiple versioned_hashes query
	var blobA, blobB eth.Blob
	blobA[0] = 0x11
	blobB[0] = 0x22
	commitA := make([]byte, 48)
	commitB := make([]byte, 48)
	for i := range commitA {
		commitA[i] = byte(i)
		commitB[i] = byte(i + 1)
	}
	proofAB := make([]byte, 48)
	bundle15 := engine.BlobsBundle{
		Commitments: []hexutil.Bytes{hexutil.Bytes(commitA), hexutil.Bytes(commitB)},
		Proofs:      []hexutil.Bytes{hexutil.Bytes(proofAB), hexutil.Bytes(proofAB)},
		Blobs:       []hexutil.Bytes{hexutil.Bytes(blobA[:]), hexutil.Bytes(blobB[:])},
	}
	slot15 := uint64(15)
	require.NoError(t, beaconApi.StoreBlobsBundle(slot15, &bundle15))

	// Helper to perform GET and decode response
	getBlobs := func(url string) (eth.APIBeaconBlobsResponse, error) {
		var resp eth.APIBeaconBlobsResponse
		r, err := http.Get(url)
		if err != nil {
			return resp, err
		}
		defer r.Body.Close()
		if r.StatusCode != http.StatusOK {
			return resp, fmt.Errorf("unexpected status: %d", r.StatusCode)
		}
		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			return resp, err
		}
		return resp, nil
	}

	t.Run("GetAllBlobsForSlot", func(t *testing.T) {
		t.Parallel()
		url := fmt.Sprintf("%s/eth/v1/beacon/blobs/%d", beaconApi.BeaconAddr(), slot10)
		apiResp, err := getBlobs(url)
		require.NoError(t, err)
		require.Len(t, apiResp.Data, 1)
		require.Equal(t, blobSlot10, *apiResp.Data[0])
	})

	t.Run("GetBlobsBySingleVersionedHash", func(t *testing.T) {
		t.Parallel()
		// compute versioned hash for slot20's commitment
		var c kzg4844.Commitment
		copy(c[:], commit20)
		vh := eth.KZGToVersionedHash(c)

		url := fmt.Sprintf("%s/eth/v1/beacon/blobs/%d?versioned_hashes=%s", beaconApi.BeaconAddr(), slot20, vh.Hex())
		apiResp, err := getBlobs(url)
		require.NoError(t, err)
		require.Len(t, apiResp.Data, 1)
		require.Equal(t, blobSlot20, *apiResp.Data[0])
	})

	t.Run("GetBlobsByMultipleVersionedHashes", func(t *testing.T) {
		t.Parallel()
		var ca, cb kzg4844.Commitment
		copy(ca[:], commitA)
		copy(cb[:], commitB)
		vhA := eth.KZGToVersionedHash(ca)
		vhB := eth.KZGToVersionedHash(cb)

		// Provide two versioned_hashes params; order shouldn't matter in behavior
		url := fmt.Sprintf("%s/eth/v1/beacon/blobs/%d?versioned_hashes=%s&versioned_hashes=%s", beaconApi.BeaconAddr(), slot15, vhA.Hex(), vhB.Hex())
		apiResp, err := getBlobs(url)
		require.NoError(t, err)
		// Both blobs should be returned (order is not strictly specified by the endpoint),
		// so assert we have exactly two and that both expected blobs are present.
		require.Len(t, apiResp.Data, 2)

		// Verify both expected blobs are somewhere in the response
		foundA, foundB := false, false
		for _, b := range apiResp.Data {
			if *b == blobA {
				foundA = true
			}
			if *b == blobB {
				foundB = true
			}
		}
		require.True(t, foundA, "blobA not returned")
		require.True(t, foundB, "blobB not returned")
	})
}
