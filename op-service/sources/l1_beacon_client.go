package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto/kzg4844"

	"github.com/ethereum-optimism/optimism/op-service/client"
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

type BlobSidecar struct {
	BlockRoot     eth.Bytes32      `json:"block_root"`
	Slot          eth.Uint64String `json:"slot"`
	Blob          eth.Blob         `json:"blob"`
	Index         eth.Uint64String `json:"index"`
	KZGCommitment eth.Bytes48      `json:"kzg_commitment"`
	KZGProof      eth.Bytes48      `json:"kzg_proof"`
}

type APIGetBlobSidecarsResponse struct {
	Data []*BlobSidecar `json:"data"`
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

// BlobsByRefAndIndexedDataHashes fetches blobs that were confirmed in the given L1 block with the
// given indexed hashes. The order of the returned blobs will match the order of `dataHashes`.
func (cl *L1BeaconClient) BlobsByRefAndIndexedDataHashes(ctx context.Context, ref eth.L1BlockRef, dataHashes []eth.IndexedDataHash) ([]*eth.Blob, error) {
	slotFn, err := cl.GetTimeToSlotFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get time to slot function: %w", err)
	}
	slot := slotFn(ref.Time)

	builder := strings.Builder{}
	builder.WriteString("eth/v1/beacon/blob_sidecars/")
	builder.WriteString(strconv.FormatUint(slot, 10))
	for i := range dataHashes {
		if i == 0 {
			builder.WriteString("?indices=")
		} else {
			builder.WriteString("&indices=")
		}
		builder.WriteString(strconv.FormatUint(dataHashes[i].Index, 10))
	}

	var resp APIGetBlobSidecarsResponse
	if err := cl.apiReq(ctx, &resp, builder.String()); err != nil {
		return nil, fmt.Errorf("failed to fetch blob sidecars of slot %d (for block %s): %w", slot, ref, err)
	}
	if len(dataHashes) != len(resp.Data) {
		return nil, fmt.Errorf("expected %v sidecars but got %v", len(dataHashes), len(resp.Data))
	}

	out := make([]*eth.Blob, len(dataHashes))
	for i, ih := range dataHashes {
		// The beacon node api makes no guarantees on order of the returned blob sidecars, so
		// search for the sidecar that matches the current indexed hash to ensure blobs are
		// returned in the same order.
		var sidecar *BlobSidecar
		for _, sc := range resp.Data {
			if uint64(sc.Index) == ih.Index {
				sidecar = sc
				break
			}
		}
		if sidecar == nil {
			return nil, fmt.Errorf("no blob in response matches desired index: %v", ih.Index)
		}

		// make sure the blob's kzg commitment hashes to the expected value
		dataHash := eth.KZGToVersionedHash(kzg4844.Commitment(sidecar.KZGCommitment))
		if dataHash != ih.DataHash {
			return nil, fmt.Errorf("expected datahash %s for blob at index %d in block %s but got %s", ih.DataHash, ih.Index, ref, dataHash)
		}

		// confirm blob data is valid by verifying its proof against the commitment
		if err := eth.VerifyBlobProof(&sidecar.Blob, kzg4844.Commitment(sidecar.KZGCommitment), kzg4844.Proof(sidecar.KZGProof)); err != nil {
			return nil, fmt.Errorf("blob at index %v failed verification: %w", i, err)
		}
		out[i] = &sidecar.Blob
	}
	return out, nil
}
