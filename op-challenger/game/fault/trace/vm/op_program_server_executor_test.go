package vm

import (
	"fmt"
	"log/slog"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestOpProgramFillHostCommand(t *testing.T) {
	dir := "mockdir"
	cfg := Config{
		L1:       "http://localhost:8888",
		L1Beacon: "http://localhost:9000",
		L2:       "http://localhost:9999",
		Server:   "./bin/mockserver",
	}
	inputs := utils.LocalGameInputs{
		L1Head:        common.Hash{0x11},
		L2Head:        common.Hash{0x22},
		L2OutputRoot:  common.Hash{0x33},
		L2Claim:       common.Hash{0x44},
		L2BlockNumber: big.NewInt(3333),
	}

	validateStandard := func(t *testing.T, args []string) {
		require.True(t, slices.Contains(args, "--server"))
		require.True(t, slices.Contains(args, "--l1"))
		require.True(t, slices.Contains(args, "--l1.beacon"))
		require.True(t, slices.Contains(args, "--l2"))
		require.True(t, slices.Contains(args, "--datadir"))
		require.True(t, slices.Contains(args, "--l1.head"))
		require.True(t, slices.Contains(args, "--l2.head"))
		require.True(t, slices.Contains(args, "--l2.outputroot"))
		require.True(t, slices.Contains(args, "--l2.claim"))
		require.True(t, slices.Contains(args, "--l2.blocknumber"))
	}

	toPairs := func(args []string) map[string]string {
		pairs := make(map[string]string, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			pairs[args[i]] = args[i+1]
		}
		return pairs
	}

	t.Run("NoExtras", func(t *testing.T) {
		vmConfig := NewOpProgramServerExecutor(testlog.Logger(t, log.LvlInfo))

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
	})

	t.Run("WithNetwork", func(t *testing.T) {
		cfg.Network = "op-test"
		vmConfig := NewOpProgramServerExecutor(testlog.Logger(t, log.LvlInfo))

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
		require.True(t, slices.Contains(args, "--network"))
	})

	t.Run("WithRollupConfigPath", func(t *testing.T) {
		cfg.RollupConfigPath = "rollup.config"
		vmConfig := NewOpProgramServerExecutor(testlog.Logger(t, log.LvlInfo))

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
		require.True(t, slices.Contains(args, "--rollup.config"))
	})

	t.Run("WithL2GenesisPath", func(t *testing.T) {
		cfg.L2GenesisPath = "l2.genesis"
		vmConfig := NewOpProgramServerExecutor(testlog.Logger(t, log.LvlInfo))

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
		require.True(t, slices.Contains(args, "--l2.genesis"))
	})

	t.Run("WithAllExtras", func(t *testing.T) {
		cfg.Network = "op-test"
		cfg.RollupConfigPath = "rollup.config"
		cfg.L2GenesisPath = "l2.genesis"
		vmConfig := NewOpProgramServerExecutor(testlog.Logger(t, log.LvlInfo))

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
		require.True(t, slices.Contains(args, "--network"))
		require.True(t, slices.Contains(args, "--rollup.config"))
		require.True(t, slices.Contains(args, "--l2.genesis"))
	})

	logTests := []struct {
		level slog.Level
		arg   string
	}{
		{log.LevelTrace, "TRACE"},
		{log.LevelDebug, "DEBUG"},
		{log.LevelInfo, "INFO"},
		{log.LevelWarn, "WARN"},
		{log.LevelError, "ERROR"},
		{log.LevelCrit, "CRIT"},
	}
	for _, logTest := range logTests {
		logTest := logTest
		t.Run(fmt.Sprintf("LogLevel-%v", logTest.arg), func(t *testing.T) {
			vmConfig := NewOpProgramServerExecutor(testlog.Logger(t, logTest.level))

			args, err := vmConfig.OracleCommand(cfg, dir, inputs)
			require.NoError(t, err)

			validateStandard(t, args)

			require.Equal(t, toPairs(args)["--log.level"], logTest.arg)
		})
	}
}
