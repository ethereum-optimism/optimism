package vm

import (
	"fmt"
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestOpProgramFillHostCommand(t *testing.T) {
	dir := "mockdir"

	toPairs := func(args []string) map[string]string {
		pairs := make(map[string]string, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			pairs[args[i]] = args[i+1]
		}
		return pairs
	}

	oracleCommand := func(t *testing.T, lvl slog.Level, configModifier func(c *Config)) map[string]string {
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
		configModifier(&cfg)
		executor := NewOpProgramServerExecutor(testlog.Logger(t, lvl))

		args, err := executor.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)
		pairs := toPairs(args)
		// Validate standard options
		require.Equal(t, "--server", pairs[cfg.Server])
		require.Equal(t, cfg.L1, pairs["--l1"])
		require.Equal(t, cfg.L1Beacon, pairs["--l1.beacon"])
		require.Equal(t, cfg.L2, pairs["--l2"])
		require.Equal(t, dir, pairs["--datadir"])
		require.Equal(t, inputs.L1Head.Hex(), pairs["--l1.head"])
		require.Equal(t, inputs.L2Head.Hex(), pairs["--l2.head"])
		require.Equal(t, inputs.L2OutputRoot.Hex(), pairs["--l2.outputroot"])
		require.Equal(t, inputs.L2Claim.Hex(), pairs["--l2.claim"])
		require.Equal(t, inputs.L2BlockNumber.String(), pairs["--l2.blocknumber"])
		return pairs
	}

	t.Run("NoExtras", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config) {})
		require.NotContains(t, pairs, "--network")
		require.NotContains(t, pairs, "--rollup.config")
		require.NotContains(t, pairs, "--l2.genesis")
	})

	t.Run("WithNetwork", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config) {
			c.Network = "op-test"
		})
		require.Equal(t, "op-test", pairs["--network"])
	})

	t.Run("WithRollupConfigPath", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config) {
			c.RollupConfigPath = "rollup.config.json"
		})
		require.Equal(t, "rollup.config.json", pairs["--rollup.config"])
	})

	t.Run("WithL2GenesisPath", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config) {
			c.L2GenesisPath = "genesis.json"
		})
		require.Equal(t, "genesis.json", pairs["--l2.genesis"])
	})

	t.Run("WithAllExtras", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config) {
			c.Network = "op-test"
			c.RollupConfigPath = "rollup.config.json"
			c.L2GenesisPath = "genesis.json"
		})
		require.Equal(t, "op-test", pairs["--network"])
		require.Equal(t, "rollup.config.json", pairs["--rollup.config"])
		require.Equal(t, "genesis.json", pairs["--l2.genesis"])
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
			pairs := oracleCommand(t, logTest.level, func(c *Config) {})
			require.Equal(t, pairs["--log.level"], logTest.arg)
		})
	}
}
