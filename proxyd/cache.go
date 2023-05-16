package proxyd

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang/snappy"
	lru "github.com/hashicorp/golang-lru"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string) error
}

const (
	// assuming an average RPCRes size of 3 KB
	memoryCacheLimit = 4096
	// Set a large ttl to avoid expirations. However, a ttl must be set for volatile-lru to take effect.
	redisTTL = 30 * 7 * 24 * time.Hour
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

func newRedisCache(rdb *redis.Client) *redisCache {
	return &redisCache{rdb}
}

func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	val, err := c.rdb.Get(ctx, key).Result()
	redisCacheDurationSumm.WithLabelValues("GET").Observe(float64(time.Since(start).Milliseconds()))

	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		RecordRedisError("CacheGet")
		return "", err
	}
	return val, nil
}

func (c *redisCache) Put(ctx context.Context, key string, value string) error {
	start := time.Now()
	err := c.rdb.SetEX(ctx, key, value, redisTTL).Err()
	redisCacheDurationSumm.WithLabelValues("SETEX").Observe(float64(time.Since(start).Milliseconds()))

	if err != nil {
		RecordRedisError("CacheSet")
	}
	return err
}

type cacheWithCompression struct {
	cache Cache
}

func newCacheWithCompression(cache Cache) *cacheWithCompression {
	return &cacheWithCompression{cache}
}

func (c *cacheWithCompression) Get(ctx context.Context, key string) (string, error) {
	encodedVal, err := c.cache.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if encodedVal == "" {
		return "", nil
	}
	val, err := snappy.Decode(nil, []byte(encodedVal))
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func (c *cacheWithCompression) Put(ctx context.Context, key string, value string) error {
	encodedVal := snappy.Encode(nil, []byte(value))
	return c.cache.Put(ctx, key, string(encodedVal))
}

type GetLatestBlockNumFn func(ctx context.Context) (uint64, error)
type GetLatestGasPriceFn func(ctx context.Context) (uint64, error)

type RPCCache interface {
	GetRPC(ctx context.Context, req *RPCReq) (*RPCRes, error)
	PutRPC(ctx context.Context, req *RPCReq, res *RPCRes) error
}

type rpcCache struct {
	cache    Cache
	handlers map[string]RPCMethodHandler
}

func newRPCCache(cache Cache) RPCCache {
	handlers := map[string]RPCMethodHandler{
		"eth_chainId":                           &StaticMethodHandler{cache: cache},
		"net_version":                           &StaticMethodHandler{cache: cache},
		"eth_getBlockTransactionCountByHash":    &StaticMethodHandler{cache: cache},
		"eth_getUncleCountByBlockHash":          &StaticMethodHandler{cache: cache},
		"eth_getBlockByHash":                    &StaticMethodHandler{cache: cache},
		"eth_getTransactionByHash":              &StaticMethodHandler{cache: cache},
		"eth_getTransactionByBlockHashAndIndex": &StaticMethodHandler{cache: cache},
		"eth_getUncleByBlockHashAndIndex":       &StaticMethodHandler{cache: cache},
		"eth_getTransactionReceipt":             &StaticMethodHandler{cache: cache},
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
	res, err := handler.GetRPCMethod(ctx, req)
	if err != nil {
		RecordCacheError(req.Method)
		return nil, err
	}
	if res == nil {
		RecordCacheMiss(req.Method)
	} else {
		RecordCacheHit(req.Method)
	}
	return res, err
}

func (c *rpcCache) PutRPC(ctx context.Context, req *RPCReq, res *RPCRes) error {
	handler := c.handlers[req.Method]
	if handler == nil {
		return nil
	}
	return handler.PutRPCMethod(ctx, req, res)
}
