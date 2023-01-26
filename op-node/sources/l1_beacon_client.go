package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1BeaconClient struct {
	cl client.HTTP

	initLock     sync.Mutex
	timeToSlotFn TimeToSlotFn
}

func NewL1BeaconClient(cl client.HTTP) *L1BeaconClient {
	return &L1BeaconClient{cl: cl}
}

type BlobsSidecarData struct {
	BeaconBlockRoot eth.Bytes32      `json:"beacon_block_root"`
	BeaconBlockSlot eth.Uint64String `json:"beacon_block_slot"`
	Blobs           []eth.Blob       `json:"blobs"`
	// Some Beacon APIs may include the 4844 KZG proof, but we do not need it,
	// we recompute the commitment from the blob.
	// KzgAggregatedProof eth.Bytes48       `json:"kzg_aggregated_proof"`
}

type APIBlobsSidecarResponse struct {
	Data BlobsSidecarData `json:"data"`
}

type TimeToSlotFn func(timestamp uint64) uint64

type ReducedGenesisData struct {
	GenesisTime eth.Uint64String `json:"genesis_time"`
}

type APIGenesisResponse struct {
	Data ReducedGenesisData `json:"data"`
}

type ReducedConfigData struct {
	SecondsPerSlot eth.Uint64String `json:"SECONDS_PER_SLOT"`
}

type APIConfigResponse struct {
	Data ReducedConfigData `json:"data"`
}

func (cl *L1BeaconClient) apiReq(ctx context.Context, dest any, method string) error {
	headers := http.Header{}
	headers.Add("Accept", "application/json")
	resp, err := cl.cl.Get(ctx, method, headers)
	if err != nil {
		return fmt.Errorf("failed genesis details request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		errMsg, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return fmt.Errorf("failed request with status %d: %s", resp.StatusCode, string(errMsg))
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		_ = resp.Body.Close()
		return err
	}
	if err := resp.Body.Close(); err != nil {
		return fmt.Errorf("failed to close genesis response object: %w", err)
	}
	return nil
}

func (cl *L1BeaconClient) GetTimeToSlotFn(ctx context.Context) (TimeToSlotFn, error) {
	cl.initLock.Lock()
	defer cl.initLock.Unlock()
	if cl.timeToSlotFn != nil {
		return cl.timeToSlotFn, nil
	}

	var genesisResp APIGenesisResponse
	if err := cl.apiReq(ctx, &genesisResp, "eth/v1/beacon/genesis"); err != nil {
		return nil, err
	}

	var configResp APIConfigResponse
	if err := cl.apiReq(ctx, &configResp, "eth/v1/config/spec"); err != nil {
		return nil, err
	}

	cl.timeToSlotFn = func(timestamp uint64) uint64 {
		return (timestamp - uint64(genesisResp.Data.GenesisTime)) / uint64(configResp.Data.SecondsPerSlot)
	}
	return cl.timeToSlotFn, nil
}

// BlobsByRefAndIndexedDataHashes fetches blobs that were confirmed in the given L1 block with the given indexed hashes.
func (cl *L1BeaconClient) BlobsByRefAndIndexedDataHashes(ctx context.Context, ref eth.L1BlockRef, dataHashes []eth.IndexedDataHash) ([]*eth.Blob, error) {
	slotFn, err := cl.GetTimeToSlotFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get time to slot function: %w", err)
	}
	slot := slotFn(ref.Time)

	// TODO: would be nice if there's a L1 API to fetch blobs 1-by-1 to avoid large requests
	// Or even just an optional argument to filter the response data before the beacon node sends it back to us
	var respData APIBlobsSidecarResponse
	if err := cl.apiReq(ctx, &respData, fmt.Sprintf("eth/v1/beacon/blobs_sidecars/%d", slot)); err != nil {
		return nil, fmt.Errorf("failed to fetch blobs sidecar of slot %d (for block %s): %w", slot, ref, err)
	}

	out := make([]*eth.Blob, 0, len(dataHashes))
	for _, ih := range dataHashes {
		if ih.Index > uint64(len(respData.Data.Blobs)) {
			return nil, fmt.Errorf("received less blobs than expected, expected %s at index %d in %s, but only have %d blobs", ih.DataHash, ih.Index, ref, len(respData.Data.Blobs))
		}
		blob := &respData.Data.Blobs[ih.Index]
		kzgCommitment, err := blob.ComputeKZGCommitment()
		if err != nil {
			return nil, fmt.Errorf("failed to compute kzg commitment for blob %s at index %d in %s: %w", ih.DataHash, ih.Index, ref, err)
		}
		dataHash := eth.KzgToVersionedHash(kzgCommitment)
		if dataHash != ih.DataHash {
			return nil, fmt.Errorf("expected datahash %s for blob %d in %s but got blob with datahash %s", ih.DataHash, ih.Index, ref, dataHash)
		}
		out = append(out, blob)
	}
	return out, nil
}
