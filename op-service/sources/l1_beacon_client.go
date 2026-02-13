package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	versionMethod        = "eth/v1/node/version"
	specMethod           = "eth/v1/config/spec"
	genesisMethod        = "eth/v1/beacon/genesis"
	sidecarsMethodPrefix = "eth/v1/beacon/blob_sidecars/"
	blobsMethodPrefix    = "eth/v1/beacon/blobs/"
)

type L1BeaconClientConfig struct {
	FetchAllSidecars bool
}

// L1BeaconClient is a high level golang client for the Beacon API.
type L1BeaconClient struct {
	cl   apis.BeaconClient
	pool *ClientPool[apis.BeaconClient]
	cfg  L1BeaconClientConfig

	initLock     sync.Mutex
	timeToSlotFn TimeToSlotFn
}

// BeaconHTTPClient implements BeaconClient. It provides golang types over the basic Beacon API.
type BeaconHTTPClient struct {
	cl client.HTTP
}

func NewBeaconHTTPClient(cl client.HTTP) *BeaconHTTPClient {
	return &BeaconHTTPClient{cl}
}

func (cl *BeaconHTTPClient) apiReq(ctx context.Context, dest any, reqPath string, reqQuery url.Values) error {
	headers := http.Header{}
	headers.Add("Accept", "application/json")
	resp, err := cl.cl.Get(ctx, reqPath, reqQuery, headers)
	if err != nil {
		return fmt.Errorf("http Get failed: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		errMsg, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return fmt.Errorf("failed request with status %d: %s: %w", resp.StatusCode, string(errMsg), ethereum.NotFound)
	} else if resp.StatusCode != http.StatusOK {
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

func (cl *BeaconHTTPClient) NodeVersion(ctx context.Context) (string, error) {
	var resp eth.APIVersionResponse
	if err := cl.apiReq(ctx, &resp, versionMethod, nil); err != nil {
		return "", err
	}
	return resp.Data.Version, nil
}

func (cl *BeaconHTTPClient) ConfigSpec(ctx context.Context) (eth.APIConfigResponse, error) {
	var configResp eth.APIConfigResponse
	if err := cl.apiReq(ctx, &configResp, specMethod, nil); err != nil {
		return eth.APIConfigResponse{}, err
	}
	return configResp, nil
}

func (cl *BeaconHTTPClient) BeaconGenesis(ctx context.Context) (eth.APIGenesisResponse, error) {
	var genesisResp eth.APIGenesisResponse
	if err := cl.apiReq(ctx, &genesisResp, genesisMethod, nil); err != nil {
		return eth.APIGenesisResponse{}, err
	}
	return genesisResp, nil
}

func (cl *BeaconHTTPClient) BeaconBlobs(ctx context.Context, slot uint64, hashes []common.Hash) (eth.APIBeaconBlobsResponse, error) {
	reqQuery := url.Values{}
	for _, hash := range hashes {
		reqQuery.Add("versioned_hashes", hash.Hex())
	}
	reqPath := path.Join(blobsMethodPrefix, strconv.FormatUint(slot, 10))
	var blobsResp eth.APIBeaconBlobsResponse
	if err := cl.apiReq(ctx, &blobsResp, reqPath, reqQuery); err != nil {
		return eth.APIBeaconBlobsResponse{}, err
	}
	if len(blobsResp.Data) != len(hashes) {
		return eth.APIBeaconBlobsResponse{}, fmt.Errorf("#returned blobs(%d) != #requested blobs(%d)", len(blobsResp.Data), len(hashes))
	}
	return blobsResp, nil
}

type ClientPool[T any] struct {
	clients []T
	index   int
}

func NewClientPool[T any](clients ...T) *ClientPool[T] {
	return &ClientPool[T]{
		clients: clients,
		index:   0,
	}
}

func (p *ClientPool[T]) Len() int {
	return len(p.clients)
}

func (p *ClientPool[T]) Get() T {
	return p.clients[p.index]
}

func (p *ClientPool[T]) MoveToNext() {
	p.index += 1
	if p.index == len(p.clients) {
		p.index = 0
	}
}

// NewL1BeaconClient returns a client for making requests to an L1 consensus layer node.
// Fallbacks are optional clients that will be used for fetching blobs. L1BeaconClient will rotate between
// the `cl` and the fallbacks whenever a client runs into an error while fetching blobs.
func NewL1BeaconClient(cl apis.BeaconClient, cfg L1BeaconClientConfig, fallbacks ...apis.BeaconClient) *L1BeaconClient {
	cs := append([]apis.BeaconClient{cl}, fallbacks...)
	return &L1BeaconClient{
		cl:   cl,
		pool: NewClientPool(cs...),
		cfg:  cfg,
	}
}

type TimeToSlotFn func(timestamp uint64) (uint64, error)

// getTimeToSlotFn returns a function that converts a timestamp to a slot number.
func (cl *L1BeaconClient) getTimeToSlotFn(ctx context.Context) (TimeToSlotFn, error) {
	cl.initLock.Lock()
	defer cl.initLock.Unlock()
	if cl.timeToSlotFn != nil {
		return cl.timeToSlotFn, nil
	}

	genesis, err := cl.cl.BeaconGenesis(ctx)
	if err != nil {
		return nil, err
	}

	config, err := cl.cl.ConfigSpec(ctx)
	if err != nil {
		return nil, err
	}

	genesisTime := uint64(genesis.Data.GenesisTime)
	secondsPerSlot := uint64(config.Data.SecondsPerSlot)
	if secondsPerSlot == 0 {
		return nil, fmt.Errorf("got bad value for seconds per slot: %v", config.Data.SecondsPerSlot)
	}
	cl.timeToSlotFn = func(timestamp uint64) (uint64, error) {
		if timestamp < genesisTime {
			return 0, fmt.Errorf("provided timestamp (%v) precedes genesis time (%v)", timestamp, genesisTime)
		}
		return (timestamp - genesisTime) / secondsPerSlot, nil
	}
	return cl.timeToSlotFn, nil
}

func (cl *L1BeaconClient) timeToSlot(ctx context.Context, timestamp uint64) (uint64, error) {
	slotFn, err := cl.getTimeToSlotFn(ctx)
	if err != nil {
		return 0, fmt.Errorf("get time to slot fn: %w", err)
	}
	slot, err := slotFn(timestamp)
	if err != nil {
		return 0, fmt.Errorf("convert timestamp %d to slot number: %w", timestamp, err)
	}
	return slot, nil
}

func (cl *L1BeaconClient) fetchBlobs(ctx context.Context, slot uint64, hashes []common.Hash) (eth.APIBeaconBlobsResponse, error) {
	var errs []error
	for i := 0; i < cl.pool.Len(); i++ {
		f := cl.pool.Get()
		resp, err := f.BeaconBlobs(ctx, slot, hashes)
		if err != nil {
			cl.pool.MoveToNext()
			errs = append(errs, err)
		} else {
			return resp, nil
		}
	}
	return eth.APIBeaconBlobsResponse{}, errors.Join(errs...)
}

// GetBlobsByHash fetches blobs that were confirmed at the given timestamp with the given versioned hashes.
// The order of the returned blobs will match the order of `hashes`. Confirms each
// blob's validity by recomputing the commitment and confirming the commitment
// hashes to the expected value. Returns error if any blob is found invalid.
func (cl *L1BeaconClient) GetBlobsByHash(ctx context.Context, time uint64, hashes []common.Hash) ([]*eth.Blob, error) {
	if len(hashes) == 0 {
		return []*eth.Blob{}, nil
	}
	slot, err := cl.timeToSlot(ctx, time)
	if err != nil {
		return nil, err
	}
	return cl.beaconBlobs(ctx, slot, hashes)
}

func (cl *L1BeaconClient) beaconBlobs(ctx context.Context, slot uint64, hashes []common.Hash) ([]*eth.Blob, error) {
	resp, err := cl.fetchBlobs(ctx, slot, hashes)
	if err != nil {
		return nil, fmt.Errorf("get blobs from beacon client: %w", err)
	}
	if len(resp.Data) != len(hashes) {
		return nil, fmt.Errorf("expected %d blobs but got %d", len(hashes), len(resp.Data))
	}
	// This function guarantees that the returned blobs will be ordered according to the provided
	// hashes. The BeaconBlobs call above has a different ordering. From the getBlobs spec:
	//   The returned blobs are ordered based on their kzg commitments in the block.
	// https://ethereum.github.io/beacon-APIs/beacon-node-oapi.yaml
	//
	// This loop
	//   1. verifies the integrity of each blob, and
	//   2. rearranges the blobs to match the order of the provided hashes.
	blobs := make([]*eth.Blob, len(hashes))
	for _, blob := range resp.Data {
		commitment, err := blob.ComputeKZGCommitment()
		if err != nil {
			return nil, fmt.Errorf("compute blob kzg commitment: %w", err)
		}
		got := eth.KZGToVersionedHash(commitment)
		idx := -1
		for i, h := range hashes {
			if got == h && blobs[i] == nil {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, fmt.Errorf("received a blob hash that does not match any expected hash: %s", got)
		}
		blobs[idx] = blob
	}
	return blobs, nil
}

// verifyBlob verifies that the blob data corresponds to the provided commitment.
// It recomputes the commitment from the blob data and checks it matches the expected commitment hash.
func verifyBlob(blob *eth.Blob, expectedCommitmentHash common.Hash) error {
	recomputedCommitment, err := blob.ComputeKZGCommitment()
	if err != nil {
		return fmt.Errorf("cannot compute KZG commitment for blob: %w", err)
	}
	recomputedCommitmentHash := eth.KZGToVersionedHash(recomputedCommitment)
	if recomputedCommitmentHash != expectedCommitmentHash {
		return fmt.Errorf("recomputed commitment %s does not match expected commitment %s", recomputedCommitmentHash, expectedCommitmentHash)
	}
	return nil
}

// GetVersion fetches the version of the Beacon-node.
func (cl *L1BeaconClient) GetVersion(ctx context.Context) (string, error) {
	return cl.cl.NodeVersion(ctx)
}
