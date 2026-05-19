package filter

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestQueryFrontendGetBlockHashByNumberRPC(t *testing.T) {
	client := newTestQueryRPCClient(t, false)

	t.Run("latest selector", func(t *testing.T) {
		var result common.Hash
		err := client.Call(&result, "interop_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), "latest")
		require.NoError(t, err)
		require.Equal(t, common.HexToHash("0x02"), result)
	})

	t.Run("numeric selector", func(t *testing.T) {
		var result common.Hash
		err := client.Call(&result, "interop_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), rpc.BlockNumber(100))
		require.NoError(t, err)
		require.Equal(t, common.HexToHash("0x01"), result)
	})

	t.Run("missing block", func(t *testing.T) {
		var result common.Hash
		err := client.Call(&result, "interop_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), rpc.BlockNumber(999))
		require.ErrorContains(t, err, "not found")
	})

	t.Run("unknown chain", func(t *testing.T) {
		var result common.Hash
		err := client.Call(&result, "interop_getBlockHashByNumber", eth.ChainIDFromUInt64(999), rpc.BlockNumber(100))
		require.ErrorContains(t, err, "unknown chain")
	})

	t.Run("unsupported tag", func(t *testing.T) {
		var result common.Hash
		err := client.Call(&result, "interop_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), "safe")
		require.ErrorContains(t, err, "unsupported block tag")
	})

	t.Run("legacy supervisor namespace unavailable by default", func(t *testing.T) {
		var result common.Hash
		err := client.Call(&result, "supervisor_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), "latest")
		require.ErrorContains(t, err, "method supervisor_getBlockHashByNumber does not exist")
	})
}

func TestQueryFrontendLegacySupervisorRPCNamespace(t *testing.T) {
	client := newTestQueryRPCClient(t, true)

	var result common.Hash
	err := client.Call(&result, "supervisor_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), "latest")
	require.NoError(t, err)
	require.Equal(t, common.HexToHash("0x02"), result)

	err = client.Call(&result, "interop_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), "latest")
	require.NoError(t, err)
	require.Equal(t, common.HexToHash("0x02"), result)
}

func newTestQueryRPCClient(t *testing.T, legacySupervisorNamespace bool) *rpc.Client {
	t.Helper()

	logger := testlog.Logger(t, log.LevelInfo)
	mock := newMockChainIngester()
	mock.AddBlock(eth.BlockID{Hash: common.HexToHash("0x01"), Number: 100})
	mock.AddBlock(eth.BlockID{Hash: common.HexToHash("0x02"), Number: 200})

	backend := NewBackend(context.Background(), BackendParams{
		Logger:         logger,
		Metrics:        metrics.NoopMetrics,
		Chains:         map[eth.ChainID]ChainIngester{eth.ChainIDFromUInt64(testChainA): mock},
		CrossValidator: &mockCrossValidator{},
	})

	service := &Service{log: logger, version: "test", backend: backend}
	require.NoError(t, service.initRPCServer(&Config{
		RPCAddr:                   "127.0.0.1",
		RPCPort:                   0,
		LegacySupervisorNamespace: legacySupervisorNamespace,
	}))
	require.NoError(t, service.rpcServer.Start())
	t.Cleanup(func() {
		_ = service.rpcServer.Stop()
	})

	client, err := rpc.Dial("http://" + service.rpcServer.Endpoint())
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}
