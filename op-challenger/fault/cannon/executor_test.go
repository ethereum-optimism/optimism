package cannon

import (
	"context"
	"fmt"
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
	cfg := config.NewConfig("http://localhost:8888", common.Address{0xaa}, common.Address{0xbb}, config.TraceTypeCannon, true, 5)
	cfg.CannonDatadir = t.TempDir()
	cfg.CannonAbsolutePreState = "pre.json"
	cfg.CannonBin = "./bin/cannon"
	cfg.CannonServer = "./bin/op-program"
	cfg.CannonL2 = "http://localhost:9999"
	cfg.CannonSnapshotFreq = 500

	executor := NewExecutor(testlog.Logger(t, log.LvlInfo), &cfg)
	executor.selectSnapshot = func(logger log.Logger, dir string, absolutePreState string, i uint64) (string, error) {
		return input, nil
	}
	var binary string
	var subcommand string
	args := make(map[string]string)
	executor.cmdExecutor = func(ctx context.Context, l log.Logger, b string, a ...string) error {
		binary = b
		subcommand = a[0]
		for i := 1; i < len(a); i += 2 {
			args[a[i]] = a[i+1]
		}
		return nil
	}
	err := executor.GenerateProof(context.Background(), cfg.CannonDatadir, 150_000_000)
	require.NoError(t, err)
	require.Equal(t, cfg.CannonBin, binary)
	require.Equal(t, "run", subcommand)
	require.Equal(t, input, args["--input"])
	require.Equal(t, "=150000000", args["--proof-at"])
	require.Equal(t, "=150000001", args["--stop-at"])
	require.Equal(t, "%500", args["--snapshot-at"])
	require.Equal(t, cfg.CannonServer, args["--"])
	require.Equal(t, cfg.L1EthRpc, args["--l1"])
	require.Equal(t, cfg.CannonL2, args["--l2"])
	require.Equal(t, filepath.Join(cfg.CannonDatadir, preimagesDir), args["--datadir"])
	require.Equal(t, filepath.Join(cfg.CannonDatadir, proofsDir, "%d.json"), args["--proof-fmt"])
	require.Equal(t, filepath.Join(cfg.CannonDatadir, snapsDir, "%d.json"), args["--snapshot-fmt"])
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
