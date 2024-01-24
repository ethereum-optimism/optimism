package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/crypto/kzg4844"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	versionMethod        = "eth/v1/node/version"
	genesisMethod        = "eth/v1/beacon/genesis"
	specMethod           = "eth/v1/config/spec"
	sidecarsMethodPrefix = "eth/v1/beacon/blob_sidecars/"
)

type L1BeaconClientConfig struct {
	FetchAllSidecars bool
}

type L1BeaconClient struct {
	cl  client.HTTP
	cfg L1BeaconClientConfig

	initLock     sync.Mutex
	timeToSlotFn TimeToSlotFn
}

// NewL1BeaconClient returns a client for making requests to an L1 consensus layer node.
func NewL1BeaconClient(cl client.HTTP, cfg L1BeaconClientConfig) *L1BeaconClient {
	return &L1BeaconClient{cl: cl, cfg: cfg}
}

func (cl *L1BeaconClient) apiReq(ctx context.Context, dest any, reqPath string, reqQuery url.Values) error {
	headers := http.Header{}
	headers.Add("Accept", "application/json")
	resp, err := cl.cl.Get(ctx, reqPath, reqQuery, headers)
	if err != nil {
		return fmt.Errorf("http Get failed: %w", err)
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
		return fmt.Errorf("failed to close response body: %w", err)
	}
	return nil
}

type TimeToSlotFn func(timestamp uint64) (uint64, error)

// GetTimeToSlotFn returns a function that converts a timestamp to a slot number.
func (cl *L1BeaconClient) GetTimeToSlotFn(ctx context.Context) (TimeToSlotFn, error) {
	cl.initLock.Lock()
	defer cl.initLock.Unlock()
	if cl.timeToSlotFn != nil {
		return cl.timeToSlotFn, nil
	}

	var genesisResp eth.APIGenesisResponse
	if err := cl.apiReq(ctx, &genesisResp, genesisMethod, nil); err != nil {
		return nil, err
	}

	var configResp eth.APIConfigResponse
	if err := cl.apiReq(ctx, &configResp, specMethod, nil); err != nil {
		return nil, err
	}

	genesisTime := uint64(genesisResp.Data.GenesisTime)
	secondsPerSlot := uint64(configResp.Data.SecondsPerSlot)
	if secondsPerSlot == 0 {
		return nil, fmt.Errorf("got bad value for seconds per slot: %v", configResp.Data.SecondsPerSlot)
	}
	cl.timeToSlotFn = func(timestamp uint64) (uint64, error) {
		if timestamp < genesisTime {
			return 0, fmt.Errorf("provided timestamp (%v) precedes genesis time (%v)", timestamp, genesisTime)
		}
		return (timestamp - genesisTime) / secondsPerSlot, nil
	}
	return cl.timeToSlotFn, nil
}

// GetBlobSidecars fetches blob sidecars that were confirmed in the specified
// L1 block with the given indexed hashes.
// Order of the returned sidecars is guaranteed to be that of the hashes.
// Blob data is not checked for validity.
func (cl *L1BeaconClient) GetBlobSidecars(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.BlobSidecar, error) {
	if len(hashes) == 0 {
		return []*eth.BlobSidecar{}, nil
	}
	slotFn, err := cl.GetTimeToSlotFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get time to slot function: %w", err)
	}
	slot, err := slotFn(ref.Time)
	if err != nil {
		return nil, fmt.Errorf("error in converting ref.Time to slot: %w", err)
	}

	reqPath := path.Join(sidecarsMethodPrefix, strconv.FormatUint(slot, 10))
	var reqQuery url.Values
	if !cl.cfg.FetchAllSidecars {
		reqQuery = url.Values{}
		for i := range hashes {
			reqQuery.Add("indices", strconv.FormatUint(hashes[i].Index, 10))
		}
	}

	var resp eth.APIGetBlobSidecarsResponse
	if err := cl.apiReq(ctx, &resp, reqPath, reqQuery); err != nil {
		return nil, fmt.Errorf("failed to fetch blob sidecars for slot %v block %v: %w", slot, ref, err)
	}

	apiscs := make([]*eth.APIBlobSidecar, 0, len(hashes))
	// filter and order by hashes
	for _, h := range hashes {
		for _, apisc := range resp.Data {
			if h.Index == uint64(apisc.Index) {
				apiscs = append(apiscs, apisc)
				break
			}
		}
	}

	if len(hashes) != len(apiscs) {
		return nil, fmt.Errorf("expected %v sidecars but got %v", len(hashes), len(resp.Data))
	}

	bscs := make([]*eth.BlobSidecar, 0, len(hashes))
	for _, apisc := range apiscs {
		bscs = append(bscs, apisc.BlobSidecar())
	}

	return bscs, nil
}

// GetBlobs fetches blobs that were confirmed in the specified L1 block with the given indexed
// hashes. The order of the returned blobs will match the order of `hashes`.  Confirms each
// blob's validity by checking its proof against the commitment, and confirming the commitment
// hashes to the expected value. Returns error if any blob is found invalid.
func (cl *L1BeaconClient) GetBlobs(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.Blob, error) {
	blobSidecars, err := cl.GetBlobSidecars(ctx, ref, hashes)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob sidecars for L1BlockRef %s: %w", ref, err)
	}
	return blobsFromSidecars(blobSidecars, hashes)
}

func blobsFromSidecars(blobSidecars []*eth.BlobSidecar, hashes []eth.IndexedBlobHash) ([]*eth.Blob, error) {
	if len(blobSidecars) != len(hashes) {
		return nil, fmt.Errorf("number of hashes and blobSidecars mismatch, %d != %d", len(hashes), len(blobSidecars))
	}

	out := make([]*eth.Blob, len(hashes))
	for i, ih := range hashes {
		sidecar := blobSidecars[i]
		if sidx := uint64(sidecar.Index); sidx != ih.Index {
			return nil, fmt.Errorf("expected sidecars to be ordered by hashes, but got %d != %d", sidx, ih.Index)
		}

		// make sure the blob's kzg commitment hashes to the expected value
		hash := eth.KZGToVersionedHash(kzg4844.Commitment(sidecar.KZGCommitment))
		if hash != ih.Hash {
			return nil, fmt.Errorf("expected hash %s for blob at index %d but got %s", ih.Hash, ih.Index, hash)
		}

		// confirm blob data is valid by verifying its proof against the commitment
		if err := eth.VerifyBlobProof(&sidecar.Blob, kzg4844.Commitment(sidecar.KZGCommitment), kzg4844.Proof(sidecar.KZGProof)); err != nil {
			return nil, fmt.Errorf("blob at index %d failed verification: %w", i, err)
		}
		out[i] = &sidecar.Blob
	}
	return out, nil
}

// GetVersion fetches the version of the Beacon-node.
func (cl *L1BeaconClient) GetVersion(ctx context.Context) (string, error) {
	var resp eth.APIVersionResponse
	if err := cl.apiReq(ctx, &resp, versionMethod, nil); err != nil {
		return "", err
	}
	return resp.Data.Version, nil
}
