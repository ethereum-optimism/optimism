package proxyd

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

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
	rdb    *redis.Client
	prefix string
}

func newRedisCache(rdb *redis.Client, prefix string) *redisCache {
	return &redisCache{rdb, prefix}
}

func (c *redisCache) namespaced(key string) string {
	if c.prefix == "" {
		return key
	}
	return strings.Join([]string{c.prefix, key}, ":")
}

func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	val, err := c.rdb.Get(ctx, c.namespaced(key)).Result()
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
	err := c.rdb.SetEX(ctx, c.namespaced(key), value, redisTTL).Err()
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

type RPCCache interface {
	GetRPC(ctx context.Context, req *RPCReq) (*RPCRes, error)
	PutRPC(ctx context.Context, req *RPCReq, res *RPCRes) error
}

type rpcCache struct {
	cache    Cache
	handlers map[string]RPCMethodHandler
}

func newRPCCache(cache Cache) RPCCache {
	staticHandler := &StaticMethodHandler{cache: cache}
	debugGetRawReceiptsHandler := &StaticMethodHandler{cache: cache,
		filter: func(req *RPCReq) bool {
			// cache only if the request is for a block hash

			var p []rpc.BlockNumberOrHash
			err := json.Unmarshal(req.Params, &p)
			if err != nil {
				return false
			}
			if len(p) != 1 {
				return false
			}
			return p[0].BlockHash != nil
		},
	}
	handlers := map[string]RPCMethodHandler{
		"eth_chainId":                           staticHandler,
		"net_version":                           staticHandler,
		"eth_getBlockTransactionCountByHash":    staticHandler,
		"eth_getUncleCountByBlockHash":          staticHandler,
		"eth_getBlockByHash":                    staticHandler,
		"eth_getTransactionByBlockHashAndIndex": staticHandler,
		"eth_getUncleByBlockHashAndIndex":       staticHandler,
		"debug_getRawReceipts":                  debugGetRawReceiptsHandler,
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
	return res, nil
}

func (c *rpcCache) PutRPC(ctx context.Context, req *RPCReq, res *RPCRes) error {
	handler := c.handlers[req.Method]
	if handler == nil {
		return nil
	}
	return handler.PutRPCMethod(ctx, req, res)
}
