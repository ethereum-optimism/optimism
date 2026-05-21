package runner

import (
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestServerExecutorForGameType(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	tests := []struct {
		gameType gameTypes.GameType
		want     vm.OracleServerExecutor
	}{
		{gameTypes.CannonGameType, &vm.OpProgramServerExecutor{}},
		{gameTypes.CannonKonaGameType, &vm.KonaExecutor{}},
		{gameTypes.SuperCannonKonaGameType, &vm.KonaSuperExecutor{}},
	}
	for _, test := range tests {
		t.Run(test.gameType.String(), func(t *testing.T) {
			got, err := serverExecutorForGameType(logger, test.gameType)
			require.NoError(t, err)
			require.IsType(t, test.want, got)
		})
	}
}

func TestServerExecutorForGameType_UnknownGameType(t *testing.T) {
	_, err := serverExecutorForGameType(testlog.Logger(t, log.LevelInfo), gameTypes.AlphabetGameType)
	require.Error(t, err)
}

// Guards against regressing super-cannon-kona dispatch into the single-chain
// KonaExecutor, which has a disjoint flag set and does not run super-state
// derivation.
func TestSuperCannonKonaProducesSuperHostCommand(t *testing.T) {
	executor, err := serverExecutorForGameType(testlog.Logger(t, log.LevelInfo), gameTypes.SuperCannonKonaGameType)
	require.NoError(t, err)

	cfg := vm.Config{
		Server:           "/path/to/kona",
		L1:               "http://l1",
		L1Beacon:         "http://beacon",
		L2s:              []string{"http://l2a", "http://l2b"},
		DepsetConfigPath: "/path/to/depset.json",
	}
	inputs := utils.LocalGameInputs{
		L1Head:           common.Hash{0x11},
		AgreedPreState:   []byte{0x01, 0x02, 0x03},
		L2Claim:          common.Hash{0x44},
		L2SequenceNumber: big.NewInt(100),
	}

	args, err := executor.OracleCommand(cfg, "/data", inputs)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(args), 2)
	require.Equal(t, "super", args[1])
	require.True(t, slices.Contains(args, "--agreed-l2-pre-state"), "missing --agreed-l2-pre-state in %v", args)
	require.True(t, slices.Contains(args, "--claimed-l2-post-state"), "missing --claimed-l2-post-state in %v", args)
	require.True(t, slices.Contains(args, "--l2-node-addresses"), "missing --l2-node-addresses in %v", args)
	require.True(t, slices.Contains(args, "--depset-cfg"), "missing --depset-cfg in %v", args)
}
