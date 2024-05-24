package asterisc

import (
	"context"
	"math"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestGenerateProof(t *testing.T) {
	input := "starting.json"
	tempDir := t.TempDir()
	dir := filepath.Join(tempDir, "gameDir")
	cfg := config.NewConfig(common.Address{0xbb}, "http://localhost:8888", "http://localhost:9000", "http://localhost:9096", "http://localhost:9095", tempDir, config.TraceTypeAsterisc)
	cfg.L2Rpc = "http://localhost:9999"
	prestate := "pre.json"
	cfg.AsteriscBin = "./bin/asterisc"
	cfg.AsteriscServer = "./bin/op-program"
	cfg.AsteriscSnapshotFreq = 500
	cfg.AsteriscInfoFreq = 900

	inputs := utils.LocalGameInputs{
		L1Head:        common.Hash{0x11},
		L2Head:        common.Hash{0x22},
		L2OutputRoot:  common.Hash{0x33},
		L2Claim:       common.Hash{0x44},
		L2BlockNumber: big.NewInt(3333),
	}
	captureExec := func(t *testing.T, cfg config.Config, proofAt uint64) (string, string, map[string]string) {
		m := &asteriscDurationMetrics{}
		executor := NewExecutor(testlog.Logger(t, log.LevelInfo), m, &cfg, prestate, inputs)
		executor.selectSnapshot = func(logger log.Logger, dir string, absolutePreState string, i uint64) (string, error) {
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
					// Skip over the divider between asterisc and server program
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
		require.Equal(t, 1, m.executionTimeRecordCount, "Should record asterisc execution time")
		return binary, subcommand, args
	}

	t.Run("Network", func(t *testing.T) {
		cfg.AsteriscNetwork = "mainnet"
		cfg.AsteriscRollupConfigPath = ""
		cfg.AsteriscL2GenesisPath = ""
		binary, subcommand, args := captureExec(t, cfg, 150_000_000)
		require.DirExists(t, filepath.Join(dir, utils.PreimagesDir))
		require.DirExists(t, filepath.Join(dir, proofsDir))
		require.DirExists(t, filepath.Join(dir, utils.SnapsDir))
		require.Equal(t, cfg.AsteriscBin, binary)
		require.Equal(t, "run", subcommand)
		require.Equal(t, input, args["--input"])
		require.Contains(t, args, "--meta")
		require.Equal(t, "", args["--meta"])
		require.Equal(t, filepath.Join(dir, utils.FinalState), args["--output"])
		require.Equal(t, "=150000000", args["--proof-at"])
		require.Equal(t, "=150000001", args["--stop-at"])
		require.Equal(t, "%500", args["--snapshot-at"])
		require.Equal(t, "%900", args["--info-at"])
		// Slight quirk of how we pair off args
		// The server binary winds up as the key and the first arg --server as the value which has no value
		// Then everything else pairs off correctly again
		require.Equal(t, "--server", args[cfg.AsteriscServer])
		require.Equal(t, cfg.L1EthRpc, args["--l1"])
		require.Equal(t, cfg.L1Beacon, args["--l1.beacon"])
		require.Equal(t, cfg.L2Rpc, args["--l2"])
		require.Equal(t, filepath.Join(dir, utils.PreimagesDir), args["--datadir"])
		require.Equal(t, filepath.Join(dir, proofsDir, "%d.json.gz"), args["--proof-fmt"])
		require.Equal(t, filepath.Join(dir, utils.SnapsDir, "%d.json.gz"), args["--snapshot-fmt"])
		require.Equal(t, cfg.AsteriscNetwork, args["--network"])
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
		cfg.AsteriscNetwork = ""
		cfg.AsteriscRollupConfigPath = "rollup.json"
		cfg.AsteriscL2GenesisPath = "genesis.json"
		_, _, args := captureExec(t, cfg, 150_000_000)
		require.NotContains(t, args, "--network")
		require.Equal(t, cfg.AsteriscRollupConfigPath, args["--rollup.config"])
		require.Equal(t, cfg.AsteriscL2GenesisPath, args["--l2.genesis"])
	})

	t.Run("NoStopAtWhenProofIsMaxUInt", func(t *testing.T) {
		cfg.AsteriscNetwork = "mainnet"
		cfg.AsteriscRollupConfigPath = "rollup.json"
		cfg.AsteriscL2GenesisPath = "genesis.json"
		_, _, args := captureExec(t, cfg, math.MaxUint64)
		// stop-at would need to be one more than the proof step which would overflow back to 0
		// so expect that it will be omitted. We'll ultimately want asterisc to execute until the program exits.
		require.NotContains(t, args, "--stop-at")
	})
}

type asteriscDurationMetrics struct {
	metrics.NoopMetricsImpl
	executionTimeRecordCount int
}

func (c *asteriscDurationMetrics) RecordAsteriscExecutionTime(_ float64) {
	c.executionTimeRecordCount++
}
