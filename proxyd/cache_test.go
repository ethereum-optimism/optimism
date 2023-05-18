package proxyd

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRPCCacheImmutableRPCs(t *testing.T) {
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
				Method:  "eth_getBlockTransactionCountByHash",
				Params:  mustMarshalJSON([]string{"0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238"}),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"eth_getBlockTransactionCountByHash":"!"}`,
				ID:      ID,
			},
			name: "eth_getBlockTransactionCountByHash",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getUncleCountByBlockHash",
				Params:  mustMarshalJSON([]string{"0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238"}),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"eth_getUncleCountByBlockHash":"!"}`,
				ID:      ID,
			},
			name: "eth_getUncleCountByBlockHash",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByHash",
				Params:  mustMarshalJSON([]string{"0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b", "false"}),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"eth_getBlockByHash":"!"}`,
				ID:      ID,
			},
			name: "eth_getBlockByHash",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getTransactionByHash",
				Params:  mustMarshalJSON([]string{"0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b"}),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"eth_getTransactionByHash":"!"}`,
				ID:      ID,
			},
			name: "eth_getTransactionByHash",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getTransactionByBlockHashAndIndex",
				Params:  mustMarshalJSON([]string{"0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331", "0x55"}),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"eth_getTransactionByBlockHashAndIndex":"!"}`,
				ID:      ID,
			},
			name: "eth_getTransactionByBlockHashAndIndex",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getUncleByBlockHashAndIndex",
				Params:  mustMarshalJSON([]string{"0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238", "0x90"}),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"eth_getUncleByBlockHashAndIndex":"!"}`,
				ID:      ID,
			},
			name: "eth_getUncleByBlockHashAndIndex",
		},
		{
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getTransactionReceipt",
				Params:  mustMarshalJSON([]string{"0x85d995eba9763907fdf35cd2034144dd9d53ce32cbec21349d4b12823c6860c5"}),
				ID:      ID,
			},
			res: &RPCRes{
				JSONRPC: "2.0",
				Result:  `{"eth_getTransactionReceipt":"!"}`,
				ID:      ID,
			},
			name: "eth_getTransactionReceipt",
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

	rpcs := []struct {
		req  *RPCReq
		name string
	}{
		{
			name: "eth_syncing",
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_syncing",
				ID:      ID,
			},
		},
		{
			name: "eth_blockNumber",
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_blockNumber",
				ID:      ID,
			},
		},
		{
			name: "eth_getBlockByNumber",
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				ID:      ID,
			},
		},
		{
			name: "eth_getBlockRange",
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_getBlockRange",
				ID:      ID,
			},
		},
		{
			name: "eth_gasPrice",
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_gasPrice",
				ID:      ID,
			},
		},
		{
			name: "eth_call",
			req: &RPCReq{
				JSONRPC: "2.0",
				Method:  "eth_gasPrice",
				ID:      ID,
			},
		},
	}

	for _, rpc := range rpcs {
		t.Run(rpc.name, func(t *testing.T) {
			fakeval := mustMarshalJSON([]string{rpc.name})
			err := cache.PutRPC(ctx, rpc.req, &RPCRes{Result: fakeval})
			require.NoError(t, err)

			cachedRes, err := cache.GetRPC(ctx, rpc.req)
			require.NoError(t, err)
			require.Nil(t, cachedRes)
		})
	}

}
