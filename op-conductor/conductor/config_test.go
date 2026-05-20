package conductor

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	conductorFlags "github.com/ethereum-optimism/optimism/op-conductor/flags"
)

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

func TestHealthCheckConfigRequiresInteropReorgLeniencyWindowSize(t *testing.T) {
	cfg := &HealthCheckConfig{
		Interval:             1,
		UnsafeInterval:       2,
		SafeInterval:         3,
		MinPeerCount:         1,
		InteropReorgLeniency: true,
	}

	err := cfg.Check()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing interop reorg leniency window size")
}

func TestNewConfigReadsInteropReorgLeniencyEnvVar(t *testing.T) {
	t.Setenv("OP_CONDUCTOR_BETA_HEALTHCHECK_INTEROP_REORG_LENIENCY", "true")
	t.Setenv("OP_CONDUCTOR_BETA_HEALTHCHECK_INTEROP_REORG_LENIENCY_WINDOW_SIZE", "7")
	t.Cleanup(func() {
		conductorFlags.HealthCheckInteropReorgLeniency.Value = false
		conductorFlags.HealthCheckInteropReorgLeniencyWindowSize.Value = 5
	})

	var cfg *Config
	app := cli.NewApp()
	app.Flags = conductorFlags.Flags
	app.Action = func(ctx *cli.Context) error {
		var err error
		cfg, err = NewConfig(ctx, log.New())
		return err
	}

	err := app.Run([]string{
		"op-conductor",
		"--consensus.addr", "127.0.0.1",
		"--consensus.port", "0",
		"--raft.server.id", "server-1",
		"--raft.storage.dir", t.TempDir(),
		"--node.rpc", "http://node.example",
		"--execution.rpc", "http://exec.example",
		"--healthcheck.interval", "1",
		"--healthcheck.unsafe-interval", "2",
		"--healthcheck.min-peer-count", "1",
		"--network", "op-sepolia",
	})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.True(t, cfg.HealthCheck.InteropReorgLeniency)
	require.Equal(t, uint64(7), cfg.HealthCheck.InteropReorgLeniencyWindowSize)
}
