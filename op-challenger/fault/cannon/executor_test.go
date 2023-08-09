package cannon

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

const execTestCannonPrestate = "/foo/pre.json"

func TestGenerateProof(t *testing.T) {
	input := "starting.json"
	cfg := config.NewConfig("http://localhost:8888", common.Address{0xaa}, config.TraceTypeCannon, true, 5)
	cfg.CannonDatadir = t.TempDir()
	cfg.CannonAbsolutePreState = "pre.json"
	cfg.CannonBin = "./bin/cannon"
	cfg.CannonServer = "./bin/op-program"
	cfg.CannonL2 = "http://localhost:9999"
	cfg.CannonSnapshotFreq = 500

	inputs := localGameInputs{
		l1Head:        common.Hash{0x11},
		l2Head:        common.Hash{0x22},
		l2OutputRoot:  common.Hash{0x33},
		l2Claim:       common.Hash{0x44},
		l2BlockNumber: big.NewInt(3333),
	}
	captureExec := func(t *testing.T, cfg config.Config, proofAt uint64) (string, string, map[string]string) {
		executor := NewExecutor(testlog.Logger(t, log.LvlInfo), &cfg, inputs)
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
					// Skip over the divider between cannon and server program
					i += 1
					continue
				}
				args[a[i]] = a[i+1]
				i += 2
			}
			return nil
		}
		err := executor.GenerateProof(context.Background(), cfg.CannonDatadir, proofAt)
		require.NoError(t, err)
		return binary, subcommand, args
	}

	t.Run("Network", func(t *testing.T) {
		cfg.CannonNetwork = "mainnet"
		cfg.CannonRollupConfigPath = ""
		cfg.CannonL2GenesisPath = ""
		binary, subcommand, args := captureExec(t, cfg, 150_000_000)
		require.DirExists(t, filepath.Join(cfg.CannonDatadir, preimagesDir))
		require.DirExists(t, filepath.Join(cfg.CannonDatadir, proofsDir))
		require.DirExists(t, filepath.Join(cfg.CannonDatadir, snapsDir))
		require.Equal(t, cfg.CannonBin, binary)
		require.Equal(t, "run", subcommand)
		require.Equal(t, input, args["--input"])
		require.Contains(t, args, "--meta")
		require.Equal(t, "", args["--meta"])
		require.Equal(t, filepath.Join(cfg.CannonDatadir, finalState), args["--output"])
		require.Equal(t, "=150000000", args["--proof-at"])
		require.Equal(t, "=150000001", args["--stop-at"])
		require.Equal(t, "%500", args["--snapshot-at"])
		// Slight quirk of how we pair off args
		// The server binary winds up as the key and the first arg --server as the value which has no value
		// Then everything else pairs off correctly again
		require.Equal(t, "--server", args[cfg.CannonServer])
		require.Equal(t, cfg.L1EthRpc, args["--l1"])
		require.Equal(t, cfg.CannonL2, args["--l2"])
		require.Equal(t, filepath.Join(cfg.CannonDatadir, preimagesDir), args["--datadir"])
		require.Equal(t, filepath.Join(cfg.CannonDatadir, proofsDir, "%d.json"), args["--proof-fmt"])
		require.Equal(t, filepath.Join(cfg.CannonDatadir, snapsDir, "%d.json"), args["--snapshot-fmt"])
		require.Equal(t, cfg.CannonNetwork, args["--network"])
		require.NotContains(t, args, "--rollup.config")
		require.NotContains(t, args, "--l2.genesis")

		// Local game inputs
		require.Equal(t, inputs.l1Head.Hex(), args["--l1.head"])
		require.Equal(t, inputs.l2Head.Hex(), args["--l2.head"])
		require.Equal(t, inputs.l2OutputRoot.Hex(), args["--l2.outputroot"])
		require.Equal(t, inputs.l2Claim.Hex(), args["--l2.claim"])
		require.Equal(t, "3333", args["--l2.blocknumber"])
	})

	t.Run("RollupAndGenesis", func(t *testing.T) {
		cfg.CannonNetwork = ""
		cfg.CannonRollupConfigPath = "rollup.json"
		cfg.CannonL2GenesisPath = "genesis.json"
		_, _, args := captureExec(t, cfg, 150_000_000)
		require.NotContains(t, args, "--network")
		require.Equal(t, cfg.CannonRollupConfigPath, args["--rollup.config"])
		require.Equal(t, cfg.CannonL2GenesisPath, args["--l2.genesis"])
	})

	t.Run("NoStopAtWhenProofIsMaxUInt", func(t *testing.T) {
		cfg.CannonNetwork = "mainnet"
		cfg.CannonRollupConfigPath = "rollup.json"
		cfg.CannonL2GenesisPath = "genesis.json"
		_, _, args := captureExec(t, cfg, math.MaxUint64)
		// stop-at would need to be one more than the proof step which would overflow back to 0
		// so expect that it will be omitted. We'll ultimately want cannon to execute until the program exits.
		require.NotContains(t, args, "--stop-at")
	})
}

