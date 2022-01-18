package proxyd

import (
	"context"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRPCCacheWhitelist(t *testing.T) {
	const blockHead = math.MaxUint64
	ctx := context.Background()

	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), fn)
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
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"difficulty": "0x1", "number": "0x1"}`,
				ID:      ID,
			},
			name: "eth_getBlockByNumber earliest",
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
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `[{"number": "0x1"}, {"number": "0x2"}]`,
				ID:      ID,
			},
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

func TestRPCCacheUnsupportedMethod(t *testing.T) {
	const blockHead = math.MaxUint64
	ctx := context.Background()

	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), fn)
	ID := []byte(strconv.Itoa(1))

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
	require.Nil(t, cachedRes)
}

func TestRPCCacheEthGetBlockByNumberForRecentBlocks(t *testing.T) {
	ctx := context.Background()

	var blockHead uint64 = 2
	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), fn)
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
				Params:  []byte(`["0x1", false]`),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"difficulty": "0x1", "number": "0x1"}`,
				ID:      ID,
			},
			name: "recent block num",
		},
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
	cache := newRPCCache(newMemoryCache(), fn)
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

func TestRPCCacheEthGetBlockRangeForRecentBlocks(t *testing.T) {
	ctx := context.Background()

	var blockHead uint64 = 0x1000
	fn := func(ctx context.Context) (uint64, error) {
		return blockHead, nil
	}
	cache := newRPCCache(newMemoryCache(), fn)
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
				Params:  []byte(`["0x1", "0x1000", false]`),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `[{"number": "0x1"}, {"number": "0x2"}]`,
				ID:      ID,
			},
			name: "recent block num",
		},
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
	cache := newRPCCache(newMemoryCache(), fn)
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
