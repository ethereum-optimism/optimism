package supervisor

import (
	"context"
	"testing"
	"time"

	"github.com/HashKeyChain/verse/op-node/rollup"
	"github.com/HashKeyChain/verse/op-service/eth"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/HashKeyChain/verse/op-service/dial"
	oplog "github.com/HashKeyChain/verse/op-service/log"
	opmetrics "github.com/HashKeyChain/verse/op-service/metrics"
	"github.com/HashKeyChain/verse/op-service/oppprof"
	oprpc "github.com/HashKeyChain/verse/op-service/rpc"
	"github.com/HashKeyChain/verse/op-service/testlog"
	"github.com/HashKeyChain/verse/op-supervisor/config"
	"github.com/HashKeyChain/verse/op-supervisor/supervisor/backend/depset"
	"github.com/HashKeyChain/verse/op-supervisor/supervisor/types"
)

func TestSupervisorService(t *testing.T) {
	depSet, err := depset.NewStaticConfigDependencySet(make(map[eth.ChainID]*depset.StaticConfigDependency))
	require.NoError(t, err)
	rollupConfigSet := depset.StaticRollupConfigSetFromRollupConfigMap(make(map[eth.ChainID]*rollup.Config), depset.StaticTimestamp(0))
	fullCfgSet, err := depset.NewFullConfigSetMerged(rollupConfigSet, depSet)
	require.NoError(t, err)

	cfg := &config.Config{
		Version: "",
		LogConfig: oplog.CLIConfig{
			Level:  log.LevelError,
			Color:  false,
			Format: oplog.FormatLogFmt,
		},
		MetricsConfig: opmetrics.CLIConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1",
			ListenPort: 0, // pick a port automatically
		},
		PprofConfig: oppprof.CLIConfig{
			ListenEnabled:   true,
			ListenAddr:      "127.0.0.1",
			ListenPort:      0, // pick a port automatically
			ProfileType:     "",
			ProfileDir:      "",
			ProfileFilename: "",
		},
		RPC: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0, // pick a port automatically
			EnableAdmin: true,
		},
		FullConfigSetSource: fullCfgSet,
		MockRun:             true,
	}
	logger := testlog.Logger(t, log.LevelError)
	supervisor, err := SupervisorFromConfig(context.Background(), cfg, logger)
	require.NoError(t, err)
	require.NoError(t, supervisor.Start(context.Background()), "start service")
	// run some RPC tests against the service with the mock backend
	{
		endpoint := "http://" + supervisor.rpcServer.Endpoint()
		t.Logf("dialing %s", endpoint)
		cl, err := dial.DialRPCClientWithTimeout(context.Background(), time.Second*5, logger, endpoint)
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		err = cl.CallContext(ctx, nil, "supervisor_checkAccessList",
			[]common.Hash{}, types.CrossUnsafe, types.ExecutingDescriptor{
				Timestamp: 1234568, ChainID: eth.ChainIDFromUInt64(123)})
		cancel()
		require.NoError(t, err)
		cl.Close()
	}
	require.NoError(t, supervisor.Stop(context.Background()), "stop service")
}
