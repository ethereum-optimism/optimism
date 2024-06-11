package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

type SnapshotSelect func(logger log.Logger, dir string, absolutePreState string, i uint64) (string, error)
type CmdExecutor func(ctx context.Context, l log.Logger, binary string, args ...string) error

const (
	SnapsDir     = "snapshots"
	PreimagesDir = "preimages"
	FinalState   = "final.json.gz"
)

var snapshotNameRegexp = regexp.MustCompile(`^[0-9]+\.json.gz$`)

func PreimageDir(dir string) string {
	return filepath.Join(dir, PreimagesDir)
}

func RunCmd(ctx context.Context, l log.Logger, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	stdOut := oplog.NewWriter(l, log.LevelInfo)
	defer stdOut.Close()
	// Keep stdErr at info level because FPVM uses stderr for progress messages
	stdErr := oplog.NewWriter(l, log.LevelInfo)
	defer stdErr.Close()
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	return cmd.Run()
}

// FindStartingSnapshot finds the closest snapshot before the specified traceIndex in snapDir.
// If no suitable snapshot can be found it returns absolutePreState.
func FindStartingSnapshot(logger log.Logger, snapDir string, absolutePreState string, traceIndex uint64) (string, error) {
	// Find the closest snapshot to start from
	entries, err := os.ReadDir(snapDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return absolutePreState, nil
		}
		return "", fmt.Errorf("list snapshots in %v: %w", snapDir, err)
	}
	bestSnap := uint64(0)
	for _, entry := range entries {
		if entry.IsDir() {
			logger.Warn("Unexpected directory in snapshots dir", "parent", snapDir, "child", entry.Name())
			continue
		}
		name := entry.Name()
		if !snapshotNameRegexp.MatchString(name) {
			logger.Warn("Unexpected file in snapshots dir", "parent", snapDir, "child", entry.Name())
			continue
		}
		index, err := strconv.ParseUint(name[0:len(name)-len(".json.gz")], 10, 64)
		if err != nil {
			logger.Error("Unable to parse trace index of snapshot file", "parent", snapDir, "child", entry.Name())
			continue
		}
		if index > bestSnap && index < traceIndex {
			bestSnap = index
		}
	}
	if bestSnap == 0 {
		return absolutePreState, nil
	}
	startFrom := fmt.Sprintf("%v/%v.json.gz", snapDir, bestSnap)

	return startFrom, nil
}
