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

	cache := newRPCCache(newMemoryCache())
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
			res:  nil,
			name: "eth_getBlockByNumber",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockRange",
				Params:  []byte(`["0x1", "0x2", false]`),
				ID:      ID,
			},
			res:  nil,
			name: "eth_getBlockRange",
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
	ctx := context.Background()

	cache := newRPCCache(newMemoryCache())
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
