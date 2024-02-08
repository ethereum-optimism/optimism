package node

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	ethereum "github.com/ledgerwatch/erigon"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutil"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/eth/tracers"
	"github.com/ledgerwatch/erigon/rpc"
	"github.com/ledgerwatch/erigon/turbo/engineapi/engine_types"
	"github.com/ledgerwatch/log/v3"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . RPC
type RPC interface {
	SetJWTAuth(jwtSecret *[32]byte) error
	SetHeader(key, value string)
	GetBlockNumber() (*big.Int, error)
	GetLatestBlock() (*Block, error)
	GetBlockByNumber(blockNumber *big.Int) (*Block, error)
	GetTransactionByHash(txHash *common.Hash) (types.Transaction, error)
	ForkchoiceUpdateV1(fc *engine_types.ForkChoiceState, attributes *engine_types.PayloadAttributes) (*ForkchoiceUpdatedResult, error)
	GetPayloadV1(payloadID *PayloadID) (*engine_types.ExecutionPayload, error)
	NewPayloadV1(executionPayload *engine_types.ExecutionPayload) (*PayloadStatusV1, error)
	TraceTransaction(txHash *common.Hash) (*TraceTransaction, error)
	GetLogs(filter *ethereum.FilterQuery) ([]*types.Log, error)
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

func (r *BackendRPC) GetBlockNumber() (*big.Int, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}
	var blockNumber *hexutil.Big
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &blockNumber, "eth_blockNumber"); err != nil {
		return nil, err
	}
	return (*big.Int)(blockNumber), nil
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

func (r *BackendRPC) ForkchoiceUpdateV1(fc *engine_types.ForkChoiceState, attributes *engine_types.PayloadAttributes) (*ForkchoiceUpdatedResult, error) {
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

func (r *BackendRPC) GetPayloadV1(payloadID *PayloadID) (*engine_types.ExecutionPayload, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}
	var result *engine_types.ExecutionPayload
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &result, "engine_getPayloadV1", payloadID); err != nil {
		return nil, fmt.Errorf("Failed to obtain new payloadId: %w", err)
	}
	return result, nil
}

func (r *BackendRPC) NewPayloadV1(executionPayload *engine_types.ExecutionPayload) (*PayloadStatusV1, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}
	var result *PayloadStatusV1
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &result, "engine_newPayloadV1", executionPayload); err != nil {
		return nil, fmt.Errorf("Failed to execute new payloadId: %w", err)
	}
	return result, nil
}

func (r *BackendRPC) TraceTransaction(txHash *common.Hash) (*TraceTransaction, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}

	var traceResult *TraceTransaction

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	tracerType := "callTracer"
	tracerTimeout := r.timeout.String()
	config := &tracers.TraceConfig{
		Tracer:  &tracerType,
		Timeout: &tracerTimeout,
	}

	if err := r.client.CallContext(ctx, &traceResult, "debug_traceTransaction", txHash, config); err != nil {
		return nil, err
	}
	return traceResult, nil
}

func (r *BackendRPC) GetLogs(filter *ethereum.FilterQuery) ([]*types.Log, error) {
	if err := r.RefreshJWTAuth(); err != nil {
		return nil, err
	}

	filterArg, err := toFilterArg(filter)
	if err != nil {
		return nil, err
	}

	var logs []*types.Log
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	if err := r.client.CallContext(ctx, &logs, "eth_getLogs", filterArg); err != nil {
		return nil, err
	}
	return logs, nil
}

func toFilterArg(q *ethereum.FilterQuery) (interface{}, error) {
	arg := map[string]interface{}{
		"address": q.Addresses,
		"topics":  q.Topics,
	}
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, errors.New("cannot specify both BlockHash and FromBlock/ToBlock")
		}
	} else {
		if q.FromBlock == nil || q.FromBlock.Sign() < 0 {
			arg["fromBlock"] = "0x0"
		} else {
			arg["fromBlock"] = hexutil.EncodeBig(q.FromBlock)
		}

		if q.ToBlock == nil || q.ToBlock.Sign() < 0 {
			arg["toBlock"] = "0x0"
		} else {
			arg["toBlock"] = hexutil.EncodeBig(q.ToBlock)
		}
	}
	return arg, nil
}
