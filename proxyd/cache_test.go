package proxyd

import (
	"context"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

const numBlockConfirmations = 10

func TestRPCCacheImmutableRPCs(t *testing.T) {
	const blockHead = math.MaxUint64
	ctx := context.Background()

	getBlockNum := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), getBlockNum, nil, numBlockConfirmations)
	ID := []byte(strconv.Itoa(1))

	rpcs := []struct {
		req  *RPCReq
		res  *RPCRes
		name string
	}{
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_chainId",
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  "0xff",
				ID:      ID,
			},
			name: "eth_chainId",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "net_version",
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  "9999",
				ID:      ID,
			},
			name: "net_version",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				Params:  []byte(`["0x1", false]`),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"difficulty": "0x1", "number": "0x1"}`,
				ID:      ID,
			},
			name: "eth_getBlockByNumber",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				Params:  []byte(`["earliest", false]`),
				ID:      ID,
			},
			res:  nil,
			name: "eth_getBlockByNumber earliest",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				Params:  []byte(`["safe", false]`),
				ID:      ID,
			},
			res:  nil,
			name: "eth_getBlockByNumber safe",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				Params:  []byte(`["finalized", false]`),
				ID:      ID,
			},
			res:  nil,
			name: "eth_getBlockByNumber finalized",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				Params:  []byte(`["pending", false]`),
				ID:      ID,
			},
			res:  nil,
			name: "eth_getBlockByNumber pending",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				Params:  []byte(`["latest", false]`),
				ID:      ID,
			},
			res:  nil,
			name: "eth_getBlockByNumber latest",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockRange",
				Params:  []byte(`["0x1", "0x2", false]`),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `[{"number": "0x1"}, {"number": "0x2"}]`,
				ID:      ID,
			},
			name: "eth_getBlockRange",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockRange",
				Params:  []byte(`["earliest", "0x2", false]`),
				ID:      ID,
			},
			res:  nil,
			name: "eth_getBlockRange earliest",
		},
	}

	for _, rpc := range rpcs {
		t.Run(rpc.name, func(t *testing.T) {
			err := cache.PutRPC(ctx, rpc.req, rpc.res)
			require.NoError(t, err)

			cachedRes, err := cache.GetRPC(ctx, rpc.req)
			require.NoError(t, err)
			require.Equal(t, rpc.res, cachedRes)
		})
	}
}

func TestRPCCacheBlockNumber(t *testing.T) {
	var blockHead uint64 = 0x1000
	var gasPrice uint64 = 0x100
	ctx := context.Background()
	ID := []byte(strconv.Itoa(1))

	getGasPrice := func(ctx context.Context) (uint64, error) {
		return gasPrice, nil
	}
	getBlockNum := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), getBlockNum, getGasPrice, numBlockConfirmations)

	req := &RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		ID:      ID,
	}
	res := &RPCRes{
		JSONRPC: "2.0",
		Result:  `0x1000`,
		ID:      ID,
	}

	err := cache.PutRPC(ctx, req, res)
	require.NoError(t, err)

	cachedRes, err := cache.GetRPC(ctx, req)
	require.NoError(t, err)
	require.Equal(t, res, cachedRes)

	blockHead = 0x1001
	cachedRes, err = cache.GetRPC(ctx, req)
	require.NoError(t, err)
	require.Equal(t, &RPCRes{JSONRPC: "2.0", Result: `0x1001`, ID: ID}, cachedRes)
}

func TestRPCCacheGasPrice(t *testing.T) {
	var blockHead uint64 = 0x1000
	var gasPrice uint64 = 0x100
	ctx := context.Background()
	ID := []byte(strconv.Itoa(1))

	getGasPrice := func(ctx context.Context) (uint64, error) {
		return gasPrice, nil
	}
	getBlockNum := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), getBlockNum, getGasPrice, numBlockConfirmations)

	req := &RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_gasPrice",
		ID:      ID,
	}
	res := &RPCRes{
		JSONRPC: "2.0",
		Result:  `0x100`,
		ID:      ID,
	}

	err := cache.PutRPC(ctx, req, res)
	require.NoError(t, err)

	cachedRes, err := cache.GetRPC(ctx, req)
	require.NoError(t, err)
	require.Equal(t, res, cachedRes)

	gasPrice = 0x101
	cachedRes, err = cache.GetRPC(ctx, req)
	require.NoError(t, err)
	require.Equal(t, &RPCRes{JSONRPC: "2.0", Result: `0x101`, ID: ID}, cachedRes)
}

func TestRPCCacheUnsupportedMethod(t *testing.T) {
	const blockHead = math.MaxUint64
	ctx := context.Background()

	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), fn, nil, numBlockConfirmations)
	ID := []byte(strconv.Itoa(1))

	req := &RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_syncing",
		ID:      ID,
	}
	res := &RPCRes{
		JSONRPC: "2.0",
		Result:  false,
		ID:      ID,
	}

	err := cache.PutRPC(ctx, req, res)
	require.NoError(t, err)

	cachedRes, err := cache.GetRPC(ctx, req)
	require.NoError(t, err)
	require.Nil(t, cachedRes)
}

func TestRPCCacheEthGetBlockByNumber(t *testing.T) {
	ctx := context.Background()

	var blockHead uint64
	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	makeCache := func() RPCCache { return newRPCCache(newMemoryCache(), fn, nil, numBlockConfirmations) }
	ID := []byte(strconv.Itoa(1))

	req := &RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []byte(`["0xa", false]`),
		ID:      ID,
	}
	res := &RPCRes{
		JSONRPC: "2.0",
		Result:  `{"difficulty": "0x1", "number": "0x1"}`,
		ID:      ID,
	}
	req2 := &RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []byte(`["0xb", false]`),
		ID:      ID,
	}
	res2 := &RPCRes{
		JSONRPC: "2.0",
		Result:  `{"difficulty": "0x2", "number": "0x2"}`,
		ID:      ID,
	}

	t.Run("set multiple finalized blocks", func(t *testing.T) {
		blockHead = 100
		cache := makeCache()
		require.NoError(t, cache.PutRPC(ctx, req, res))
		require.NoError(t, cache.PutRPC(ctx, req2, res2))
		cachedRes, err := cache.GetRPC(ctx, req)
		require.NoError(t, err)
		require.Equal(t, res, cachedRes)
		cachedRes, err = cache.GetRPC(ctx, req2)
		require.NoError(t, err)
		require.Equal(t, res2, cachedRes)
	})

	t.Run("unconfirmed block", func(t *testing.T) {
		blockHead = 0xc
		cache := makeCache()
		require.NoError(t, cache.PutRPC(ctx, req, res))
		cachedRes, err := cache.GetRPC(ctx, req)
		require.NoError(t, err)
		require.Nil(t, cachedRes)
	})
}

func TestRPCCacheEthGetBlockByNumberForRecentBlocks(t *testing.T) {
	ctx := context.Background()

	var blockHead uint64 = 2
	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), fn, nil, numBlockConfirmations)
	ID := []byte(strconv.Itoa(1))

	rpcs := []struct {
		req  *RPCReq
		res  *RPCRes
		name string
	}{
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				Params:  []byte(`["latest", false]`),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"difficulty": "0x1", "number": "0x1"}`,
				ID:      ID,
			},
			name: "latest block",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				Params:  []byte(`["pending", false]`),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"difficulty": "0x1", "number": "0x1"}`,
				ID:      ID,
			},
			name: "pending block",
		},
	}

	for _, rpc := range rpcs {
		t.Run(rpc.name, func(t *testing.T) {
			err := cache.PutRPC(ctx, rpc.req, rpc.res)
			require.NoError(t, err)

			cachedRes, err := cache.GetRPC(ctx, rpc.req)
			require.NoError(t, err)
			require.Nil(t, cachedRes)
		})
	}
}

func TestRPCCacheEthGetBlockByNumberInvalidRequest(t *testing.T) {
	ctx := context.Background()

	const blockHead = math.MaxUint64
	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), fn, nil, numBlockConfirmations)
	ID := []byte(strconv.Itoa(1))

	req := &RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []byte(`["0x1"]`), // missing required boolean param
		ID:      ID,
	}
	res := &RPCRes{
		JSONRPC: "2.0",
		Result:  `{"difficulty": "0x1", "number": "0x1"}`,
		ID:      ID,
	}

	err := cache.PutRPC(ctx, req, res)
	require.Error(t, err)

	cachedRes, err := cache.GetRPC(ctx, req)
	require.Error(t, err)
	require.Nil(t, cachedRes)
}

func TestRPCCacheEthGetBlockRange(t *testing.T) {
	ctx := context.Background()

	var blockHead uint64
	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	makeCache := func() RPCCache { return newRPCCache(newMemoryCache(), fn, nil, numBlockConfirmations) }
	ID := []byte(strconv.Itoa(1))

	t.Run("finalized block", func(t *testing.T) {
		req := &RPCReq{
			JSONRPC: "2.0",
			Method:  "eth_getBlockRange",
			Params:  []byte(`["0x1", "0x10", false]`),
			ID:      ID,
		}
		res := &RPCRes{
			JSONRPC: "2.0",
			Result:  `[{"number": "0x1"}, {"number": "0x10"}]`,
			ID:      ID,
		}
		blockHead = 0x1000
		cache := makeCache()
		require.NoError(t, cache.PutRPC(ctx, req, res))
		cachedRes, err := cache.GetRPC(ctx, req)
		require.NoError(t, err)
		require.Equal(t, res, cachedRes)
	})

	t.Run("unconfirmed block", func(t *testing.T) {
		cache := makeCache()
		req := &RPCReq{
			JSONRPC: "2.0",
			Method:  "eth_getBlockRange",
			Params:  []byte(`["0x1", "0x1000", false]`),
			ID:      ID,
		}
		res := &RPCRes{
			JSONRPC: "2.0",
			Result:  `[{"number": "0x1"}, {"number": "0x2"}]`,
			ID:      ID,
		}
		require.NoError(t, cache.PutRPC(ctx, req, res))
		cachedRes, err := cache.GetRPC(ctx, req)
		require.NoError(t, err)
		require.Nil(t, cachedRes)
	})
}

func TestRPCCacheEthGetBlockRangeForRecentBlocks(t *testing.T) {
	ctx := context.Background()

	var blockHead uint64 = 0x1000
	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), fn, nil, numBlockConfirmations)
	ID := []byte(strconv.Itoa(1))

	rpcs := []struct {
		req  *RPCReq
		res  *RPCRes
		name string
	}{
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockRange",
				Params:  []byte(`["0x1", "latest", false]`),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `[{"number": "0x1"}, {"number": "0x2"}]`,
				ID:      ID,
			},
			name: "latest block",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockRange",
				Params:  []byte(`["0x1", "pending", false]`),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `[{"number": "0x1"}, {"number": "0x2"}]`,
				ID:      ID,
			},
			name: "pending block",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockRange",
				Params:  []byte(`["latest", "0x1000", false]`),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `[{"number": "0x1"}, {"number": "0x2"}]`,
				ID:      ID,
			},
			name: "latest block 2",
		},
	}

	for _, rpc := range rpcs {
		t.Run(rpc.name, func(t *testing.T) {
			err := cache.PutRPC(ctx, rpc.req, rpc.res)
			require.NoError(t, err)

			cachedRes, err := cache.GetRPC(ctx, rpc.req)
			require.NoError(t, err)
			require.Nil(t, cachedRes)
		})
	}
}

func TestRPCCacheEthGetBlockRangeInvalidRequest(t *testing.T) {
	ctx := context.Background()

	const blockHead = math.MaxUint64
	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), fn, nil, numBlockConfirmations)
	ID := []byte(strconv.Itoa(1))

	rpcs := []struct {
		req  *RPCReq
		res  *RPCRes
		name string
	}{
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockRange",
				Params:  []byte(`["0x1", "0x2"]`), // missing required boolean param
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `[{"number": "0x1"}, {"number": "0x2"}]`,
				ID:      ID,
			},
			name: "missing boolean param",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockRange",
				Params:  []byte(`["abc", "0x2", true]`), // invalid block hex
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `[{"number": "0x1"}, {"number": "0x2"}]`,
				ID:      ID,
			},
			name: "invalid block hex",
		},
	}

	for _, rpc := range rpcs {
		t.Run(rpc.name, func(t *testing.T) {
			err := cache.PutRPC(ctx, rpc.req, rpc.res)
			require.Error(t, err)

			cachedRes, err := cache.GetRPC(ctx, rpc.req)
			require.Error(t, err)
			require.Nil(t, cachedRes)
		})
	}
}

func TestRPCCacheEthCall(t *testing.T) {
	ctx := context.Background()

	var blockHead uint64
	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}

	makeCache := func() RPCCache { return newRPCCache(newMemoryCache(), fn, nil, numBlockConfirmations) }
	ID := []byte(strconv.Itoa(1))

	req := &RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_call",
		Params:  []byte(`[{"to": "0xDEADBEEF", "data": "0x1"}, "0x10"]`),
		ID:      ID,
	}
	res := &RPCRes{
		JSONRPC: "2.0",
		Result:  `0x0`,
		ID:      ID,
	}

	t.Run("finalized block", func(t *testing.T) {
		blockHead = 0x100
		cache := makeCache()
		err := cache.PutRPC(ctx, req, res)
		require.NoError(t, err)
		cachedRes, err := cache.GetRPC(ctx, req)
		require.NoError(t, err)
		require.Equal(t, res, cachedRes)
	})

	t.Run("unconfirmed block", func(t *testing.T) {
		blockHead = 0x10
		cache := makeCache()
		require.NoError(t, cache.PutRPC(ctx, req, res))
		cachedRes, err := cache.GetRPC(ctx, req)
		require.NoError(t, err)
		require.Nil(t, cachedRes)
	})

	t.Run("latest block", func(t *testing.T) {
		blockHead = 0x100
		req := &RPCReq{
			JSONRPC: "2.0",
			Method:  "eth_call",
			Params:  []byte(`[{"to": "0xDEADBEEF", "data": "0x1"}, "latest"]`),
			ID:      ID,
		}
		cache := makeCache()
		require.NoError(t, cache.PutRPC(ctx, req, res))
		cachedRes, err := cache.GetRPC(ctx, req)
		require.NoError(t, err)
		require.Nil(t, cachedRes)
	})

	t.Run("pending block", func(t *testing.T) {
		blockHead = 0x100
		req := &RPCReq{
			JSONRPC: "2.0",
			Method:  "eth_call",
			Params:  []byte(`[{"to": "0xDEADBEEF", "data": "0x1"}, "pending"]`),
			ID:      ID,
		}
		cache := makeCache()
		require.NoError(t, cache.PutRPC(ctx, req, res))
		cachedRes, err := cache.GetRPC(ctx, req)
		require.NoError(t, err)
		require.Nil(t, cachedRes)
	})
}
