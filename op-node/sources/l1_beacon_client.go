package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/core/types"
	kzgeth "github.com/protolambda/go-kzg/eth"
)

type L1BeaconClient struct {
	cl client.HTTP
}

func NewL1BeaconClient(cl client.HTTP) *L1BeaconClient {
	return &L1BeaconClient{cl: cl}
}

type APIBlobsSidecarResponse struct {
	Data struct {
		BeaconBlockRoot eth.Bytes32      `json:"beacon_block_root"`
		BeaconBlockSlot eth.Uint64String `json:"beacon_block_slot"`
		Blobs           []types.Blob     `json:"blobs"`
		//KzgAggregatedProof eth.Bytes48       `json:"kzg_aggregated_proof"`
		// TODO: ignored proof, Prysm api endpoint returns empty data for some reason. Unused anyway; data-hashes are used for verification.
	} `json:"data"`
}

// BlobsByRefAndIndexedDataHashes fetches blobs that were confirmed in the given L1 block with the given indexed hashes.
func (cl *L1BeaconClient) BlobsByRefAndIndexedDataHashes(ctx context.Context, ref eth.L1BlockRef, dataHashes []eth.IndexedDataHash) ([]types.Blob, error) {
	slot := 0
	var headers http.Header
	headers.Add("Accept", "application/json")
	resp, err := cl.cl.Get(ctx, fmt.Sprintf("eth/v1/beacon/blobs_sidecars/%d", slot), headers)
	if err != nil {
		return nil, fmt.Errorf("failed blobs sidecar request: %w", err)
	}
	var respData APIBlobsSidecarResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		_ = resp.Body.Close()
		return nil, err
	}
	if err := resp.Body.Close(); err != nil {
		return nil, fmt.Errorf("failed to close blobs sidecar response object: %w", err)
	}
	out := make([]types.Blob, 0, len(dataHashes))
	for _, ih := range dataHashes {
		if ih.Index > uint64(len(respData.Data.Blobs)) {
			return nil, fmt.Errorf("received less blobs than expected, expected %s at index %d in %s, but only have %d blobs", ih.DataHash, ih.Index, ref, len(respData.Data.Blobs))
		}
		blob := &respData.Data.Blobs[ih.Index]
		kzgCommitment, ok := kzgeth.BlobToKZGCommitment(blob)
		if !ok {
			return nil, fmt.Errorf("failed to compute kzg commitment for blob %s at index %d in %s", ih.DataHash, ih.Index, ref)
		}
		dataHash := types.KZGCommitment(kzgCommitment).ComputeVersionedHash()
		if dataHash != ih.DataHash {
			return nil, fmt.Errorf("expected datahash %s for blob %d in %s but got blob with datahash %s", ih.DataHash, ih.Index, ref, dataHash)
		}
		out = append(out, *blob)
	}
	return out, nil
}
