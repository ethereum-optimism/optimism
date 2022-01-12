package proxyd

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"
	"github.com/golang/snappy"
	lru "github.com/hashicorp/golang-lru"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string) error
}

// assuming an average RPCRes size of 3 KB
const memoryCacheLimit = 4096

var supportedRPCMethods = map[string]bool{
	"eth_chainId":          true,
	"net_version":          true,
	"eth_getBlockByNumber": true,
	"eth_getBlockRange":    true,
}

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
	if err != nil {
		return "", err
	}
	return val, nil
}

func (c *redisCache) Put(ctx context.Context, key string, value string) error {
	err := c.rdb.Set(ctx, key, value, 0).Err()
	return err
}

type RPCCache struct {
	cache Cache
}

func newRPCCache(cache Cache) *RPCCache {
	return &RPCCache{cache: cache}
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

	key := mustMarshalJSON(req)
	val := mustMarshalJSON(res)
	encodedVal := snappy.Encode(nil, val)
	return c.cache.Put(ctx, string(key), string(encodedVal))
}

func (c *RPCCache) isCacheable(req *RPCReq) bool {
	if !supportedRPCMethods[req.Method] {
		return false
	}

	switch req.Method {
	case "eth_getBlockByNumber":
		var params []interface{}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return false
		}
		if len(params) != 2 {
			return false
		}
		blockNum, ok := params[0].(string)
		if !ok {
			return false
		}
		if isBlockDependentParam(blockNum) {
			return false
		}

	case "eth_getBlockRange":
		var params []interface{}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return false
		}
		if len(params) != 3 {
			return false
		}
		startBlockNum, ok := params[0].(string)
		if !ok {
			return false
		}
		endBlockNum, ok := params[1].(string)
		if !ok {
			return false
		}
		if isBlockDependentParam(startBlockNum) || isBlockDependentParam(endBlockNum) {
			return false
		}
	}

	return true
}

func isBlockDependentParam(s string) bool {
	return s == "latest" || s == "pending"
}
