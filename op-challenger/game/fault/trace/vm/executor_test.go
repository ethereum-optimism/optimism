package vm

import (
	"context"
	"math"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestGenerateProof(t *testing.T) {
	input := "starting.json"
	tempDir := t.TempDir()
	dir := filepath.Join(tempDir, "gameDir")
	cfg := Config{
		VmType:       "test",
		L1:           "http://localhost:8888",
		L1Beacon:     "http://localhost:9000",
		L2:           "http://localhost:9999",
		VmBin:        "./bin/testvm",
		Server:       "./bin/testserver",
		Network:      "op-test",
		SnapshotFreq: 500,
		InfoFreq:     900,
	}
	prestate := "pre.json"

	inputs := utils.LocalGameInputs{
		L1Head:        common.Hash{0x11},
		L2Head:        common.Hash{0x22},
		L2OutputRoot:  common.Hash{0x33},
		L2Claim:       common.Hash{0x44},
		L2BlockNumber: big.NewInt(3333),
	}
	captureExec := func(t *testing.T, cfg Config, proofAt uint64) (string, string, map[string]string) {
		m := &stubVmMetrics{}
		executor := NewExecutor(testlog.Logger(t, log.LevelInfo), m, cfg, NewOpProgramServerExecutor(testlog.Logger(t, log.LvlInfo)), prestate, inputs)
		executor.selectSnapshot = func(logger log.Logger, dir string, absolutePreState string, i uint64, binary bool) (string, error) {
			return input, nil
		}
		var binary string
		var subcommand string
		args := make(map[string]string)
		executor.cmdExecutor = func(ctx context.Context, l log.Logger, b string, a ...string) error {
			binary = b
			subcommand = a[0]
			for i := 1; i < len(a); {
				if a[i] == "--" {
					// Skip over the divider between vm and server program
					i += 1
					continue
				}
				args[a[i]] = a[i+1]
				i += 2
			}
			return nil
		}
		err := executor.GenerateProof(context.Background(), dir, proofAt)
		require.NoError(t, err)
		require.Equal(t, 1, m.executionTimeRecordCount, "Should record vm execution time")
		return binary, subcommand, args
	}

	t.Run("Network", func(t *testing.T) {
		cfg.Network = "mainnet"
		cfg.RollupConfigPath = ""
		cfg.L2GenesisPath = ""
		binary, subcommand, args := captureExec(t, cfg, 150_000_000)
		require.DirExists(t, filepath.Join(dir, PreimagesDir))
		require.DirExists(t, filepath.Join(dir, utils.ProofsDir))
		require.DirExists(t, filepath.Join(dir, SnapsDir))
		require.Equal(t, cfg.VmBin, binary)
		require.Equal(t, "run", subcommand)
		require.Equal(t, input, args["--input"])
		require.Contains(t, args, "--meta")
		require.Equal(t, "", args["--meta"])
		require.Equal(t, FinalStatePath(dir, cfg.BinarySnapshots), args["--output"])
		require.Equal(t, "=150000000", args["--proof-at"])
		require.Equal(t, "=150000001", args["--stop-at"])
		require.Equal(t, "%500", args["--snapshot-at"])
		require.Equal(t, "%900", args["--info-at"])
		// Slight quirk of how we pair off args
		// The server binary winds up as the key and the first arg --server as the value which has no value
		// Then everything else pairs off correctly again
		require.Equal(t, "--server", args[cfg.Server])
		require.Equal(t, cfg.L1, args["--l1"])
		require.Equal(t, cfg.L1Beacon, args["--l1.beacon"])
		require.Equal(t, cfg.L2, args["--l2"])
		require.Equal(t, filepath.Join(dir, PreimagesDir), args["--datadir"])
		require.Equal(t, filepath.Join(dir, utils.ProofsDir, "%d.json.gz"), args["--proof-fmt"])
		require.Equal(t, filepath.Join(dir, SnapsDir, "%d.json.gz"), args["--snapshot-fmt"])
		require.Equal(t, cfg.Network, args["--network"])
		require.NotContains(t, args, "--rollup.config")
		require.NotContains(t, args, "--l2.genesis")

		// Local game inputs
		require.Equal(t, inputs.L1Head.Hex(), args["--l1.head"])
		require.Equal(t, inputs.L2Head.Hex(), args["--l2.head"])
		require.Equal(t, inputs.L2OutputRoot.Hex(), args["--l2.outputroot"])
		require.Equal(t, inputs.L2Claim.Hex(), args["--l2.claim"])
		require.Equal(t, "3333", args["--l2.blocknumber"])
	})

	t.Run("RollupAndGenesis", func(t *testing.T) {
		cfg.Network = ""
		cfg.RollupConfigPath = "rollup.json"
		cfg.L2GenesisPath = "genesis.json"
		_, _, args := captureExec(t, cfg, 150_000_000)
		require.NotContains(t, args, "--network")
		require.Equal(t, cfg.RollupConfigPath, args["--rollup.config"])
		require.Equal(t, cfg.L2GenesisPath, args["--l2.genesis"])
	})

	t.Run("NoStopAtWhenProofIsMaxUInt", func(t *testing.T) {
		cfg.Network = "mainnet"
		cfg.RollupConfigPath = "rollup.json"
		cfg.L2GenesisPath = "genesis.json"
		_, _, args := captureExec(t, cfg, math.MaxUint64)
		// stop-at would need to be one more than the proof step which would overflow back to 0
		// so expect that it will be omitted. We'll ultimately want asterisc to execute until the program exits.
		require.NotContains(t, args, "--stop-at")
	})

	t.Run("BinarySnapshots", func(t *testing.T) {
		cfg.Network = "mainnet"
		cfg.BinarySnapshots = true
		_, _, args := captureExec(t, cfg, 100)
		require.Equal(t, filepath.Join(dir, SnapsDir, "%d.bin.gz"), args["--snapshot-fmt"])
	})

	t.Run("JsonSnapshots", func(t *testing.T) {
		cfg.Network = "mainnet"
		cfg.BinarySnapshots = false
		_, _, args := captureExec(t, cfg, 100)
		require.Equal(t, filepath.Join(dir, SnapsDir, "%d.json.gz"), args["--snapshot-fmt"])
	})
}

type stubVmMetrics struct {
	executionTimeRecordCount int
}

func (c *stubVmMetrics) RecordExecutionTime(_ time.Duration) {
	c.executionTimeRecordCount++
}

func (c *stubVmMetrics) RecordMemoryUsed(_ uint64) {
}
