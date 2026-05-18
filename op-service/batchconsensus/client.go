package batchconsensus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type ProofRequest struct {
	L1ChainID           string         `json:"l1_chain_id"`
	L2ChainID           string         `json:"l2_chain_id"`
	BatchInbox          common.Address `json:"batch_inbox"`
	Batcher             common.Address `json:"batcher"`
	BlobVersionedHashes []common.Hash  `json:"blob_versioned_hashes"`
}

type ProofResponse struct {
	Provider    string        `json:"provider,omitempty"`
	Certificate hexutil.Bytes `json:"certificate,omitempty"`
	Calldata    hexutil.Bytes `json:"calldata"`
}

const (
	ProviderCommonwarePOC        = "commonware-poc-secp256k1"
	ProviderCommonwareSimplexPOC = "commonware-simplex-poc-ed25519"
)

func NewProofRequest(l1ChainID, l2ChainID *big.Int, batchInbox common.Address, batcher common.Address, blobHashes []common.Hash) (ProofRequest, error) {
	if l1ChainID == nil {
		return ProofRequest{}, fmt.Errorf("missing L1 chain ID")
	}
	if l2ChainID == nil {
		return ProofRequest{}, fmt.Errorf("missing L2 chain ID")
	}
	return ProofRequest{
		L1ChainID:           l1ChainID.String(),
		L2ChainID:           l2ChainID.String(),
		BatchInbox:          batchInbox,
		Batcher:             batcher,
		BlobVersionedHashes: blobHashes,
	}, nil
}

func FetchProofCalldata(ctx context.Context, endpoint string, req ProofRequest) ([]byte, error) {
	resp, err := FetchProof(ctx, endpoint, req)
	if err != nil {
		return nil, err
	}
	return resp.Calldata, nil
}

func FetchProof(ctx context.Context, endpoint string, req ProofRequest) (ProofResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return ProofResponse{}, fmt.Errorf("marshal proof request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ProofResponse{}, fmt.Errorf("build proof request: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return ProofResponse{}, fmt.Errorf("send proof request: %w", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return ProofResponse{}, fmt.Errorf("proof sidecar returned status %d", httpResp.StatusCode)
	}
	var resp ProofResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return ProofResponse{}, fmt.Errorf("decode proof response: %w", err)
	}
	if len(resp.Calldata) == 0 {
		return ProofResponse{}, fmt.Errorf("proof sidecar returned empty calldata")
	}
	return resp, nil
}
