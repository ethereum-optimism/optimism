package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/config"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestV2Service(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MetricsConfig.Enabled = true
	cfg.PprofConfig.ListenEnabled = true
	// pick a port automatically
	cfg.MetricsConfig.ListenPort = 0
	cfg.PprofConfig.ListenPort = 0
	cfg.RPC.ListenPort = 0

	cfg.Datadir = t.TempDir()

	logger := testlog.Logger(t, log.LevelError)
	opn, err := FromConfig(context.Background(), cfg, logger)
	require.NoError(t, err)
	require.NoError(t, opn.Start(context.Background()), "start service")
	// run some RPC tests against the service with the mock backend
	{
		endpoint := "http://" + opn.rpcServer.Endpoint()
		t.Logf("dialing %s", endpoint)
		cl, err := dial.DialRPCClientWithTimeout(context.Background(), time.Second*5, logger, endpoint)
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		err = cl.CallContext(ctx, nil, "supervisor_checkAccessList",
			[]common.Hash{}, types.CrossUnsafe, types.ExecutingDescriptor{
				Timestamp: 1234568, ChainID: eth.ChainIDFromUInt64(123)})
		cancel()
		var errJson rpc.Error
		require.ErrorAs(t, err, &errJson)
		require.Equal(t, 123, errJson.ErrorCode()) // TODO error codes
		cl.Close()
	}
	require.NoError(t, opn.Stop(context.Background()), "stop service")
}
