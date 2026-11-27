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
	warnings := captured.FindLogs(testlog.NewMessageFilter("Ignoring deprecated op-conductor flag. It will be removed in an upcoming release after 2027-03-01. Configure the paired op-node instead."))
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

// TestConfigCheck_HTTPBodyLimit tests the HTTP body limit validation in Config.Check()
func TestConfigCheck_HTTPBodyLimit(t *testing.T) {
	tests := []struct {
		name          string
		bodyLimitMB   int
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid: default 5MB",
			bodyLimitMB: 5,
			expectError: false,
		},
		{
			name:        "valid: 64MB (recommended)",
			bodyLimitMB: 64,
			expectError: false,
		},
		{
			name:        "valid: 128MB (large)",
			bodyLimitMB: 128,
			expectError: false,
		},
		{
			name:        "valid: 10MB",
			bodyLimitMB: 10,
			expectError: false,
		},
		{
			name:        "valid: 4MB",
			bodyLimitMB: 4,
			expectError: false,
		},
		{
			name:        "valid: 1MB",
			bodyLimitMB: 1,
			expectError: false,
		},
		{
			name:          "invalid: 0MB",
			bodyLimitMB:   0,
			expectError:   true,
			errorContains: "HTTP body limit must be greater than 0, got 0MB",
		},
		{
			name:          "invalid: -1MB (negative)",
			bodyLimitMB:   -1,
			expectError:   true,
			errorContains: "HTTP body limit must be greater than 0, got -1MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidConfig(t)
			cfg.HTTPBodyLimitMB = tt.bodyLimitMB

			err := cfg.Check()

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// newValidConfig creates a valid Config for testing
func newValidConfig(t *testing.T) Config {
	return Config{
		ConsensusAddr:   "127.0.0.1",
		ConsensusPort:   50050,
		RaftServerID:    "test-node",
		RaftStorageDir:  t.TempDir(),
		RaftBootstrap:   true,
		NodeRPC:         "http://localhost:8545",
		ExecutionRPC:    "http://localhost:8551",
		Paused:          false,
		HTTPBodyLimitMB: 5, // Default valid value
		HealthCheck: HealthCheckConfig{
			Interval:       1,
			UnsafeInterval: 3,
			SafeInterval:   5,
			MinPeerCount:   1,
		},
	}
}
