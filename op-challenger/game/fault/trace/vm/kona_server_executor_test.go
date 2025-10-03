package vm

import (
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestKonaFillHostCommand(t *testing.T) {
	dir := "mockdir"
	cfg := Config{
		L1:            "http://localhost:8888",
		L1Beacon:      "http://localhost:9000",
		L2s:           []string{"http://localhost:9999"},
		Server:        "./bin/mockserver",
		Networks:      []string{"op-mainnet"},
		L1GenesisPath: "mockdir/l1-genesis-1.json",
	}
	inputs := utils.LocalGameInputs{
		L1Head:           common.Hash{0x11},
		L2Head:           common.Hash{0x22},
		L2OutputRoot:     common.Hash{0x33},
		L2Claim:          common.Hash{0x44},
		L2SequenceNumber: big.NewInt(3333),
	}
	vmConfig := NewKonaExecutor()

	args, err := vmConfig.OracleCommand(cfg, dir, inputs)
	require.NoError(t, err)

	require.True(t, slices.Contains(args, "single"))
	require.True(t, slices.Contains(args, "--server"))
	require.True(t, slices.Contains(args, "--l1-node-address"))
	require.True(t, slices.Contains(args, "--l1-beacon-address"))
	require.True(t, slices.Contains(args, "--l2-node-address"))
	require.True(t, slices.Contains(args, "--data-dir"))
	require.True(t, slices.Contains(args, "--l2-chain-id"))
	require.True(t, slices.Contains(args, "--l1-head"))
	require.True(t, slices.Contains(args, "--agreed-l2-head-hash"))
	require.True(t, slices.Contains(args, "--agreed-l2-output-root"))
	require.True(t, slices.Contains(args, "--claimed-l2-output-root"))
	require.True(t, slices.Contains(args, "--claimed-l2-block-number"))
	require.True(t, slices.Contains(args, "--l1-config-path"))
}

func TestKonaL1BeaconSkipBlobVerification(t *testing.T) {
	dir := "mockdir"
	baseCfg := Config{
		L1:            "http://localhost:8888",
		L1Beacon:      "http://localhost:9000",
		L2s:           []string{"http://localhost:9999"},
		Server:        "./bin/mockserver",
		Networks:      []string{"op-mainnet"},
		L1GenesisPath: "mockdir/l1-genesis-1.json",
	}
	inputs := utils.LocalGameInputs{
		L1Head:           common.Hash{0x11},
		L2Head:           common.Hash{0x22},
		L2OutputRoot:     common.Hash{0x33},
		L2Claim:          common.Hash{0x44},
		L2SequenceNumber: big.NewInt(3333),
	}

	t.Run("NotIncludedByDefault", func(t *testing.T) {
		cfg := baseCfg
		cfg.L1BeaconSkipBlobVerification = false
		vmConfig := NewKonaExecutor()
		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)
		require.False(t, slices.Contains(args, "--l1-beacon-skip-blob-verification"))
	})

	t.Run("IncludedWhenTrue", func(t *testing.T) {
		cfg := baseCfg
		cfg.L1BeaconSkipBlobVerification = true
		vmConfig := NewKonaExecutor()
		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)
		require.True(t, slices.Contains(args, "--l1-beacon-skip-blob-verification"))
	})
}

func TestKonaSuperL1BeaconSkipBlobVerification(t *testing.T) {
	dir := "mockdir"
	baseCfg := Config{
		L1:                "http://localhost:8888",
		L1Beacon:          "http://localhost:9000",
		L2s:               []string{"http://localhost:9999", "http://localhost:9998"},
		Server:            "./bin/mockserver",
		RollupConfigPaths: []string{"rollup1.json", "rollup2.json"},
		L1GenesisPath:     "mockdir/l1-genesis-1.json",
	}
	inputs := utils.LocalGameInputs{
		L1Head:           common.Hash{0x11},
		AgreedPreState:   []byte{1, 2, 3, 4},
		L2Claim:          common.Hash{0x44},
		L2SequenceNumber: big.NewInt(3333),
	}

	t.Run("NotIncludedByDefault", func(t *testing.T) {
		cfg := baseCfg
		cfg.L1BeaconSkipBlobVerification = false
		vmConfig := NewKonaSuperExecutor()
		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)
		require.False(t, slices.Contains(args, "--l1-beacon-skip-blob-verification"))
	})

	t.Run("IncludedWhenTrue", func(t *testing.T) {
		cfg := baseCfg
		cfg.L1BeaconSkipBlobVerification = true
		vmConfig := NewKonaSuperExecutor()
		args, err := vmConfig.OracleCommand(cfg, dir, inputs)
		require.NoError(t, err)
		require.True(t, slices.Contains(args, "--l1-beacon-skip-blob-verification"))
	})
}
