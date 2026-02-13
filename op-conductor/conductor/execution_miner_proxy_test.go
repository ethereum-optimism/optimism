package conductor

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	clientmocks "github.com/ethereum-optimism/optimism/op-conductor/client/mocks"
	consensusmocks "github.com/ethereum-optimism/optimism/op-conductor/consensus/mocks"
	healthmocks "github.com/ethereum-optimism/optimism/op-conductor/health/mocks"
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/mockrpc"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// TestSetMaxDASize tests the SetMaxDASize method of the ExecutionMinerProxyBackend
// It ensures that the proxy is transparently proxying the call to the execution engine
func TestSetMaxDASize(t *testing.T) {
	t.Run("compliant sequencer", func(t *testing.T) {
		testSetMaxDASize(t, true, false)
	})
	t.Run("non-compliant sequencer", func(t *testing.T) {
		testSetMaxDASize(t, false, false)
	})
	t.Run("sequencer down", func(t *testing.T) {
		testSetMaxDASize(t, true, true)
	})
}

func testSetMaxDASize(t *testing.T, compliantSequencer bool, sequencerDown bool) {
	ctx := context.Background()
	var expectationsFile string
	if compliantSequencer {
		expectationsFile = "testdata/compliant-sequencer.json"
	} else {
		expectationsFile = "testdata/non-compliant-sequencer.json"
	}

	sequencer := mockrpc.NewMockRPC(t, testlog.Logger(t, slog.LevelDebug), mockrpc.WithExpectationsFile(t, expectationsFile))
	endpoint := sequencer.Endpoint()

	config := mockConfig(t)
	config.ExecutionRPC = endpoint
	config.NodeRPC = endpoint // this won't be used but needs to be set to get the conductor to init properly
	config.RPCEnableProxy = true
	config.RPC.ListenAddr = "localhost"
	config.RPC.ListenPort = 0 // Let the system pick a random port, which we will inspect later

	logger, logs := testlog.CaptureLogger(t, slog.LevelDebug)

	conductor, err := NewOpConductor(
		ctx,
		&config,
		logger,
		&metrics.NoopMetricsImpl{},
		"test-version",
		&clientmocks.SequencerControl{}, // not used in this test
		&consensusmocks.Consensus{},     // not used in this test
		&healthmocks.HealthMonitor{},    // not used in this test
	)
	require.NoError(t, err)

	// Start the RPC server part of the conductor
	err = conductor.rpcServer.Start()
	require.NoError(t, err)
	defer func() { _ = conductor.rpcServer.Stop() }()

	port, err := conductor.rpcServer.Port()
	require.NoError(t, err)
	t.Log("RPC server listening on port:", port)

	url := fmt.Sprintf("http://localhost:%d", port)

	rpcClient, err := rpc.Dial(url)
	require.NoError(t, err)
	defer rpcClient.Close()

	if sequencerDown {
		require.NoError(t, sequencer.Close())
	}

	var result bool
	err = rpcClient.CallContext(ctx, &result, "miner_setMaxDASize", "0x1", "0x2")

	if sequencerDown {
		require.Error(t, err)
		expectedLog := "proxy miner_setMaxDASize call failed"
		r := logs.FindLog(testlog.NewMessageContainsFilter(expectedLog))
		require.NotNil(t, r, "could not find log message containing '%s'", expectedLog)
		return
	}

	if compliantSequencer {
		require.NoError(t, err)
		require.True(t, result)
		expectedLog := "successfully proxied miner_setMaxDASize call"
		r := logs.FindLog(testlog.NewMessageContainsFilter(expectedLog))
		require.NotNil(t, r, "could not find log message containing '%s'", expectedLog)
	} else {
		require.Error(t, err)
		require.Contains(t, err.Error(), "Method not found")
		expectedLog := "proxy miner_setMaxDASize call returned an RPC error"
		r := logs.FindLog(testlog.NewMessageContainsFilter(expectedLog))
		require.NotNil(t, r, "could not find log message containing '%s'", expectedLog)
	}
}
