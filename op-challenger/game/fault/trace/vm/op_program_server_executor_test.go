package vm

import (
	"fmt"
	"log/slog"
	"math/big"
	"strings"
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
			// l2.custom is a boolean flag so can't accept a value after a space
			if args[i] == "--l2.custom" {
				pairs[args[i]] = "true"
				i--
				continue
			}
			pairs[args[i]] = args[i+1]
		}
		return pairs
	}

	oracleCommand := func(t *testing.T, lvl slog.Level, configModifier func(c *Config, inputs *utils.LocalGameInputs)) map[string]string {
		cfg := Config{
			L1:       "http://localhost:8888",
			L1Beacon: "http://localhost:9000",
			L2s:      []string{"http://localhost:9999", "http://localhost:9999/two"},
			Server:   "./bin/mockserver",
		}
		inputs := utils.LocalGameInputs{
			L1Head:           common.Hash{0x11},
			L2Head:           common.Hash{0x22},
			L2OutputRoot:     common.Hash{0x33},
			L2Claim:          common.Hash{0x44},
			L2SequenceNumber: big.NewInt(3333),
		}
		configModifier(&cfg, &inputs)
		executor := NewOpProgramServerExecutor(testlog.Logger(t, lvl))

		args, err := executor.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)
		pairs := toPairs(args)
		// Validate standard options
		require.Equal(t, "--server", pairs[cfg.Server])
		require.Equal(t, cfg.L1, pairs["--l1"])
		require.Equal(t, cfg.L1Beacon, pairs["--l1.beacon"])
		require.Equal(t, strings.Join(cfg.L2s, ","), pairs["--l2"])
		require.Equal(t, dir, pairs["--datadir"])
		require.Equal(t, inputs.L1Head.Hex(), pairs["--l1.head"])
		require.Equal(t, inputs.L2Claim.Hex(), pairs["--l2.claim"])
		require.Equal(t, inputs.L2SequenceNumber.String(), pairs["--l2.blocknumber"])
		return pairs
	}

	t.Run("NoExtras", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {})
		require.NotContains(t, pairs, "--network")
		require.NotContains(t, pairs, "--rollup.config")
		require.NotContains(t, pairs, "--l2.genesis")
	})

	t.Run("WithNetwork", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.Networks = []string{"op-test"}
		})
		require.Equal(t, "op-test", pairs["--network"])
	})

	t.Run("WithMultipleNetworks", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.Networks = []string{"op-test", "op-other"}
		})
		require.Equal(t, "op-test,op-other", pairs["--network"])
	})

	t.Run("WithL2Custom", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.L2Custom = true
		})
		require.Equal(t, "true", pairs["--l2.custom"])
	})

	t.Run("WithRollupConfigPath", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.RollupConfigPaths = []string{"rollup.config.json"}
		})
		require.Equal(t, "rollup.config.json", pairs["--rollup.config"])
	})

	t.Run("WithMultipleRollupConfigPaths", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.RollupConfigPaths = []string{"rollup.config.json", "rollup2.json"}
		})
		require.Equal(t, "rollup.config.json,rollup2.json", pairs["--rollup.config"])
	})

	t.Run("WithL2GenesisPath", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.L2GenesisPaths = []string{"genesis.json"}
		})
		require.Equal(t, "genesis.json", pairs["--l2.genesis"])
	})

	t.Run("WithMultipleL2GenesisPaths", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.L2GenesisPaths = []string{"genesis.json", "genesis2.json"}
		})
		require.Equal(t, "genesis.json,genesis2.json", pairs["--l2.genesis"])
	})

	t.Run("WithAllExtras", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.Networks = []string{"op-test"}
			c.RollupConfigPaths = []string{"rollup.config.json"}
			c.L2GenesisPaths = []string{"genesis.json"}
		})
		require.Equal(t, "op-test", pairs["--network"])
		require.Equal(t, "rollup.config.json", pairs["--rollup.config"])
		require.Equal(t, "genesis.json", pairs["--l2.genesis"])
	})

	t.Run("WithoutL2Head", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(_ *Config, inputs *utils.LocalGameInputs) {
			inputs.L2Head = common.Hash{}
		})
		require.NotContains(t, pairs, "--l2.head")
	})

	t.Run("WithL2Head", func(t *testing.T) {
		val := common.Hash{0xab}
		pairs := oracleCommand(t, log.LvlInfo, func(_ *Config, inputs *utils.LocalGameInputs) {
			inputs.L2Head = val
		})
		require.Equal(t, val.Hex(), pairs["--l2.head"])
	})

	t.Run("WithoutL2OutputRoot", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(_ *Config, inputs *utils.LocalGameInputs) {
			inputs.L2OutputRoot = common.Hash{}
		})
		require.NotContains(t, pairs, "--l2.outputroot")
	})

	t.Run("WithL2OutputRoot", func(t *testing.T) {
		val := common.Hash{0xab}
		pairs := oracleCommand(t, log.LvlInfo, func(_ *Config, inputs *utils.LocalGameInputs) {
			inputs.L2OutputRoot = val
		})
		require.Equal(t, val.Hex(), pairs["--l2.outputroot"])
	})

	t.Run("NilAgreedPrestate", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(_ *Config, inputs *utils.LocalGameInputs) {
			inputs.AgreedPreState = nil
		})
		require.NotContains(t, pairs, "--l2.agreed-prestate")
	})

	t.Run("EmptyAgreedPrestate", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(_ *Config, inputs *utils.LocalGameInputs) {
			inputs.AgreedPreState = []byte{}
		})
		require.NotContains(t, pairs, "--l2.agreed-prestate")
	})

	t.Run("WithAgreedPrestate", func(t *testing.T) {
		val := []byte{1, 6, 53, 42}
		pairs := oracleCommand(t, log.LvlInfo, func(_ *Config, inputs *utils.LocalGameInputs) {
			inputs.AgreedPreState = val
		})
		require.Equal(t, common.Bytes2Hex(val), pairs["--l2.agreed-prestate"])
	})

	t.Run("WithouDepsetConfig", func(t *testing.T) {
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.DepsetConfigPath = ""
		})
		require.NotContains(t, pairs, "--depset.config")
	})

	t.Run("WithL2OutputRoot", func(t *testing.T) {
		val := "depset.json"
		pairs := oracleCommand(t, log.LvlInfo, func(c *Config, _ *utils.LocalGameInputs) {
			c.DepsetConfigPath = val
		})
		require.Equal(t, val, pairs["--depset.config"])
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
			pairs := oracleCommand(t, logTest.level, func(c *Config, _ *utils.LocalGameInputs) {})
			require.Equal(t, pairs["--log.level"], logTest.arg)
		})
	}
}
