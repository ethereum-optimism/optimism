package proxyd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	errInvalidRPCParams = errors.New("invalid RPC params")
)

type RPCMethodHandler interface {
	GetRPCMethod(context.Context, *RPCReq) (*RPCRes, error)
	PutRPCMethod(context.Context, *RPCReq, *RPCRes) error
}

type StaticMethodHandler struct {
	cache interface{}
	m     sync.RWMutex
}

func (e *StaticMethodHandler) GetRPCMethod(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	e.m.RLock()
	cache := e.cache
	e.m.RUnlock()

	if cache == nil {
		return nil, nil
	}
	return &RPCRes{
		JSONRPC: req.JSONRPC,
		Result:  cache,
		ID:      req.ID,
	}, nil
}

func (e *StaticMethodHandler) PutRPCMethod(ctx context.Context, req *RPCReq, res *RPCRes) error {
	e.m.Lock()
	if e.cache == nil {
		e.cache = res.Result
	}
	e.m.Unlock()
	return nil
}

type EthGetBlockByNumberMethodHandler struct {
	cache                 Cache
	getLatestBlockNumFn   GetLatestBlockNumFn
	numBlockConfirmations int
}

func (e *EthGetBlockByNumberMethodHandler) cacheKey(req *RPCReq) string {
	input, includeTx, err := decodeGetBlockByNumberParams(req.Params)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("method:eth_getBlockByNumber:%s:%t", input, includeTx)
}

func (e *EthGetBlockByNumberMethodHandler) cacheable(req *RPCReq) (bool, error) {
	blockNum, _, err := decodeGetBlockByNumberParams(req.Params)
	if err != nil {
		return false, err
	}
	return !isBlockDependentParam(blockNum), nil
}

func (e *EthGetBlockByNumberMethodHandler) GetRPCMethod(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	if ok, err := e.cacheable(req); !ok || err != nil {
		return nil, err
	}
	key := e.cacheKey(req)
	return getImmutableRPCResponse(ctx, e.cache, key, req)
}

func (e *EthGetBlockByNumberMethodHandler) PutRPCMethod(ctx context.Context, req *RPCReq, res *RPCRes) error {
	if ok, err := e.cacheable(req); !ok || err != nil {
		return err
	}

	blockInput, _, err := decodeGetBlockByNumberParams(req.Params)
	if err != nil {
		return err
	}
	if isBlockDependentParam(blockInput) {
		return nil
	}
	if blockInput != "earliest" {
		curBlock, err := e.getLatestBlockNumFn(ctx)
		if err != nil {
			return err
		}
		blockNum, err := decodeBlockInput(blockInput)
		if err != nil {
			return err
		}
		if curBlock <= blockNum+uint64(e.numBlockConfirmations) {
			return nil
		}
	}

	key := e.cacheKey(req)
	return putImmutableRPCResponse(ctx, e.cache, key, req, res)
}

type EthGetBlockRangeMethodHandler struct {
	cache                 Cache
	getLatestBlockNumFn   GetLatestBlockNumFn
	numBlockConfirmations int
}

func (e *EthGetBlockRangeMethodHandler) cacheKey(req *RPCReq) string {
	start, end, includeTx, err := decodeGetBlockRangeParams(req.Params)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("method:eth_getBlockRange:%s:%s:%t", start, end, includeTx)
}

func (e *EthGetBlockRangeMethodHandler) cacheable(req *RPCReq) (bool, error) {
	start, end, _, err := decodeGetBlockRangeParams(req.Params)
	if err != nil {
		return false, err
	}
	return !isBlockDependentParam(start) && !isBlockDependentParam(end), nil
}

func (e *EthGetBlockRangeMethodHandler) GetRPCMethod(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	if ok, err := e.cacheable(req); !ok || err != nil {
		return nil, err
	}

	key := e.cacheKey(req)
	return getImmutableRPCResponse(ctx, e.cache, key, req)
}

func (e *EthGetBlockRangeMethodHandler) PutRPCMethod(ctx context.Context, req *RPCReq, res *RPCRes) error {
	if ok, err := e.cacheable(req); !ok || err != nil {
		return err
	}

	start, end, _, err := decodeGetBlockRangeParams(req.Params)
	if err != nil {
		return err
	}
	curBlock, err := e.getLatestBlockNumFn(ctx)
	if err != nil {
		return err
	}
	if start != "earliest" {
		startNum, err := decodeBlockInput(start)
		if err != nil {
			return err
		}
		if curBlock <= startNum+uint64(e.numBlockConfirmations) {
			return nil
		}
	}
	if end != "earliest" {
		endNum, err := decodeBlockInput(end)
		if err != nil {
			return err
		}
		if curBlock <= endNum+uint64(e.numBlockConfirmations) {
			return nil
		}
	}

	key := e.cacheKey(req)
	return putImmutableRPCResponse(ctx, e.cache, key, req, res)
}

