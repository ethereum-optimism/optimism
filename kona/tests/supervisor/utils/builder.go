package utils

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	opeth "github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type rpcRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type rpcResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type TestBlockBuilderConfig struct {
	safeBlockDistance      uint64
	finalizedBlockDistance uint64

	GethRPC string

	EngineRPC string
	JWTSecret string
}

type TestBlockBuilder struct {
	t devtest.CommonT

	withdrawalsIndex uint64

	cfg       TestBlockBuilderConfig
	ethClient *ethclient.Client
}

func NewTestBlockBuilder(t devtest.CommonT, cfg TestBlockBuilderConfig) *TestBlockBuilder {
	ethClient, err := ethclient.Dial(cfg.GethRPC)
	if err != nil {
		t.Errorf("failed to connect to Geth RPC: %v", err)
		return nil
	}

	return &TestBlockBuilder{t, 1001, cfg, ethClient}
}

func createJWT(secret []byte) (string, error) {
	// try to decode hex string (support "0x..." or plain hex), fall back to raw bytes
	secretStr := string(secret)
	secretStr = strings.TrimPrefix(secretStr, "0x")
	key, err := hex.DecodeString(secretStr)
	if err != nil {
		key = secret
	}

	// typos:disable
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	// typos:enable
	payload := fmt.Sprintf(`{"iat":%d}`, time.Now().Unix())
	payloadEnc := base64.RawURLEncoding.EncodeToString([]byte(payload))
	toSign := header + "." + payloadEnc
	h := hmac.New(sha256.New, key)
	h.Write([]byte(toSign))
	sig := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return toSign + "." + sig, nil
}

