package proposer

import (
	"testing"

	proposerFlags "github.com/ethereum-optimism/optimism/op-proposer/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestValidConfigIsValid(t *testing.T) {
	cfg := validConfig()
	require.NoError(t, cfg.Check())
}

func TestNewConfigReadsSuperNodeRpcs(t *testing.T) {
	var cfg *CLIConfig
	app := cli.NewApp()
	app.Flags = proposerFlags.Flags
	app.Action = func(ctx *cli.Context) error {
		cfg = NewConfig(ctx)
		return nil
	}
	err := app.Run([]string{
		"op-proposer",
		"--supernode-rpcs", "http://localhost:8882/supernode-a",
		"--supernode-rpcs", "http://localhost:8883/supernode-b",
	})
	require.NoError(t, err)
	require.Equal(t, []string{
		"http://localhost:8882/supernode-a",
		"http://localhost:8883/supernode-b",
	}, cfg.SuperNodeRpcs)
}

func TestNewConfigReadsAutoDisputeGameType(t *testing.T) {
	var cfg *CLIConfig
	app := cli.NewApp()
	app.Flags = proposerFlags.Flags
	app.Action = func(ctx *cli.Context) error {
		cfg = NewConfig(ctx)
		return nil
	}
	err := app.Run([]string{"op-proposer"})
	require.NoError(t, err)
	require.True(t, cfg.DisputeGameTypeAuto)
	require.Equal(t, uint32(0), cfg.DisputeGameType)
}

func TestNewConfigReadsNumericDisputeGameType(t *testing.T) {
	var cfg *CLIConfig
	app := cli.NewApp()
	app.Flags = proposerFlags.Flags
	app.Action = func(ctx *cli.Context) error {
		cfg = NewConfig(ctx)
		return nil
	}
	err := app.Run([]string{
		"op-proposer",
		"--game-type", "9",
	})
	require.NoError(t, err)
	require.False(t, cfg.DisputeGameTypeAuto)
	require.Equal(t, uint32(9), cfg.DisputeGameType)
}

func TestNewConfigReadsNetworkAndSystemConfig(t *testing.T) {
	var cfg *CLIConfig
	app := cli.NewApp()
	app.Flags = proposerFlags.Flags
	app.Action = func(ctx *cli.Context) error {
		cfg = NewConfig(ctx)
		return nil
	}
	systemConfig := common.Address{0xab, 0xcd}.Hex()
	err := app.Run([]string{
		"op-proposer",
		"--network", "op-sepolia",
		"--system-config-address", systemConfig,
	})
	require.NoError(t, err)
	require.Equal(t, "op-sepolia", cfg.Network)
	require.Equal(t, systemConfig, cfg.SystemConfigAddress)
}

func TestInvalidDisputeGameType(t *testing.T) {
	cfg := validConfig()
	cfg.DisputeGameTypeRaw = "invalid"
	require.ErrorContains(t, cfg.Check(), "invalid `game-type`")
}

func TestRollupRpc(t *testing.T) {
	for _, gameType := range preInteropGameTypes {
		t.Run("RequiredWithPreInteropGame", func(t *testing.T) {
			cfg := validConfig()
			cfg.DGFAddress = common.Address{0xaa}.Hex()
			cfg.ProposalInterval = 20
			cfg.RollupRpc = ""
			cfg.SuperNodeRpcs = []string{"http://localhost:8882/supernode"}
			cfg.DisputeGameType = gameType
			require.ErrorIs(t, cfg.Check(), ErrMissingRollupRpc)
		})
	}

	t.Run("NotRequiredForOtherGameTypes", func(t *testing.T) {
		cfg := validConfig()
		cfg.DGFAddress = common.Address{0xaa}.Hex()
		cfg.ProposalInterval = 20
		cfg.RollupRpc = ""
		cfg.SuperNodeRpcs = []string{"http://localhost:8882/supernode"}
		cfg.DisputeGameType = 492743
		require.NoError(t, cfg.Check())
	})
}

