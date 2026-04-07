package filter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestBlockSelectorUnmarshalJSON(t *testing.T) {
	t.Run("latest", func(t *testing.T) {
		var selector BlockSelector
		require.NoError(t, json.Unmarshal([]byte(`"latest"`), &selector))
		require.True(t, selector.Latest())
	})

	t.Run("json number", func(t *testing.T) {
		var selector BlockSelector
		require.NoError(t, json.Unmarshal([]byte(`123`), &selector))
		require.False(t, selector.Latest())
		require.Equal(t, uint64(123), selector.Number())
	})

	t.Run("quoted decimal", func(t *testing.T) {
		var selector BlockSelector
		require.NoError(t, json.Unmarshal([]byte(`"123"`), &selector))
		require.Equal(t, uint64(123), selector.Number())
	})

	t.Run("quoted hex", func(t *testing.T) {
		var selector BlockSelector
		require.NoError(t, json.Unmarshal([]byte(`"0x7b"`), &selector))
		require.Equal(t, uint64(123), selector.Number())
	})

	t.Run("invalid", func(t *testing.T) {
		var selector BlockSelector
		require.Error(t, json.Unmarshal([]byte(`"safe"`), &selector))
	})
}

func TestQueryFrontendGetBlockHashByNumberRPC(t *testing.T) {
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

	server := oprpc.NewServer(
		"127.0.0.1",
		0,
		"test",
		oprpc.WithLogger(logger),
	)
	server.AddAPI(rpc.API{
		Namespace: "supervisor",
		Service:   &QueryFrontend{backend: backend},
	})

	require.NoError(t, server.Start())
	t.Cleanup(func() {
		_ = server.Stop()
	})

	client, err := rpc.Dial("http://" + server.Endpoint())
	require.NoError(t, err)
	t.Cleanup(client.Close)

	t.Run("latest selector", func(t *testing.T) {
		var result common.Hash
		err := client.Call(&result, "supervisor_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), "latest")
		require.NoError(t, err)
		require.Equal(t, common.HexToHash("0x02"), result)
	})

	t.Run("numeric selector", func(t *testing.T) {
		var result common.Hash
		err := client.Call(&result, "supervisor_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), uint64(100))
		require.NoError(t, err)
		require.Equal(t, common.HexToHash("0x01"), result)
	})

	t.Run("missing block", func(t *testing.T) {
		var result common.Hash
		err := client.Call(&result, "supervisor_getBlockHashByNumber", eth.ChainIDFromUInt64(testChainA), uint64(999))
		require.ErrorContains(t, err, "not found")
	})
}
