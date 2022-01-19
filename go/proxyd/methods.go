package proxyd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/snappy"
)

var (
	cacheTTL = 5 * time.Second

	errInvalidRPCParams = errors.New("invalid RPC params")
)

type RPCMethodHandler interface {
	GetRPCMethod(context.Context, *RPCReq) (*RPCRes, error)
	PutRPCMethod(context.Context, *RPCReq, *RPCRes, uint64) error
}

type StaticMethodHandler struct {
	cache *RPCRes
	m     sync.RWMutex
}

func (e *StaticMethodHandler) GetRPCMethod(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	e.m.RLock()
	cache := e.cache
	e.m.RUnlock()

	if cache != nil {
		cache = copyRes(cache)
		cache.ID = req.ID
	}
	return cache, nil
}

func (e *StaticMethodHandler) PutRPCMethod(ctx context.Context, req *RPCReq, res *RPCRes, blockNumSync uint64) error {
	e.m.Lock()
	if e.cache == nil {
		e.cache = copyRes(res)
	}
	e.m.Unlock()
	return nil
}

type EthGetBlockByNumberMethodHandler struct {
	cache               Cache
	getLatestBlockNumFn GetLatestBlockNumFn
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
	return getBlockDependentCachedRPCResponse(ctx, e.cache, e.getLatestBlockNumFn, key, req)
}

func (e *EthGetBlockByNumberMethodHandler) PutRPCMethod(ctx context.Context, req *RPCReq, res *RPCRes, blockNumberSync uint64) error {
	if ok, err := e.cacheable(req); !ok || err != nil {
		return err
	}
	key := e.cacheKey(req)
	return putBlockDependentCachedRPCResponse(ctx, e.cache, key, res, blockNumberSync)
}

type EthGetBlockRangeMethodHandler struct {
	cache               Cache
	getLatestBlockNumFn GetLatestBlockNumFn
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
	return getBlockDependentCachedRPCResponse(ctx, e.cache, e.getLatestBlockNumFn, key, req)
}

func (e *EthGetBlockRangeMethodHandler) PutRPCMethod(ctx context.Context, req *RPCReq, res *RPCRes, blockNumberSync uint64) error {
	if ok, err := e.cacheable(req); !ok || err != nil {
		return err
	}
	key := e.cacheKey(req)
	return putBlockDependentCachedRPCResponse(ctx, e.cache, key, res, blockNumberSync)
}

type EthCallMethodHandler struct {
	cache               Cache
	getLatestBlockNumFn GetLatestBlockNumFn
}

func (e *EthCallMethodHandler) cacheKey(req *RPCReq) string {
	type ethCallParams struct {
		From     string `json:"from"`
		To       string `json:"to"`
		Gas      string `json:"gas"`
		GasPrice string `json:"gasPrice"`
		Value    string `json:"value"`
		Data     string `json:"data"`
	}
	var params ethCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ""
	}
	// ensure the order is consistent
	keyParams := fmt.Sprintf("%s:%s:%s:%s:%s:%s", params.From, params.To, params.Gas, params.GasPrice, params.Value, params.Data)
	return fmt.Sprintf("method:eth_call:%s", keyParams)
}

func (e *EthCallMethodHandler) GetRPCMethod(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	key := e.cacheKey(req)
	return getBlockDependentCachedRPCResponse(ctx, e.cache, e.getLatestBlockNumFn, key, req)
}

func (e *EthCallMethodHandler) PutRPCMethod(ctx context.Context, req *RPCReq, res *RPCRes, blockNumberSync uint64) error {
	key := e.cacheKey(req)
	return putBlockDependentCachedRPCResponse(ctx, e.cache, key, res, blockNumberSync)
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

func (e *EthBlockNumberMethodHandler) PutRPCMethod(context.Context, *RPCReq, *RPCRes, uint64) error {
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

func (e *EthGasPriceMethodHandler) PutRPCMethod(context.Context, *RPCReq, *RPCRes, uint64) error {
	return nil
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

func makeRPCRes(req *RPCReq, result interface{}) *RPCRes {
	return &RPCRes{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

func copyResError(err *RPCErr) *RPCErr {
	if err == nil {
		return nil
	}
	return &RPCErr{
		Code:          err.Code,
		Message:       err.Message,
		HTTPErrorCode: err.HTTPErrorCode,
	}
}

func copyRes(res *RPCRes) *RPCRes {
	return &RPCRes{
		JSONRPC: res.JSONRPC,
		Result:  res.Result,
		Error:   copyResError(res.Error),
		ID:      res.ID,
	}
}

type CachedRPC struct {
	BlockNum   uint64  `json:"blockNum"`
	Res        *RPCRes `json:"res"`
	Expiration int64   `json:"expiration"` // in millis since epoch
}

func (c *CachedRPC) Encode() []byte {
	return mustMarshalJSON(c)
}

func (c *CachedRPC) Decode(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CachedRPC) ExpirationTime() time.Time {
	return time.Unix(0, c.Expiration*int64(time.Millisecond))
}

func getBlockDependentCachedRPCResponse(ctx context.Context, cache Cache, getLatestBlockNumFn GetLatestBlockNumFn, key string, req *RPCReq) (*RPCRes, error) {
	encodedVal, err := cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if encodedVal == "" {
		return nil, nil
	}
	val, err := snappy.Decode(nil, []byte(encodedVal))
	if err != nil {
		return nil, err
	}

	item := new(CachedRPC)
	if err := json.Unmarshal(val, item); err != nil {
		return nil, err
	}
	curBlockNum, err := getLatestBlockNumFn(ctx)
	if err != nil {
		return nil, err
	}
	expired := time.Now().After(item.ExpirationTime())
	if curBlockNum > item.BlockNum && expired {
		// Remove the key now to avoid biasing LRU list
		// TODO: be careful removing keys once there are multiple proxyd instances
		return nil, cache.Remove(ctx, key)
	} else if curBlockNum < item.BlockNum { /* desync: reorgs, backend failover, slow backend, etc */
		return nil, nil
	}

	res := item.Res
	res.ID = req.ID
	return res, nil
}

func putBlockDependentCachedRPCResponse(ctx context.Context, cache Cache, key string, res *RPCRes, blockNumberSync uint64) error {
	if key == "" {
		return nil
	}
	item := CachedRPC{
		BlockNum:   blockNumberSync,
		Res:        res,
		Expiration: time.Now().Add(cacheTTL).UnixNano() / int64(time.Millisecond),
	}
	val := item.Encode()

	encodedVal := snappy.Encode(nil, val)
	return cache.Put(ctx, key, string(encodedVal), cacheTTL)
}
