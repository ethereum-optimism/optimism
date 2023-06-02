package node

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/rpc"
	"github.com/ledgerwatch/log/v3"
)

type RPC interface {
	SetJWTAuth(jwtSecret *[32]byte) error
	SetHeader(key, value string)
	GetLatestBlock() (*Block, error)
	GetBlockByNumber(blockNumber *big.Int) (*Block, error)
	GetTransactionByHash(txHash *common.Hash) (types.Transaction, error)
	ForkchoiceUpdateV1(fc *commands.ForkChoiceState, attributes *commands.PayloadAttributes) (*ForkchoiceUpdatedResult, error)
	GetPayloadV1(payloadID *PayloadID) (*commands.ExecutionPayload, error)
	NewPayloadV1(executionPayload *commands.ExecutionPayload) (*PayloadStatusV1, error)
}

type BackendRPC struct {
	client  *rpc.Client
	timeout time.Duration
	jwtAuth *JWTAuth
}

func NewRPC(endpoint string, timeout time.Duration, logger log.Logger) (*BackendRPC, error) {
	rpc, err := rpc.Dial(endpoint, logger)
	if err != nil {
		return nil, err
	}
	return &BackendRPC{
		client:  rpc,
		timeout: timeout,
		jwtAuth: &JWTAuth{},
	}, nil
}

func (r *BackendRPC) SetJWTAuth(jwtSecret *[32]byte) error {
	if err := r.jwtAuth.NewJWTAuth(r, jwtSecret); err != nil {
		return fmt.Errorf("failed to set jwt auth: %w", err)
	}
	return nil
}

func (r *BackendRPC) SetHeader(key, value string) {
	r.client.SetHeader(key, value)
}

func (r *BackendRPC) RefreshJWTAuth() error {
	if r.jwtAuth.JWTSecret == nil {
		return nil
	}
	if err := r.jwtAuth.RefreshJWTAuth(r); err != nil {
		return fmt.Errorf("failed to refresh jwt auth: %w", err)
	}
	return nil
}

func (r *BackendRPC) GetLatestBlock() (*Block, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}
	var block *Block
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &block, "eth_getBlockByNumber", "latest", false); err != nil {
		return nil, err
	}
	return block, nil
}

func (r *BackendRPC) GetBlockByNumber(blockNumber *big.Int) (*Block, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}
	var block *Block
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &block, "eth_getBlockByNumber", hexutil.EncodeBig(blockNumber), false); err != nil {
		return nil, err
	}
	return block, nil
}

func (r *BackendRPC) GetTransactionByHash(txHash *common.Hash) (types.Transaction, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}
	var tx *types.LegacyTx
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &tx, "eth_getTransactionByHash", txHash); err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *BackendRPC) ForkchoiceUpdateV1(fc *commands.ForkChoiceState, attributes *commands.PayloadAttributes) (*ForkchoiceUpdatedResult, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}
	var result ForkchoiceUpdatedResult
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &result, "engine_forkchoiceUpdatedV1", fc, attributes); err != nil {
		return nil, fmt.Errorf("failed to call engine_forkchoiceUpdatedV1: %w", err)
	}
	return &result, nil
}

func (r *BackendRPC) GetPayloadV1(payloadID *PayloadID) (*commands.ExecutionPayload, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}
	var result *commands.ExecutionPayload
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &result, "engine_getPayloadV1", payloadID); err != nil {
		return nil, fmt.Errorf("Failed to obtain new payloadId: %v", err)
	}
	return result, nil
}

func (r *BackendRPC) NewPayloadV1(executionPayload *commands.ExecutionPayload) (*PayloadStatusV1, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}
	var result *PayloadStatusV1
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &result, "engine_newPayloadV1", executionPayload); err != nil {
		return nil, fmt.Errorf("Failed to execute new payloadId: %v", err)
	}
	return result, nil
}
