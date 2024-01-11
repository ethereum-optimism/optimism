package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto/kzg4844"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	genesisMethod        = "eth/v1/beacon/genesis"
	specMethod           = "eth/v1/config/spec"
	sidecarsMethodPrefix = "eth/v1/beacon/blob_sidecars/"
)

type L1BeaconClient struct {
	cl client.HTTP

	initLock     sync.Mutex
	timeToSlotFn TimeToSlotFn
}

// NewL1BeaconClient returns a client for making requests to an L1 consensus layer node.
func NewL1BeaconClient(cl client.HTTP) *L1BeaconClient {
	return &L1BeaconClient{cl: cl}
}

func (cl *L1BeaconClient) apiReq(ctx context.Context, dest any, method string) error {
	headers := http.Header{}
	headers.Add("Accept", "application/json")
	resp, err := cl.cl.Get(ctx, method, headers)
	if err != nil {
		return fmt.Errorf("%w: http Get failed", err)
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
		return fmt.Errorf("%w: failed to close response body", err)
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
	if err := cl.apiReq(ctx, &genesisResp, genesisMethod); err != nil {
		return nil, err
	}

	var configResp eth.APIConfigResponse
	if err := cl.apiReq(ctx, &configResp, specMethod); err != nil {
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

// GetBlobSidecars fetches blob sidecars that were confirmed in the specified L1 block with the
// given indexed hashes. Order of the returned sidecars is not guaranteed, and blob data is not
// checked for validity.
func (cl *L1BeaconClient) GetBlobSidecars(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.BlobSidecar, error) {
	if len(hashes) == 0 {
		return []*eth.BlobSidecar{}, nil
	}
	slotFn, err := cl.GetTimeToSlotFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get time to slot function", err)
	}
	slot, err := slotFn(ref.Time)
	if err != nil {
		return nil, fmt.Errorf("%w: error in converting ref.Time to slot", err)
	}

	builder := strings.Builder{}
	builder.WriteString(sidecarsMethodPrefix)
	builder.WriteString(strconv.FormatUint(slot, 10))
	builder.WriteRune('?')
	v := url.Values{}
	for i := range hashes {
		v.Add("indices", strconv.FormatUint(hashes[i].Index, 10))
	}
	builder.WriteString(v.Encode())

	var resp eth.APIGetBlobSidecarsResponse
	if err := cl.apiReq(ctx, &resp, builder.String()); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch blob sidecars for slot %v block %v", err, slot, ref)
	}
	if len(hashes) != len(resp.Data) {
		return nil, fmt.Errorf("expected %v sidecars but got %v", len(hashes), len(resp.Data))
	}

	return resp.Data, nil
}

// GetBlobs fetches blobs that were confirmed in the specified L1 block with the given indexed
// hashes. The order of the returned blobs will match the order of `hashes`.  Confirms each
// blob's validity by checking its proof against the commitment, and confirming the commitment
// hashes to the expected value. Returns error if any blob is found invalid.
func (cl *L1BeaconClient) GetBlobs(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.Blob, error) {
	blobSidecars, err := cl.GetBlobSidecars(ctx, ref, hashes)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get blob sidecars for L1BlockRef %s", err, ref)
	}
	return blobsFromSidecars(blobSidecars, hashes)
}

func blobsFromSidecars(blobSidecars []*eth.BlobSidecar, hashes []eth.IndexedBlobHash) ([]*eth.Blob, error) {
	out := make([]*eth.Blob, len(hashes))
	for i, ih := range hashes {
		// The beacon node api makes no guarantees on order of the returned blob sidecars, so
		// search for the sidecar that matches the current indexed hash to ensure blobs are
		// returned in the same order.
		scIndex := slices.IndexFunc(
			blobSidecars,
			func(sc *eth.BlobSidecar) bool { return uint64(sc.Index) == ih.Index })
		if scIndex == -1 {
			return nil, fmt.Errorf("no blob in response matches desired index: %v", ih.Index)
		}
		sidecar := blobSidecars[scIndex]

		// make sure the blob's kzg commitment hashes to the expected value
		hash := eth.KZGToVersionedHash(kzg4844.Commitment(sidecar.KZGCommitment))
		if hash != ih.Hash {
			return nil, fmt.Errorf("expected hash %s for blob at index %d but got %s", ih.Hash, ih.Index, hash)
		}

		// confirm blob data is valid by verifying its proof against the commitment
		if err := eth.VerifyBlobProof(&sidecar.Blob, kzg4844.Commitment(sidecar.KZGCommitment), kzg4844.Proof(sidecar.KZGProof)); err != nil {
			return nil, fmt.Errorf("%w: blob at index %d failed verification", err, i)
		}
		out[i] = &sidecar.Blob
	}
	return out, nil
}