func TestRunCmdLogsOutput(t *testing.T) {
	bin := "/bin/echo"
	if _, err := os.Stat(bin); err != nil {
		t.Skip(bin, " not available", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger := testlog.Logger(t, log.LvlInfo)
	logs := testlog.Capture(logger)
	err := runCmd(ctx, logger, bin, "Hello World")
	require.NoError(t, err)
	require.NotNil(t, logs.FindLog(log.LvlInfo, "Hello World"))
}

func TestFindStartingSnapshot(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)

	withSnapshots := func(t *testing.T, files ...string) string {
		dir := t.TempDir()
		for _, file := range files {
			require.NoError(t, os.WriteFile(fmt.Sprintf("%v/%v", dir, file), nil, 0o644))
		}
		return dir
	}

	t.Run("UsePrestateWhenSnapshotsDirDoesNotExist", func(t *testing.T) {
		dir := t.TempDir()
		snapshot, err := findStartingSnapshot(logger, filepath.Join(dir, "doesNotExist"), execTestCannonPrestate, 1200)
		require.NoError(t, err)
		require.Equal(t, execTestCannonPrestate, snapshot)
	})

	t.Run("UsePrestateWhenSnapshotsDirEmpty", func(t *testing.T) {
		dir := withSnapshots(t)
		snapshot, err := findStartingSnapshot(logger, dir, execTestCannonPrestate, 1200)
		require.NoError(t, err)
		require.Equal(t, execTestCannonPrestate, snapshot)
	})

	t.Run("UsePrestateWhenNoSnapshotBeforeTraceIndex", func(t *testing.T) {
		dir := withSnapshots(t, "100.json", "200.json")
		snapshot, err := findStartingSnapshot(logger, dir, execTestCannonPrestate, 99)
		require.NoError(t, err)
		require.Equal(t, execTestCannonPrestate, snapshot)

		snapshot, err = findStartingSnapshot(logger, dir, execTestCannonPrestate, 100)
		require.NoError(t, err)
		require.Equal(t, execTestCannonPrestate, snapshot)
	})

	t.Run("UseClosestAvailableSnapshot", func(t *testing.T) {
		dir := withSnapshots(t, "100.json", "123.json", "250.json")

		snapshot, err := findStartingSnapshot(logger, dir, execTestCannonPrestate, 101)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, "100.json"), snapshot)

		snapshot, err = findStartingSnapshot(logger, dir, execTestCannonPrestate, 123)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, "100.json"), snapshot)

		snapshot, err = findStartingSnapshot(logger, dir, execTestCannonPrestate, 124)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, "123.json"), snapshot)

		snapshot, err = findStartingSnapshot(logger, dir, execTestCannonPrestate, 256)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, "250.json"), snapshot)
	})

	t.Run("IgnoreDirectories", func(t *testing.T) {
		dir := withSnapshots(t, "100.json")
		require.NoError(t, os.Mkdir(filepath.Join(dir, "120.json"), 0o777))
		snapshot, err := findStartingSnapshot(logger, dir, execTestCannonPrestate, 150)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, "100.json"), snapshot)
	})

	t.Run("IgnoreUnexpectedFiles", func(t *testing.T) {
		dir := withSnapshots(t, ".file", "100.json", "foo", "bar.json")
		snapshot, err := findStartingSnapshot(logger, dir, execTestCannonPrestate, 150)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, "100.json"), snapshot)
	})
}