type EthCallMethodHandler struct {
	cache                 Cache
	getLatestBlockNumFn   GetLatestBlockNumFn
	numBlockConfirmations int
}

func (e *EthCallMethodHandler) cacheable(params *ethCallParams, blockTag string) bool {
	if isBlockDependentParam(blockTag) {
		return false
	}
	if params.From != "" || params.Gas != "" {
		return false
	}
	if params.Value != "" && params.Value != "0x0" {
		return false
	}
	return true
}

func (e *EthCallMethodHandler) cacheKey(params *ethCallParams, blockTag string) string {
	keyParams := fmt.Sprintf("%s:%s:%s", params.To, params.Data, blockTag)
	return fmt.Sprintf("method:eth_call:%s", keyParams)
}

func (e *EthCallMethodHandler) GetRPCMethod(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	params, blockTag, err := decodeEthCallParams(req)
	if err != nil {
		return nil, err
	}
	if !e.cacheable(params, blockTag) {
		return nil, nil
	}
	key := e.cacheKey(params, blockTag)
	return getImmutableRPCResponse(ctx, e.cache, key, req)
}

func (e *EthCallMethodHandler) PutRPCMethod(ctx context.Context, req *RPCReq, res *RPCRes) error {
	params, blockTag, err := decodeEthCallParams(req)
	if err != nil {
		return err
	}
	if !e.cacheable(params, blockTag) {
		return nil
	}

	if blockTag != "earliest" {
		curBlock, err := e.getLatestBlockNumFn(ctx)
		if err != nil {
			return err
		}
		blockNum, err := decodeBlockInput(blockTag)
		if err != nil {
			return err
		}
		if curBlock <= blockNum+uint64(e.numBlockConfirmations) {
			return nil
		}
	}

	key := e.cacheKey(params, blockTag)
	return putImmutableRPCResponse(ctx, e.cache, key, req, res)
}

type EthBlockNumberMethodHandler struct {
	getLatestBlockNumFn GetLatestBlockNumFn
}

func (e *EthBlockNumberMethodHandler) GetRPCMethod(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	blockNum, err := e.getLatestBlockNumFn(ctx)
	if err != nil {
		return nil, err
	}
	return makeRPCRes(req, hexutil.EncodeUint64(blockNum)), nil
}

func (e *EthBlockNumberMethodHandler) PutRPCMethod(context.Context, *RPCReq, *RPCRes) error {
	return nil
}

type EthGasPriceMethodHandler struct {
	getLatestGasPrice GetLatestGasPriceFn
}

func (e *EthGasPriceMethodHandler) GetRPCMethod(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	gasPrice, err := e.getLatestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	return makeRPCRes(req, hexutil.EncodeUint64(gasPrice)), nil
}

func (e *EthGasPriceMethodHandler) PutRPCMethod(context.Context, *RPCReq, *RPCRes) error {
	return nil
}

func isBlockDependentParam(s string) bool {
	return s == "latest" ||
		s == "pending" ||
		s == "finalized" ||
		s == "safe"
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

type ethCallParams struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Value    string `json:"value"`
	Data     string `json:"data"`
}

func decodeEthCallParams(req *RPCReq) (*ethCallParams, string, error) {
	var input []json.RawMessage
	if err := json.Unmarshal(req.Params, &input); err != nil {
		return nil, "", err
	}
	if len(input) != 2 {
		return nil, "", fmt.Errorf("invalid eth_call parameters")
	}
	params := new(ethCallParams)
	if err := json.Unmarshal(input[0], params); err != nil {
		return nil, "", err
	}
	var blockTag string
	if err := json.Unmarshal(input[1], &blockTag); err != nil {
		return nil, "", err
	}
	return params, blockTag, nil
}

func validBlockInput(input string) bool {
	if input == "earliest" ||
		input == "latest" ||
		input == "pending" ||
		input == "finalized" ||
		input == "safe" {
		return true
	}
	_, err := decodeBlockInput(input)
	return err == nil
}

func makeRPCRes(req *RPCReq, result interface{}) *RPCRes {
	return &RPCRes{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

func getImmutableRPCResponse(ctx context.Context, cache Cache, key string, req *RPCReq) (*RPCRes, error) {
	val, err := cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if val == "" {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return &RPCRes{
		JSONRPC: req.JSONRPC,
		Result:  result,
		ID:      req.ID,
	}, nil
}

func putImmutableRPCResponse(ctx context.Context, cache Cache, key string, req *RPCReq, res *RPCRes) error {
	if key == "" {
		return nil
	}
	val := mustMarshalJSON(res.Result)
	return cache.Put(ctx, key, string(val))
}
