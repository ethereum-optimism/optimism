package conductor

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-conductor/flags"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestNewConfigWithoutRegistry(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		testConductorCLIConfig(t, nil, 1000, nil)
	})
	t.Run("legacy settings", func(t *testing.T) {
		t.Setenv("OP_CONDUCTOR_NETWORK", "unknown-network")
		t.Setenv("OP_CONDUCTOR_OVERRIDE_PECTRABLOBSCHEDULE", "0")
		t.Setenv("OP_CONDUCTOR_EXECUTION_HEADER_CACHE_SIZE", "256")
		testConductorCLIConfig(t, []string{
			"--rollup.config", t.TempDir() + "/missing.json",
			"--override.canyon", "0",
			"--override.keep-karst-upgrade-gas=false",
		}, 256, []string{
			"network", "rollup.config", "override.canyon",
			"override.pectrablobschedule", "override.keep-karst-upgrade-gas",
		})
	})
}

func testConductorCLIConfig(t *testing.T, extraArgs []string, cacheSize int, deprecated []string) {
	t.Helper()
	logger, captured := testlog.CaptureLogger(t, slog.LevelWarn)
	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	// urfave/cli mutates scalar flag definitions when applying environment values.
	for i, flag := range app.Flags {
		switch flag := flag.(type) {
		case *cli.IntFlag:
			copy := *flag
			app.Flags[i] = &copy
		case *cli.StringFlag:
			copy := *flag
			app.Flags[i] = &copy
		case *cli.Uint64Flag:
			copy := *flag
			app.Flags[i] = &copy
		}
	}
	app.Action = func(ctx *cli.Context) error {
		cfg, err := NewConfig(ctx, logger)
		require.NoError(t, err)
		require.NoError(t, cfg.Check())
		require.Equal(t, cacheSize, cfg.ExecutionHeaderCacheSize)
		return nil
	}
	args := []string{
		"op-conductor",
		"--consensus.addr", "127.0.0.1",
		"--consensus.port", "0",
		"--raft.server.id", "sequencer",
		"--raft.storage.dir", t.TempDir(),
		"--node.rpc", "http://localhost:9545",
		"--execution.rpc", "http://localhost:8545",
		"--healthcheck.interval", "1",
		"--healthcheck.unsafe-interval", "3",
		"--healthcheck.min-peer-count", "1",
	}
	require.NoError(t, app.Run(append(args, extraArgs...)))
	warnings := captured.FindLogs(testlog.NewMessageFilter("Ignoring deprecated op-conductor flag"))
	require.Len(t, warnings, len(deprecated))
	for _, name := range deprecated {
		require.NotNil(t, captured.FindLog(testlog.NewAttributesFilter("flag", name)))
	}
}

func TestConfigCheckRollupBoostAndNextMutuallyExclusive(t *testing.T) {
	cfg := &Config{
		ConsensusAddr:                 "127.0.0.1",
		ConsensusPort:                 9000,
		RaftServerID:                  "server-1",
		RaftStorageDir:                "/tmp/op-conductor",
		NodeRPC:                       "http://node.example",
		ExecutionRPC:                  "http://exec.example",
		RollupBoostEnabled:            true,
		RollupBoostNextEnabled:        true,
		RollupBoostNextHealthcheckURL: "http://rollupboost.example",
	}

	err := cfg.Check()
	require.Error(t, err)
	require.Contains(t, err.Error(), "only one of rollup-boost or rollup-boost next healthchecks can be enabled")
}