func (s *TestBlockBuilder) rpcCallWithJWT(url, method string, params interface{}) (*rpcResponse, error) {
	reqBody, _ := json.Marshal(rpcRequest{Jsonrpc: "2.0", Method: method, Params: params, ID: 1})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Create JWT token
	jwtToken, err := createJWT([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	// Non-200 -> surface the body for debugging
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response %d from %s: %s", resp.StatusCode, url, string(bodyBytes))
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		// include raw body to help debugging the bad payload
		return nil, fmt.Errorf("failed to parse RPC response: %w; body: %s", err, string(bodyBytes))
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}
	return &rpcResp, nil
}

func (s *TestBlockBuilder) rpcCall(url, method string, params interface{}) (*rpcResponse, error) {
	return s.rpcCallWithJWT(url, method, params)
}

func (s *TestBlockBuilder) rewindTo(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	s.t.Logf("Rewinding to block %s", blockHash.Hex())

	block, err := s.ethClient.BlockByHash(ctx, blockHash)
	if err != nil {
		s.t.Errorf("failed to fetch block by hash %s: %v", blockHash.Hex(), err)
		return nil, fmt.Errorf("failed to fetch block by hash: %v", err)
	}

	// Attempt rewind using debug_setHead
	_, err = s.rpcCall(s.cfg.GethRPC, "debug_setHead", []interface{}{fmt.Sprintf("0x%x", block.NumberU64())})
	if err != nil {
		s.t.Errorf("failed to rewind to block %s: %v", blockHash.Hex(), err)
		return nil, fmt.Errorf("rewind failed: %v", err)
	}

	// Confirm head matches requested parent
	head, err := s.ethClient.BlockByNumber(ctx, big.NewInt(int64(rpc.LatestBlockNumber)))
	if err != nil {
		s.t.Errorf("failed to fetch latest block: %v", err)
		return nil, fmt.Errorf("failed to fetch latest block: %v", err)
	}

	if head.Hash() != blockHash {
		s.t.Errorf("head mismatch after rewind: expected %s, got %s", blockHash.Hex(), head.Hash().Hex())
		return nil, fmt.Errorf("head mismatch after rewind")
	}

	s.t.Logf("Successfully rewound to block %s", blockHash.Hex())
	return block, nil
}

func (s *TestBlockBuilder) BuildBlock(ctx context.Context, parentHash *common.Hash) {
	var head *types.Block
	var err error
	if parentHash != nil {
		head, err = s.rewindTo(ctx, *parentHash)
		if err != nil {
			s.t.Errorf("failed to rewind to parent block: %v", err)
			return
		}
	} else {
		head, err = s.ethClient.BlockByNumber(ctx, big.NewInt(int64(rpc.LatestBlockNumber)))
		if err != nil {
			s.t.Errorf("failed to fetch latest block: %v", err)
			return
		}
	}

	finalizedBlock, _ := s.ethClient.BlockByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	if finalizedBlock == nil {
		// set sb to genesis if safe block is not set
		finalizedBlock, err = s.ethClient.BlockByNumber(ctx, big.NewInt(0))
		if err != nil {
			s.t.Errorf("failed to fetch genesis block: %v", err)
			return
		}
	}

	// progress finalised block
	if head.NumberU64() > uint64(s.cfg.finalizedBlockDistance) {
		finalizedBlock, err = s.ethClient.BlockByNumber(ctx, big.NewInt(int64(head.NumberU64()-s.cfg.finalizedBlockDistance)))
		if err != nil {
			s.t.Errorf("failed to fetch safe block: %v", err)
			return
		}
	}

	safeBlock, _ := s.ethClient.BlockByNumber(ctx, big.NewInt(rpc.SafeBlockNumber.Int64()))
	if safeBlock == nil {
		safeBlock = finalizedBlock
	}

	// progress safe block
	if head.NumberU64() > uint64(s.cfg.safeBlockDistance) {
		safeBlock, err = s.ethClient.BlockByNumber(ctx, big.NewInt(int64(head.NumberU64()-s.cfg.safeBlockDistance)))
		if err != nil {
			s.t.Errorf("failed to fetch safe block: %v", err)
			return
		}
	}

	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash:      head.Hash(),
		SafeBlockHash:      safeBlock.Hash(),
		FinalizedBlockHash: finalizedBlock.Hash(),
	}

	newBlockTimestamp := head.Time() + 6
	nonce := time.Now().UnixNano()
	var nonceBytes [8]byte
	binary.LittleEndian.PutUint64(nonceBytes[:], uint64(nonce))
	randomHash := crypto.Keccak256Hash(nonceBytes[:])
	payloadAttrs := engine.PayloadAttributes{
		Timestamp:             uint64(newBlockTimestamp),
		Random:                randomHash,
		SuggestedFeeRecipient: head.Coinbase(),
		Withdrawals:           randomWithdrawals(s.withdrawalsIndex),
		BeaconRoot:            fakeBeaconBlockRoot(uint64(head.Time())),
	}

	// Start payload build
	fcResp, err := s.rpcCallWithJWT(s.cfg.EngineRPC, "engine_forkchoiceUpdatedV3",
		[]interface{}{fcState, payloadAttrs})
	if err != nil {
		s.t.Errorf("forkchoiceUpdated failed: %v", err)
		return
	}

	var fcResult engine.ForkChoiceResponse
	json.Unmarshal(fcResp.Result, &fcResult)
	if fcResult.PayloadStatus.Status != "VALID" && fcResult.PayloadStatus.Status != "SYNCING" {
		s.t.Errorf("forkchoiceUpdated returned invalid status: %s", fcResult.PayloadStatus.Status)
		return
	}

	if fcResult.PayloadID == nil {
		s.t.Errorf("forkchoiceUpdated did not return a payload ID")
		return
	}

	time.Sleep(150 * time.Millisecond)

	// Get payload
	plResp, err := s.rpcCallWithJWT(s.cfg.EngineRPC, "engine_getPayloadV3", []interface{}{fcResult.PayloadID})
	if err != nil {
		s.t.Errorf("getPayload failed: %v", err)
		return
	}

	var envelope engine.ExecutionPayloadEnvelope
	json.Unmarshal(plResp.Result, &envelope)
	if envelope.ExecutionPayload == nil {
		s.t.Errorf("getPayload returned empty execution payload")
		return
	}

	blobHashes := make([]common.Hash, 0)
	if envelope.BlobsBundle != nil {
		for _, commitment := range envelope.BlobsBundle.Commitments {
			if len(commitment) != 48 {
				break
			}
			blobHashes = append(blobHashes, opeth.KZGToVersionedHash(*(*[48]byte)(commitment)))
		}
		if len(blobHashes) != len(envelope.BlobsBundle.Commitments) {
			s.t.Errorf("blob hashes length mismatch: expected %d, got %d", len(envelope.BlobsBundle.Commitments), len(blobHashes))
			return
		}
	}

	// Insert
	newPayloadResp, err := s.rpcCallWithJWT(s.cfg.EngineRPC, "engine_newPayloadV3", []interface{}{envelope.ExecutionPayload, blobHashes, payloadAttrs.BeaconRoot})
	if err != nil {
		s.t.Errorf("newPayload failed: %v", err)
		return
	}

	var npRes engine.PayloadStatusV1
	json.Unmarshal(newPayloadResp.Result, &npRes)
	if npRes.Status != "VALID" && npRes.Status != "ACCEPTED" {
		s.t.Errorf("newPayload returned invalid status: %s", npRes.Status)
		return
	}

	// Update forkchoice
	updateFc := engine.ForkchoiceStateV1{
		HeadBlockHash:      envelope.ExecutionPayload.BlockHash,
		SafeBlockHash:      safeBlock.Hash(),
		FinalizedBlockHash: finalizedBlock.Hash(),
	}
	_, err = s.rpcCallWithJWT(s.cfg.EngineRPC, "engine_forkchoiceUpdatedV3", []interface{}{updateFc, nil})
	if err != nil {
		s.t.Errorf("forkchoiceUpdated failed after newPayload: %v", err)
		return
	}

	s.withdrawalsIndex += uint64(len(envelope.ExecutionPayload.Withdrawals))

	s.t.Logf("Successfully built block %s:%d at timestamp %d", envelope.ExecutionPayload.BlockHash.Hex(), envelope.ExecutionPayload.Number, newBlockTimestamp)
}

func fakeBeaconBlockRoot(time uint64) *common.Hash {
	var dat [8]byte
	binary.LittleEndian.PutUint64(dat[:], time)
	hash := crypto.Keccak256Hash(dat[:])
	return &hash
}

func randomWithdrawals(startIndex uint64) []*types.Withdrawal {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	withdrawals := make([]*types.Withdrawal, r.Intn(4))
	for i := 0; i < len(withdrawals); i++ {
		withdrawals[i] = &types.Withdrawal{
			Index:     startIndex + uint64(i),
			Validator: r.Uint64() % 100_000_000, // 100 million fake validators
			Address:   testutils.RandomAddress(r),
			Amount:    uint64(r.Intn(50_000_000_000) + 1),
		}
	}
	return withdrawals
}
