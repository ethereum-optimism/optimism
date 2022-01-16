package proxyd

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	lru "github.com/hashicorp/golang-lru"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string, ttl time.Duration) error
	Remove(ctx context.Context, key string) error
}

const (
	// assuming an average RPCRes size of 3 KB
	memoryCacheLimit = 4096
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

func (c *cache) Put(ctx context.Context, key string, value string, ttl time.Duration) error {
	c.lru.Add(key, value)
	return nil
}

func (c *cache) Remove(ctx context.Context, key string) error {
	c.lru.Remove(key)
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
		RecordRedisError("CacheGet")
		return "", err
	}
	return val, nil
}

func (c *redisCache) Put(ctx context.Context, key string, value string, ttl time.Duration) error {
	err := c.rdb.Set(ctx, key, value, 0).Err()
	if err != nil {
		RecordRedisError("CacheSet")
	}
	return err
}

func (c *redisCache) Remove(ctx context.Context, key string) error {
	err := c.rdb.Del(ctx, key).Err()
	return err
}

type GetLatestBlockNumFn func(ctx context.Context) (uint64, error)
type GetLatestGasPriceFn func(ctx context.Context) (uint64, error)

type RPCCache interface {
	GetRPC(ctx context.Context, req *RPCReq) (*RPCRes, error)

	// The blockNumberSync is used to enforce Sequential Consistency. We make the following assumptions to do this:
	// 1. No Reorgs. Reoorgs are handled by the Cache during retrieval
	// 2. The backend yields synchronized block numbers and RPC Responses.
	// 2. No backend failover. If there's a failover then we may desync as we use a different backend
	// that doesn't have our block.
	PutRPC(ctx context.Context, req *RPCReq, res *RPCRes, blockNumberSync uint64) error
}

type rpcCache struct {
	cache    Cache
	handlers map[string]RPCMethodHandler
}

func newRPCCache(cache Cache, getLatestBlockNumFn GetLatestBlockNumFn, getLatestGasPriceFn GetLatestGasPriceFn) RPCCache {
	handlers := map[string]RPCMethodHandler{
		"eth_chainId":          &StaticMethodHandler{},
		"net_version":          &StaticMethodHandler{},
		"eth_getBlockByNumber": &EthGetBlockByNumberMethodHandler{cache, getLatestBlockNumFn},
		"eth_getBlockRange":    &EthGetBlockRangeMethodHandler{cache, getLatestBlockNumFn},
		"eth_blockNumber":      &EthBlockNumberMethodHandler{getLatestBlockNumFn},
		"eth_gasPrice":         &EthGasPriceMethodHandler{getLatestGasPriceFn},
		"eth_call":             &EthCallMethodHandler{cache, getLatestBlockNumFn},
	}
	return &rpcCache{
		cache:    cache,
		handlers: handlers,
	}
}

func (c *rpcCache) GetRPC(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	handler := c.handlers[req.Method]
	if handler == nil {
		return nil, nil
	}
	return handler.GetRPCMethod(ctx, req)
}

func (c *rpcCache) PutRPC(ctx context.Context, req *RPCReq, res *RPCRes, blockNumberSync uint64) error {
	handler := c.handlers[req.Method]
	if handler == nil {
		return nil
	}
	return handler.PutRPCMethod(ctx, req, res, blockNumberSync)
}
