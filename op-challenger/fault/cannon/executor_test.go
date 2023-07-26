package cannon

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

const execTestCannonPrestate = "/foo/pre.json"

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
