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
	zero := uint64(0)
	beaconApi := NewBeacon(l, blobStore, zero, uint64(12))
	t.Cleanup(func() { _ = beaconApi.Close() })
	require.NoError(t, beaconApi.Start("127.0.0.1:0"))

	blobToCommitmentProofAndBundle := func(blob eth.Blob) (kzg4844.Commitment, kzg4844.Proof, engine.BlobsBundle) {
		kzgBlob := kzg4844.Blob(blob)
		commitment, err := kzg4844.BlobToCommitment(&kzgBlob)
		require.NoError(t, err)
		proof, err := kzg4844.ComputeBlobProof(&kzgBlob, commitment)
		require.NoError(t, err)
		bundle := engine.BlobsBundle{
			Commitments: []hexutil.Bytes{hexutil.Bytes(commitment[:])},
			Proofs:      []hexutil.Bytes{hexutil.Bytes(proof[:])},
			Blobs:       []hexutil.Bytes{hexutil.Bytes(blob[:])},
		}
		return commitment, proof, bundle
	}

	// Prepare bundles for different slots used in subtests.

	// Slot 10: single blob (pattern used by first subtest)
	var blobSlot10 eth.Blob
	for i := range blobSlot10 {
		blobSlot10[i] = 0x01
	}
	_, _, bundle10 := blobToCommitmentProofAndBundle(blobSlot10)
	slot10 := uint64(10)
	require.NoError(t, beaconApi.StoreBlobsBundle(slot10, &bundle10))

	// Slot 20: single blob, we'll query by its versioned hash
	var blobSlot20 eth.Blob
	blobSlot20[0] = 0x42
	commit20, _, bundle20 := blobToCommitmentProofAndBundle(blobSlot20)
	slot20 := uint64(20)
	require.NoError(t, beaconApi.StoreBlobsBundle(slot20, &bundle20))

	// Slot 15: four blobs; used to test multiple versioned_hashes query
	var blobA, blobB, blobC, blobD eth.Blob
	blobA[0] = 0x11
	blobB[0] = 0x22
	blobC[0] = 0x33
	blobD[0] = 0x44
	commitA, proofA, _ := blobToCommitmentProofAndBundle(blobA)
	commitB, proofB, _ := blobToCommitmentProofAndBundle(blobB)
	commitC, proofC, _ := blobToCommitmentProofAndBundle(blobC)
	commitD, proofD, _ := blobToCommitmentProofAndBundle(blobD)
	bundle15 := engine.BlobsBundle{
		Commitments: []hexutil.Bytes{hexutil.Bytes(commitA[:]), hexutil.Bytes(commitB[:]), hexutil.Bytes(commitC[:]), hexutil.Bytes(commitD[:])},
		Proofs:      []hexutil.Bytes{hexutil.Bytes(proofA[:]), hexutil.Bytes(proofB[:]), hexutil.Bytes(proofC[:]), hexutil.Bytes(proofD[:])},
		Blobs:       []hexutil.Bytes{hexutil.Bytes(blobA[:]), hexutil.Bytes(blobB[:]), hexutil.Bytes(blobC[:]), hexutil.Bytes(blobD[:])},
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
		return resp, json.NewDecoder(r.Body).Decode(&resp)
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
		vh := eth.KZGToVersionedHash(commit20)

		url := fmt.Sprintf("%s/eth/v1/beacon/blobs/%d?versioned_hashes=%s", beaconApi.BeaconAddr(), slot20, vh.Hex())
		apiResp, err := getBlobs(url)
		require.NoError(t, err)
		require.Len(t, apiResp.Data, 1)
		require.Equal(t, blobSlot20, *apiResp.Data[0])
	})

	t.Run("GetBlobsByMultipleVersionedHashesProperSubset", func(t *testing.T) {
		t.Parallel()
		vhA := eth.KZGToVersionedHash(commitA)
		vhC := eth.KZGToVersionedHash(commitC)

		// Provide two versioned_hashes params;
		// Let's reverse the order in the query params for a stronger test
		// And remember we stored 4 blobs in this slot, so the query is for a proper subset
		url := fmt.Sprintf("%s/eth/v1/beacon/blobs/%d?versioned_hashes=%s&versioned_hashes=%s", beaconApi.BeaconAddr(), slot15, vhC.Hex(), vhA.Hex())
		apiResp, err := getBlobs(url)
		require.NoError(t, err)
		// Both blobs should be returned (order is not strictly specified by the endpoint),
		// so assert we have exactly two and that both expected blobs are present.
		require.Len(t, apiResp.Data, 2)

		require.Condition(t, func() bool {
			for _, b := range apiResp.Data {
				if *b == blobA {
					return true
				}
			}
			return false
		}, "blobA not returned")

		require.Condition(t, func() bool {
			for _, b := range apiResp.Data {
				if *b == blobC {
					return true
				}
			}
			return false
		}, "blobC not returned")

	})
}
