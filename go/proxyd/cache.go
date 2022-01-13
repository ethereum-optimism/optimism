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
const (
	memoryCacheLimit      = 4096
	numBlockConfirmations = 50
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

type RPCCache interface {
	GetRPC(ctx context.Context, req *RPCReq) (*RPCRes, error)
	PutRPC(ctx context.Context, req *RPCReq, res *RPCRes) error
}

type rpcCache struct {
	cache               Cache
	getLatestBlockNumFn GetLatestBlockNumFn
	handlers            map[string]RPCMethodHandler
}

func newRPCCache(cache Cache, getLatestBlockNumFn GetLatestBlockNumFn) RPCCache {
	handlers := map[string]RPCMethodHandler{
		"eth_chainId":          &StaticRPCMethodHandler{"eth_chainId"},
		"net_version":          &StaticRPCMethodHandler{"net_version"},
		"eth_getBlockByNumber": &EthGetBlockByNumberMethod{getLatestBlockNumFn},
		"eth_getBlockRange":    &EthGetBlockRangeMethod{getLatestBlockNumFn},
	}
	return &rpcCache{cache: cache, getLatestBlockNumFn: getLatestBlockNumFn, handlers: handlers}
}

func (c *rpcCache) GetRPC(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	handler := c.handlers[req.Method]
	if handler == nil {
		return nil, nil
	}
	cacheable, err := handler.IsCacheable(req)
	if err != nil {
		return nil, err
	}
	if !cacheable {
		return nil, nil
	}

	key := handler.CacheKey(req)
	encodedVal, err := c.cache.Get(ctx, key)
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
		return nil, err
	}
	res.ID = req.ID
	return res, nil
}

func (c *rpcCache) PutRPC(ctx context.Context, req *RPCReq, res *RPCRes) error {
	handler := c.handlers[req.Method]
	if handler == nil {
		return nil
	}
	cacheable, err := handler.IsCacheable(req)
	if err != nil {
		return err
	}
	if !cacheable {
		return nil
	}
	requiresConfirmations, err := handler.RequiresUnconfirmedBlocks(ctx, req)
	if err != nil {
		return err
	}
	if requiresConfirmations {
		return nil
	}

	key := handler.CacheKey(req)
	val := mustMarshalJSON(res)
	encodedVal := snappy.Encode(nil, val)
	return c.cache.Put(ctx, key, string(encodedVal))
}
