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

func TestNewConfigReadsSuperRootRpcs(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		envName  string
		expected []string
	}{
		{
			name: "PrimaryFlag",
			args: []string{
				"--superroot-rpcs", "http://localhost:8882/superroot-a",
				"--superroot-rpcs", "http://localhost:8883/superroot-b",
			},
			expected: []string{
				"http://localhost:8882/superroot-a",
				"http://localhost:8883/superroot-b",
			},
		},
		{
			name: "LegacyFlagAlias",
			args: []string{
				"--supernode-rpcs", "http://localhost:8882/supernode-a",
				"--supernode-rpcs", "http://localhost:8883/supernode-b",
			},
			expected: []string{
				"http://localhost:8882/supernode-a",
				"http://localhost:8883/supernode-b",
			},
		},
		{
			name:     "PrimaryEnvVar",
			envName:  "OP_PROPOSER_SUPERROOT_RPCS",
			expected: []string{"http://localhost:8882/superroot"},
		},
		{
			name:     "LegacyEnvVarAlias",
			envName:  "OP_PROPOSER_SUPERNODE_RPCS",
			expected: []string{"http://localhost:8882/superroot"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.envName != "" {
				t.Setenv(testCase.envName, testCase.expected[0])
			}

			var cfg *CLIConfig
			app := cli.NewApp()
			app.Flags = proposerFlags.Flags
			app.Action = func(ctx *cli.Context) error {
				cfg = NewConfig(ctx)
				return nil
			}
			err := app.Run(append([]string{"op-proposer"}, testCase.args...))
			require.NoError(t, err)
			require.Equal(t, testCase.expected, cfg.SuperRootRpcs)
		})
	}
}

func TestRollupRpc(t *testing.T) {
	for _, gameType := range preInteropGameTypes {
		t.Run("RequiredWithPreInteropGame", func(t *testing.T) {
			cfg := validConfig()
			cfg.DGFAddress = common.Address{0xaa}.Hex()
			cfg.ProposalInterval = 20
			cfg.RollupRpc = ""
			cfg.SuperRootRpcs = []string{"http://localhost:8882/superroot"}
			cfg.DisputeGameType = gameType
			require.ErrorIs(t, cfg.Check(), ErrMissingRollupRpc)
		})
	}

	t.Run("NotRequiredForOtherGameTypes", func(t *testing.T) {
		cfg := validConfig()
		cfg.DGFAddress = common.Address{0xaa}.Hex()
		cfg.ProposalInterval = 20
		cfg.RollupRpc = ""
		cfg.SuperRootRpcs = []string{"http://localhost:8882/superroot"}
		cfg.DisputeGameType = 492743
		require.NoError(t, cfg.Check())
	})
}

func TestSuperRootRpc(t *testing.T) {
	for _, gameType := range postInteropGameTypes {
		t.Run("RequiredWithPostInteropGame", func(t *testing.T) {
			cfg := validConfig()
			cfg.DGFAddress = common.Address{0xaa}.Hex()
			cfg.ProposalInterval = 20
			cfg.RollupRpc = "http://localhost:8882/rollup"
			cfg.SuperRootRpcs = nil
			cfg.DisputeGameType = gameType
			require.ErrorIs(t, cfg.Check(), ErrMissingSuperRootRpc)
		})

		t.Run("AllowedWithPostInteropGame", func(t *testing.T) {
			cfg := validConfig()
			cfg.DGFAddress = common.Address{0xaa}.Hex()
			cfg.ProposalInterval = 20
			cfg.RollupRpc = ""
			cfg.SuperRootRpcs = []string{"http://localhost:8882/superroot"}
			cfg.DisputeGameType = gameType
			require.NoError(t, cfg.Check())
		})
	}

	t.Run("AllowedForOtherGameTypes", func(t *testing.T) {
		cfg := validConfig()
		cfg.DGFAddress = common.Address{0xaa}.Hex()
		cfg.ProposalInterval = 20
		cfg.RollupRpc = ""
		cfg.SuperRootRpcs = []string{"http://localhost:8882/superroot"}
		cfg.DisputeGameType = 492743
		require.NoError(t, cfg.Check())
	})
}

func TestDisallowRollupAndSuperRootRPC(t *testing.T) {
	cfg := validConfig()
	cfg.ProposalInterval = 20
	cfg.RollupRpc = "http://localhost:8882/rollup"
	cfg.SuperRootRpcs = []string{"http://localhost:8882/superroot"}
	cfg.DisputeGameType = 492743
	require.ErrorIs(t, cfg.Check(), ErrConflictingSource)
}

func TestRequireSomeRPCSourceForUnknownGameTypes(t *testing.T) {
	cfg := validConfig()
	cfg.RollupRpc = ""
	cfg.SuperRootRpcs = nil
	cfg.DisputeGameType = 492743
	require.ErrorIs(t, cfg.Check(), ErrMissingSource)
}

func validConfig() *CLIConfig {
	return &CLIConfig{
		L1EthRpc:                     "http://localhost:8888/l1",
		RollupRpc:                    "http://localhost:8888/l2",
		SuperRootRpcs:                nil,
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
