package proposer

import (
	"fmt"
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

func TestRootFormatRPCValidation(t *testing.T) {
	for _, gameType := range []uint32{0, 1, 2, 3, 6, 8, 254, 255, 1337} {
		t.Run(fmt.Sprintf("OutputRootGameType-%d-RequiresRollupRPC", gameType), func(t *testing.T) {
			cfg := validConfig()
			cfg.RollupRpc = ""
			cfg.SuperNodeRpcs = []string{"http://localhost:8882/supernode"}
			cfg.DisputeGameType = gameType
			require.ErrorIs(t, cfg.Check(), ErrMissingRollupRpc)
		})
	}

	for _, gameType := range []uint32{5, 9} {
		t.Run(fmt.Sprintf("SuperRootGameType-%d-AcceptsRollupRPC", gameType), func(t *testing.T) {
			cfg := validConfig()
			cfg.RollupRpc = "http://localhost:8882/rollup"
			cfg.SuperNodeRpcs = nil
			cfg.DisputeGameType = gameType
			require.NoError(t, cfg.Check())
		})

		t.Run(fmt.Sprintf("SuperRootGameType-%d-AcceptsSuperNodeRPC", gameType), func(t *testing.T) {
			cfg := validConfig()
			cfg.RollupRpc = ""
			cfg.SuperNodeRpcs = []string{"http://localhost:8882/supernode"}
			cfg.DisputeGameType = gameType
			require.NoError(t, cfg.Check())
		})
	}

	for _, gameType := range []uint32{4, 492743} {
		t.Run(fmt.Sprintf("UnknownGameType-%d-AcceptsRollupRPC", gameType), func(t *testing.T) {
			cfg := validConfig()
			cfg.RollupRpc = "http://localhost:8882/rollup"
			cfg.SuperNodeRpcs = nil
			cfg.DisputeGameType = gameType
			require.NoError(t, cfg.Check())
		})

		t.Run(fmt.Sprintf("UnknownGameType-%d-AcceptsSuperNodeRPC", gameType), func(t *testing.T) {
			cfg := validConfig()
			cfg.RollupRpc = ""
			cfg.SuperNodeRpcs = []string{"http://localhost:8882/supernode"}
			cfg.DisputeGameType = gameType
			require.NoError(t, cfg.Check())
		})
	}
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
