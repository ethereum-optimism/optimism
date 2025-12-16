package vm

import (
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSuperKonaFillHostCommand(t *testing.T) {
	dir := "mockdir"
	cfg := Config{
		L1:            "http://localhost:8888",
		L1Beacon:      "http://localhost:9000",
		L2s:           []string{"http://localhost:9999", "http://localhost:9998"},
		Server:        "./bin/mockserver",
		Networks:      []string{"op-mainnet", "uni-mainnet"},
		L1GenesisPath: "mockdir/l1-genesis-1.json",
	}
	inputs := utils.LocalGameInputs{
		L1Head:           common.Hash{0x11},
		AgreedPreState:   common.FromHex("0x33"),
		L2Claim:          common.Hash{0x44},
		L2SequenceNumber: big.NewInt(3333),
	}
	vmConfig := NewKonaSuperExecutor()

	args, err := vmConfig.OracleCommand(cfg, dir, inputs)
	require.NoError(t, err)

	require.True(t, slices.Contains(args, "super"))
	require.True(t, slices.Contains(args, "--server"))
	require.True(t, slices.Contains(args, "--l1-node-address"))
	require.True(t, slices.Contains(args, "--l1-beacon-address"))
	require.True(t, slices.Contains(args, "--l2-node-addresses"))
	require.True(t, slices.Contains(args, "--l1-head"))
	require.True(t, slices.Contains(args, "--agreed-l2-pre-state"))
	require.True(t, slices.Contains(args, "--claimed-l2-post-state"))
	require.True(t, slices.Contains(args, "--claimed-l2-timestamp"))
	require.True(t, slices.Contains(args, "--data-dir"))
	require.True(t, slices.Contains(args, "--l1-config-paths"))
}
