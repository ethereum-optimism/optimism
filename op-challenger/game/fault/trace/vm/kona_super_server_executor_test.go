package vm

import (
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestKonaSuperExecutorWithWitnessEndpoint(t *testing.T) {
	t.Parallel()
	executor := NewKonaSuperExecutor()
	cfg := Config{
		Server:                            "/path/to/kona",
		L1:                                "http://l1",
		L1Beacon:                          "http://beacon",
		L2s:                               []string{"http://l2a", "http://l2b"},
		EnableExperimentalWitnessEndpoint: true,
	}
	inputs := utils.LocalGameInputs{
		L1Head:           common.Hash{0x11},
		AgreedPreState:   []byte{0x01, 0x02},
		L2Claim:          common.Hash{0x44},
		L2SequenceNumber: big.NewInt(100),
	}

	args, err := executor.OracleCommand(cfg, "/data", inputs)
	require.NoError(t, err)
	require.True(t, slices.Contains(args, "--enable-experimental-witness-endpoint"))
}

func TestKonaSuperExecutorWithoutWitnessEndpoint(t *testing.T) {
	t.Parallel()
	executor := NewKonaSuperExecutor()
	cfg := Config{
		Server:                            "/path/to/kona",
		L1:                                "http://l1",
		L1Beacon:                          "http://beacon",
		L2s:                               []string{"http://l2a", "http://l2b"},
		EnableExperimentalWitnessEndpoint: false,
	}
	inputs := utils.LocalGameInputs{
		L1Head:           common.Hash{0x11},
		AgreedPreState:   []byte{0x01, 0x02},
		L2Claim:          common.Hash{0x44},
		L2SequenceNumber: big.NewInt(100),
	}

	args, err := executor.OracleCommand(cfg, "/data", inputs)
	require.NoError(t, err)
	require.False(t, slices.Contains(args, "--enable-experimental-witness-endpoint"))
}
