package proxyd

import (
	"context"
	"encoding/json"
	"time"

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
		RecordRedisError("CacheGet")
		return "", err
	}
	return val, nil
}

func (c *redisCache) Put(ctx context.Context, key string, value string) error {
	err := c.rdb.Set(ctx, key, value, 0).Err()
	if err != nil {
		RecordRedisError("CacheSet")
	}
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

type CachedRPC struct {
	BlockNum uint64  `json:"blockNum"`
	Res      *RPCRes `json:"res"`
	TTL      int64   `json:"ttl"`
}

func (c *CachedRPC) Encode() []byte {
	return mustMarshalJSON(c)
}

func (c *CachedRPC) Decode(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *CachedRPC) Expiration() time.Time {
	return time.Unix(0, c.TTL*int64(time.Millisecond))
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
		RecordCacheMiss(req.Method)
		return nil, nil
	}

	key := handler.CacheKey(req)
	encodedVal, err := c.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if encodedVal == "" {
		RecordCacheMiss(req.Method)
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
	expired := item.Expiration().After(time.Now())
	curBlockNum, err := c.getLatestBlockNumFn(ctx)
	if err != nil {
		return nil, err
	}
	if curBlockNum > item.BlockNum && expired {
		// TODO: what to do with expired items? Ideally they shouldn't count towards recency
		return nil, nil
	} else if curBlockNum < item.BlockNum { // reorg?
		return nil, nil
	}

	RecordCacheHit(req.Method)
	res := item.Res
	res.ID = req.ID
	return res, nil

	/*
		res := new(RPCRes)
		err = json.Unmarshal(val, res)
		if err != nil {
			return nil, err
		}
		res.ID = req.ID
		return res, nil
	*/
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

	blockNum, err := c.getLatestBlockNumFn(ctx)
	if err != nil {
		return err
	}
	key := handler.CacheKey(req)
	item := CachedRPC{BlockNum: blockNum, Res: res, TTL: time.Now().UnixNano() / int64(time.Millisecond)}
	val := item.Encode()

	//val := mustMarshalJSON(res)
	encodedVal := snappy.Encode(nil, val)
	return c.cache.Put(ctx, key, string(encodedVal))
}
