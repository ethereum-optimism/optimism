package proxyd

import (
	"context"
	"encoding/json"

	"github.com/golang/snappy"
	lru "github.com/hashicorp/golang-lru"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string) error
}

const memoryCacheLimit = 1024 * 1024

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

	var params []interface{}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return false
	}

	switch req.Method {
	case "eth_getBlockByNumber":
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
