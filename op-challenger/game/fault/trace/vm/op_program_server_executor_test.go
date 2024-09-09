package vm

import (
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum/go-ethereum/common"
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

	t.Run("NoExtras", func(t *testing.T) {
		vmConfig := NewOpProgramServerExecutor()

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
	})

	t.Run("WithNetwork", func(t *testing.T) {
		cfg.Network = "op-test"
		vmConfig := NewOpProgramServerExecutor()

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
		require.True(t, slices.Contains(args, "--network"))
	})

	t.Run("WithRollupConfigPath", func(t *testing.T) {
		cfg.RollupConfigPath = "rollup.config"
		vmConfig := NewOpProgramServerExecutor()

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
		require.True(t, slices.Contains(args, "--rollup.config"))
	})

	t.Run("WithL2GenesisPath", func(t *testing.T) {
		cfg.L2GenesisPath = "l2.genesis"
		vmConfig := NewOpProgramServerExecutor()

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
		require.True(t, slices.Contains(args, "--l2.genesis"))
	})

	t.Run("WithAllExtras", func(t *testing.T) {
		cfg.Network = "op-test"
		cfg.RollupConfigPath = "rollup.config"
		cfg.L2GenesisPath = "l2.genesis"
		vmConfig := NewOpProgramServerExecutor()

		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)

		validateStandard(t, args)
		require.True(t, slices.Contains(args, "--network"))
		require.True(t, slices.Contains(args, "--rollup.config"))
		require.True(t, slices.Contains(args, "--l2.genesis"))
	})
}
