package monitor

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type testExtension struct {
	env      ExtensionEnv
	started  bool
	stopped  bool
	observed map[eth.ChainID]int
}

func (e *testExtension) Start(context.Context) error { e.started = true; return nil }
func (e *testExtension) Stop() error                 { e.stopped = true; return nil }

func (e *testExtension) ObserveBlock(chainID eth.ChainID, receipts []*types.Receipt) {
	e.observed[chainID] += len(receipts)
}

func TestExtensionLifecycle(t *testing.T) {
	cfg := &CLIConfig{
		RPCConfig:     oprpc.CLIConfig{ListenAddr: "127.0.0.1"},
		MetricsConfig: opmetrics.CLIConfig{Enabled: true, ListenAddr: "127.0.0.1"},
	}
	ext := &testExtension{observed: make(map[eth.ChainID]int)}
	factory := func(_ context.Context, env ExtensionEnv) (Extension, error) {
		ext.env = env
		return ext, nil
	}

	ms, err := InteropMonitorServiceFromClients(context.Background(), "test", cfg,
		map[eth.ChainID]*sources.EthClient{}, testlog.Logger(t, slog.LevelInfo), WithExtension(factory))
	require.NoError(t, err)
	require.NotNil(t, ext.env.Registry, "the extension must get the served metrics registry")
	require.NotZero(t, ext.env.MessageExpiryWindow)

	require.NoError(t, ms.Start(context.Background()))
	require.True(t, ext.started)
	require.NoError(t, ms.Stop(context.Background()))
	require.True(t, ext.stopped)
}

func TestObservingJobFilter(t *testing.T) {
	ext := &testExtension{observed: make(map[eth.ChainID]int)}
	filtered := 0
	filter := observingJobFilter(func(receipts []*types.Receipt, _ eth.ChainID) []*Job {
		filtered += len(receipts)
		return nil
	}, []BlockObserver{ext})

	chainID := eth.ChainIDFromUInt64(10)
	filter([]*types.Receipt{{}, {}}, chainID)

	require.Equal(t, 2, ext.observed[chainID], "the observer must see every receipt")
	require.Equal(t, 2, filtered, "the wrapped filter must still run")
}
