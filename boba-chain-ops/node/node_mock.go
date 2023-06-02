package node

import (
	"errors"
	"math/big"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/erigon/core/types"
)

var (
	ErrGetLatestBlock = errors.New("GetLatestBlock mock rpc error")
	ErrGetBlockByNum  = errors.New("GetBlockByNumber mock rpc error")
	ErrGetTxByHash    = errors.New("GetTransactionByHash mock rpc error")
	ErrForkchoice     = errors.New("ForkchoiceUpdateV1 mock rpc error")
	ErrGetPayload     = errors.New("GetPayloadV1 mock rpc error")
	ErrNewPayload     = errors.New("NewPayloadV1 mock rpc error")
)

type MockRPC struct {
	response map[string]interface{}
}

func (r *MockRPC) SetResponse(key string, value interface{}) {
	if r.response == nil {
		r.response = make(map[string]interface{})
	}
	r.response[key] = value
}

func (r *MockRPC) SetJWTAuth(jwtSecret *[32]byte) error {
	return nil
}

func (r *MockRPC) SetHeader(key, value string) {}

func (r *MockRPC) GetLatestBlock() (*Block, error) {
	block, ok := r.response["GetLatestBlock"].(*Block)
	if !ok {
		return nil, ErrGetLatestBlock
	}
	return block, nil
}

func (r *MockRPC) GetBlockByNumber(blockNumber *big.Int) (*Block, error) {
	block, ok := r.response["GetBlockByNumber"].(*Block)
	if !ok {
		return nil, ErrGetBlockByNum
	}
	return block, nil
}

func (r *MockRPC) GetTransactionByHash(txHash *common.Hash) (types.Transaction, error) {
	tx, ok := r.response["GetTransactionByHash"].(types.Transaction)
	if !ok {
		return &types.LegacyTx{}, ErrGetTxByHash
	}
	return tx, nil
}

func (r *MockRPC) ForkchoiceUpdateV1(fc *commands.ForkChoiceState, attributes *commands.PayloadAttributes) (*ForkchoiceUpdatedResult, error) {
	forkchoiceUpdatedResult, ok := r.response["ForkchoiceUpdateV1"].(*ForkchoiceUpdatedResult)
	if !ok {
		return nil, ErrForkchoice
	}
	return forkchoiceUpdatedResult, nil
}

func (r *MockRPC) GetPayloadV1(payloadID *PayloadID) (*commands.ExecutionPayload, error) {
	executionPayload, ok := r.response["GetPayloadV1"].(*commands.ExecutionPayload)
	if !ok {
		return nil, ErrGetPayload
	}
	return executionPayload, nil
}

func (r *MockRPC) NewPayloadV1(executionPayload *commands.ExecutionPayload) (*PayloadStatusV1, error) {
	payloadStatusV1, ok := r.response["NewPayloadV1"].(*PayloadStatusV1)
	if !ok {
		return nil, ErrNewPayload
	}
	return payloadStatusV1, nil
}
