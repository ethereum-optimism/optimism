package proxyd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

var errInvalidRPCParams = errors.New("invalid RPC params")

type RPCMethodHandler interface {
	CacheKey(req *RPCReq) string
	IsCacheable(req *RPCReq) (bool, error)
	RequiresUnconfirmedBlocks(ctx context.Context, req *RPCReq) (bool, error)
}

type StaticRPCMethodHandler struct {
	method string
}

func (s *StaticRPCMethodHandler) CacheKey(req *RPCReq) string {
	return fmt.Sprintf("method:%s", s.method)
}

func (s *StaticRPCMethodHandler) IsCacheable(*RPCReq) (bool, error) { return true, nil }
func (s *StaticRPCMethodHandler) RequiresUnconfirmedBlocks(context.Context, *RPCReq) (bool, error) {
	return false, nil
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

func (e *EthGetBlockByNumberMethod) IsCacheable(req *RPCReq) (bool, error) {
	blockNum, _, err := decodeGetBlockByNumberParams(req.Params)
	if err != nil {
		return false, err
	}
	return !isBlockDependentParam(blockNum), nil
}

func (e *EthGetBlockByNumberMethod) RequiresUnconfirmedBlocks(ctx context.Context, req *RPCReq) (bool, error) {
	curBlock, err := e.getLatestBlockNumFn(ctx)
	if err != nil {
		return false, err
	}
	blockInput, _, err := decodeGetBlockByNumberParams(req.Params)
	if err != nil {
		return false, err
	}
	if isBlockDependentParam(blockInput) {
		return true, nil
	}
	if blockInput == "earliest" {
		return false, nil
	}
	blockNum, err := decodeBlockInput(blockInput)
	if err != nil {
		return false, err
	}
	return curBlock <= blockNum+numBlockConfirmations, nil
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

func (e *EthGetBlockRangeMethod) IsCacheable(req *RPCReq) (bool, error) {
	start, end, _, err := decodeGetBlockRangeParams(req.Params)
	if err != nil {
		return false, err
	}
	return !isBlockDependentParam(start) && !isBlockDependentParam(end), nil
}

func (e *EthGetBlockRangeMethod) RequiresUnconfirmedBlocks(ctx context.Context, req *RPCReq) (bool, error) {
	curBlock, err := e.getLatestBlockNumFn(ctx)
	if err != nil {
		return false, err
	}

	start, end, _, err := decodeGetBlockRangeParams(req.Params)
	if err != nil {
		return false, err
	}
	if isBlockDependentParam(start) || isBlockDependentParam(end) {
		return true, nil
	}
	if start == "earliest" && end == "earliest" {
		return false, nil
	}

	if start != "earliest" {
		startNum, err := decodeBlockInput(start)
		if err != nil {
			return false, err
		}
		if curBlock <= startNum+numBlockConfirmations {
			return true, nil
		}
	}
	if end != "earliest" {
		endNum, err := decodeBlockInput(end)
		if err != nil {
			return false, err
		}
		if curBlock <= endNum+numBlockConfirmations {
			return true, nil
		}
	}
	return false, nil
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
		return "", false, errInvalidRPCParams
	}
	blockNum, ok := list[0].(string)
	if !ok {
		return "", false, errInvalidRPCParams
	}
	includeTx, ok := list[1].(bool)
	if !ok {
		return "", false, errInvalidRPCParams
	}
	if !validBlockInput(blockNum) {
		return "", false, errInvalidRPCParams
	}
	return blockNum, includeTx, nil
}

func decodeGetBlockRangeParams(params json.RawMessage) (string, string, bool, error) {
	var list []interface{}
	if err := json.Unmarshal(params, &list); err != nil {
		return "", "", false, err
	}
	if len(list) != 3 {
		return "", "", false, errInvalidRPCParams
	}
	startBlockNum, ok := list[0].(string)
	if !ok {
		return "", "", false, errInvalidRPCParams
	}
	endBlockNum, ok := list[1].(string)
	if !ok {
		return "", "", false, errInvalidRPCParams
	}
	includeTx, ok := list[2].(bool)
	if !ok {
		return "", "", false, errInvalidRPCParams
	}
	if !validBlockInput(startBlockNum) || !validBlockInput(endBlockNum) {
		return "", "", false, errInvalidRPCParams
	}
	return startBlockNum, endBlockNum, includeTx, nil
}

func decodeBlockInput(input string) (uint64, error) {
	return hexutil.DecodeUint64(input)
}

func validBlockInput(input string) bool {
	if input == "earliest" || input == "pending" || input == "latest" {
		return true
	}
	_, err := decodeBlockInput(input)
	return err == nil
}
