package proxyd

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-redis/redis/v8"
	"github.com/golang/snappy"
	lru "github.com/hashicorp/golang-lru"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string) error
}

// assuming an average RPCRes size of 3 KB
const (
	memoryCacheLimit      = 4096
	numBlockConfirmations = 50
)

var (
	supportedRPCMethods = map[string]bool{
		"eth_chainId":          true,
		"net_version":          true,
		"eth_getBlockByNumber": true,
		"eth_getBlockRange":    true,
	}
	supportedBlockRPCMethods = map[string]bool{
		"eth_getBlockByNumber": true,
		"eth_getBlockRange":    true,
	}
)

var (
	errInvalidBlockByNumberParams = errors.New("invalid eth_getBlockByNumber params")
	errUnavailableBlockNumSyncer  = errors.New("getLatestBlockFn not set for required RPC")
)

type cache struct {
	lru *lru.Cache
}

func newMemoryCache() *cache {
	rep, _ := lru.New(memoryCacheLimit)
	return &cache{rep}
}

func (c *cache) Get(ctx context.Context, key string) (string, error) {
	if val, ok := c.lru.Get(key); ok {
		return val.(string), nil
	}
	return "", nil
}

func (c *cache) Put(ctx context.Context, key string, value string) error {
	c.lru.Add(key, value)
	return nil
}

type redisCache struct {
	rdb *redis.Client
}

func newRedisCache(url string) (*redisCache, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, wrapErr(err, "error connecting to redis")
	}
	return &redisCache{rdb}, nil
}

func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return val, nil
}

func (c *redisCache) Put(ctx context.Context, key string, value string) error {
	err := c.rdb.Set(ctx, key, value, 0).Err()
	return err
}

type GetLatestBlockNumFn func(ctx context.Context) (uint64, error)

type RPCCache struct {
	cache               Cache
	getLatestBlockNumFn GetLatestBlockNumFn
}

func newRPCCache(cache Cache, getLatestBlockNumFn GetLatestBlockNumFn) *RPCCache {
	return &RPCCache{cache: cache, getLatestBlockNumFn: getLatestBlockNumFn}
}

func (c *RPCCache) GetRPC(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	if !c.isCacheable(req) {
		return nil, nil
	}

	key := mustMarshalJSON(req)
	encodedVal, err := c.cache.Get(ctx, string(key))
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

	res := new(RPCRes)
	err = json.Unmarshal(val, res)
	if err != nil {
		panic(err)
	}

	return res, nil
}

func (c *RPCCache) PutRPC(ctx context.Context, req *RPCReq, res *RPCRes) error {
	if !c.isCacheable(req) {
		return nil
	}
	if supportedBlockRPCMethods[req.Method] {
		if ok, err := c.isConfirmed(ctx, req); err != nil {
			return err
		} else if !ok {
			return nil
		}
	}

	key := mustMarshalJSON(req)
	val := mustMarshalJSON(res)
	encodedVal := snappy.Encode(nil, val)
	return c.cache.Put(ctx, string(key), string(encodedVal))
}

func (c *RPCCache) isConfirmed(ctx context.Context, req *RPCReq) (bool, error) {
	if c.getLatestBlockNumFn == nil {
		return false, errUnavailableBlockNumSyncer
	}
	curBlock, err := c.getLatestBlockNumFn(ctx)
	if err != nil {
		return false, err
	}

	switch req.Method {
	case "eth_getBlockByNumber":
		blockInput, _, err := decodeGetBlockByNumberParams(req.Params)
		if err != nil {
			return false, err
		}
		if isBlockDependentParam(blockInput) {
			return false, nil
		}
		if blockInput == "earliest" {
			return true, nil
		}
		blockNum, err := decodeBlockInput(blockInput)
		if err != nil {
			return false, err
		}
		return blockNum+numBlockConfirmations <= curBlock, nil

	case "eth_getBlockRange":
		start, end, _, err := decodeGetBlockRangeParams(req.Params)
		if err != nil {
			return false, err
		}
		if isBlockDependentParam(start) || isBlockDependentParam(end) {
			return false, nil
		}
		if start == "earliest" || end == "earliest" {
			return true, nil
		}
		startNum, err := decodeBlockInput(start)
		if err != nil {
			return false, err
		}
		endNum, err := decodeBlockInput(end)
		if err != nil {
			return false, err
		}
		return startNum+numBlockConfirmations <= curBlock && endNum+numBlockConfirmations <= curBlock, nil
	}

	return true, nil
}

func (c *RPCCache) isCacheable(req *RPCReq) bool {
	if !supportedRPCMethods[req.Method] {
		return false
	}

	switch req.Method {
	case "eth_getBlockByNumber":
		blockNum, _, err := decodeGetBlockByNumberParams(req.Params)
		if err != nil {
			return false
		}
		return !isBlockDependentParam(blockNum)
	case "eth_getBlockRange":
		start, end, _, err := decodeGetBlockRangeParams(req.Params)
		if err != nil {
			return false
		}
		return !isBlockDependentParam(start) && !isBlockDependentParam(end)
	}

	return true
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
