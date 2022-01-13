package proxyd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type RPCMethodHandler interface {
	CacheKey(req *RPCReq) string
	IsCacheable(req *RPCReq) bool
	RequiresUnconfirmedBlocks(ctx context.Context, req *RPCReq) bool
}

type StaticRPCMethodHandler struct {
	method string
}

func (s *StaticRPCMethodHandler) CacheKey(req *RPCReq) string {
	return fmt.Sprintf("method:%s", s.method)
}

func (s *StaticRPCMethodHandler) IsCacheable(*RPCReq) bool { return true }
func (s *StaticRPCMethodHandler) RequiresUnconfirmedBlocks(context.Context, *RPCReq) bool {
	return false
}

type EthGetBlockByNumberMethod struct {
	getLatestBlockNumFn GetLatestBlockNumFn
}

func (e *EthGetBlockByNumberMethod) CacheKey(req *RPCReq) string {
	input, includeTx, err := decodeGetBlockByNumberParams(req.Params)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("method:eth_getBlockByNumber:%s:%t", input, includeTx)
}

func (e *EthGetBlockByNumberMethod) IsCacheable(req *RPCReq) bool {
	blockNum, _, err := decodeGetBlockByNumberParams(req.Params)
	if err != nil {
		return false
	}
	return !isBlockDependentParam(blockNum)
}

func (e *EthGetBlockByNumberMethod) RequiresUnconfirmedBlocks(ctx context.Context, req *RPCReq) bool {
	curBlock, err := e.getLatestBlockNumFn(ctx)
	if err != nil {
		return false
	}
	blockInput, _, err := decodeGetBlockByNumberParams(req.Params)
	if err != nil {
		return false
	}
	if isBlockDependentParam(blockInput) {
		return true
	}
	if blockInput == "earliest" {
		return false
	}
	blockNum, err := decodeBlockInput(blockInput)
	if err != nil {
		return false
	}
	return curBlock <= blockNum+numBlockConfirmations
}

type EthGetBlockRangeMethod struct {
	getLatestBlockNumFn GetLatestBlockNumFn
}

func (e *EthGetBlockRangeMethod) CacheKey(req *RPCReq) string {
	start, end, includeTx, err := decodeGetBlockRangeParams(req.Params)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("method:eth_getBlockRange:%s:%s:%t", start, end, includeTx)
}

func (e *EthGetBlockRangeMethod) IsCacheable(req *RPCReq) bool {
	start, end, _, err := decodeGetBlockRangeParams(req.Params)
	if err != nil {
		return false
	}
	return !isBlockDependentParam(start) && !isBlockDependentParam(end)
}

func (e *EthGetBlockRangeMethod) RequiresUnconfirmedBlocks(ctx context.Context, req *RPCReq) bool {
	curBlock, err := e.getLatestBlockNumFn(ctx)
	if err != nil {
		return false
	}

	start, end, _, err := decodeGetBlockRangeParams(req.Params)
	if err != nil {
		return false
	}
	if isBlockDependentParam(start) || isBlockDependentParam(end) {
		return true
	}
	if start == "earliest" && end == "earliest" {
		return false
	}
	startNum, err := decodeBlockInput(start)
	if err != nil {
		return false
	}
	endNum, err := decodeBlockInput(end)
	if err != nil {
		return false
	}
	return curBlock <= startNum+numBlockConfirmations || curBlock <= endNum+numBlockConfirmations
}

func isBlockDependentParam(s string) bool {
	return s == "latest" || s == "pending"
}

func decodeGetBlockByNumberParams(params json.RawMessage) (string, bool, error) {
	var list []interface{}
	if err := json.Unmarshal(params, &list); err != nil {
		return "", false, err
	}
	if len(list) != 2 {
		return "", false, errInvalidBlockByNumberParams
	}
	blockNum, ok := list[0].(string)
	if !ok {
		return "", false, errInvalidBlockByNumberParams
	}
	includeTx, ok := list[1].(bool)
	if !ok {
		return "", false, errInvalidBlockByNumberParams
	}
	return blockNum, includeTx, nil
}

func decodeGetBlockRangeParams(params json.RawMessage) (string, string, bool, error) {
	var list []interface{}
	if err := json.Unmarshal(params, &list); err != nil {
		return "", "", false, err
	}
	if len(list) != 3 {
		return "", "", false, errInvalidBlockByNumberParams
	}
	startBlockNum, ok := list[0].(string)
	if !ok {
		return "", "", false, errInvalidBlockByNumberParams
	}
	endBlockNum, ok := list[1].(string)
	if !ok {
		return "", "", false, errInvalidBlockByNumberParams
	}
	includeTx, ok := list[2].(bool)
	if !ok {
		return "", "", false, errInvalidBlockByNumberParams
	}
	return startBlockNum, endBlockNum, includeTx, nil
}

func decodeBlockInput(input string) (uint64, error) {
	return hexutil.DecodeUint64(input)
}