func TestSuperNodeRpc(t *testing.T) {
	for _, gameType := range postInteropGameTypes {
		t.Run("RequiredWithPostInteropGame", func(t *testing.T) {
			cfg := validConfig()
			cfg.DGFAddress = common.Address{0xaa}.Hex()
			cfg.ProposalInterval = 20
			cfg.RollupRpc = "http://localhost:8882/rollup"
			cfg.SuperNodeRpcs = nil
			cfg.DisputeGameType = gameType
			require.ErrorIs(t, cfg.Check(), ErrMissingSuperNodeRpc)
		})

		t.Run("AllowedWithPostInteropGame", func(t *testing.T) {
			cfg := validConfig()
			cfg.DGFAddress = common.Address{0xaa}.Hex()
			cfg.ProposalInterval = 20
			cfg.RollupRpc = ""
			cfg.SuperNodeRpcs = []string{"http://localhost:8882/supernode"}
			cfg.DisputeGameType = gameType
			require.NoError(t, cfg.Check())
		})
	}

	t.Run("AllowedForOtherGameTypes", func(t *testing.T) {
		cfg := validConfig()
		cfg.DGFAddress = common.Address{0xaa}.Hex()
		cfg.ProposalInterval = 20
		cfg.RollupRpc = ""
		cfg.SuperNodeRpcs = []string{"http://localhost:8882/supernode"}
		cfg.DisputeGameType = 492743
		require.NoError(t, cfg.Check())
	})
}

func TestDisallowRollupAndSuperNodeRPC(t *testing.T) {
	cfg := validConfig()
	cfg.ProposalInterval = 20
	cfg.RollupRpc = "http://localhost:8882/rollup"
	cfg.SuperNodeRpcs = []string{"http://localhost:8882/supernode"}
	cfg.DisputeGameType = 492743
	require.ErrorIs(t, cfg.Check(), ErrConflictingSource)
}

func TestRequireSomeRPCSourceForUnknownGameTypes(t *testing.T) {
	cfg := validConfig()
	cfg.RollupRpc = ""
	cfg.SuperNodeRpcs = nil
	cfg.DisputeGameType = 492743
	require.ErrorIs(t, cfg.Check(), ErrMissingSource)
}

func TestAutoGameTypeAllowsAnySingleRPCSource(t *testing.T) {
	cfg := validConfig()
	cfg.DisputeGameTypeRaw = "auto"
	cfg.DisputeGameTypeAuto = true
	cfg.SystemConfigAddress = common.Address{0x11}.Hex()
	cfg.RollupRpc = ""
	cfg.SuperNodeRpcs = []string{"http://localhost:8882/supernode"}
	require.NoError(t, cfg.Check())
}

func TestAutoGameTypeRequiresSystemConfigOrNetwork(t *testing.T) {
	cfg := validConfig()
	cfg.DisputeGameTypeRaw = "auto"
	cfg.DisputeGameTypeAuto = true
	cfg.SystemConfigAddress = ""
	cfg.Network = ""
	require.ErrorContains(t, cfg.Check(), "`SystemConfig` or `network` is required")
}

func TestAutoGameTypeAllowsSystemConfig(t *testing.T) {
	cfg := validConfig()
	cfg.DGFAddress = ""
	cfg.SystemConfigAddress = common.Address{0x12}.Hex()
	cfg.DisputeGameTypeRaw = "auto"
	cfg.DisputeGameTypeAuto = true
	require.NoError(t, cfg.Check())
}

func TestAutoGameTypeAllowsNetwork(t *testing.T) {
	cfg := validConfig()
	cfg.DGFAddress = ""
	cfg.Network = networkWithSystemConfig(t)
	cfg.DisputeGameTypeRaw = "auto"
	cfg.DisputeGameTypeAuto = true
	require.NoError(t, cfg.Check())
}

func validConfig() *CLIConfig {
	return &CLIConfig{
		L1EthRpc:                     "http://localhost:8888/l1",
		RollupRpc:                    "http://localhost:8888/l2",
		SuperNodeRpcs:                nil,
		PollInterval:                 100,
		AllowNonFinalized:            false,
		TxMgrConfig:                  txmgr.NewCLIConfig("http://localhost:8888/l1", txmgr.DefaultBatcherFlagValues),
		RPCConfig:                    oprpc.DefaultCLIConfig(),
		LogConfig:                    oplog.DefaultCLIConfig(),
		MetricsConfig:                opmetrics.DefaultCLIConfig(),
		PprofConfig:                  oppprof.DefaultCLIConfig(),
		DGFAddress:                   common.Address{0xaa, 0xbb, 0xcc}.Hex(),
		ProposalInterval:             50,
		DisputeGameType:              0,
		ActiveSequencerCheckDuration: 0,
		WaitNodeSync:                 false,
	}
}
