package cannon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum/go-ethereum/log"
)

const snapsDir = "snapshots"

var snapshotNameRegexp = regexp.MustCompile(`^[0-9]+\.json$`)

const snapshotFrequency = 10_000

type snapshotSelect func(logger log.Logger, dir string, absolutePreState string, i uint64) (string, error)

type Executor struct {
	logger           log.Logger
	l1               string
	l2               string
	cannon           string
	absolutePreState string
	dataDir          string
	selectSnapshot   snapshotSelect
}

func NewExecutor(logger log.Logger, cfg *config.Config) *Executor {
	return &Executor{
		logger:           logger,
		l1:               cfg.L1EthRpc,
		l2:               cfg.CannonL2,
		cannon:           cfg.CannonBin,
		absolutePreState: cfg.CannonAbsolutePreState,
		dataDir:          cfg.CannonDatadir,
		selectSnapshot:   findStartingSnapshot,
	}
}

func (e *Executor) GenerateProof(dir string, i uint64) error {
	start, err := e.selectSnapshot(e.logger, filepath.Join(e.dataDir, snapsDir), e.absolutePreState, i)
	if err != nil {
		return fmt.Errorf("find starting snapshot: %w", err)
	}
	return fmt.Errorf("please execute cannon with --input %v --proof-at %v --proof-fmt %v/%v/%%d.json --snapshot-at %%%d --snapshot-fmt '%v/%v/%%d.json",
		start, i, dir, proofsDir, snapshotFrequency, dir, snapsDir)
}

// findStartingSnapshot finds the closest snapshot before the specified traceIndex in snapDir.
// If no suitable snapshot can be found it returns absolutePreState.
func findStartingSnapshot(logger log.Logger, snapDir string, absolutePreState string, traceIndex uint64) (string, error) {
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
			logger.Warn("Unexpected directory in snapshots dir: %v/%v", snapDir, entry.Name())
			continue
		}
		name := entry.Name()
		if !snapshotNameRegexp.MatchString(name) {
			logger.Warn("Unexpected file in snapshots dir: %v/%v", snapDir, entry.Name())
			continue
		}
		index, err := strconv.ParseUint(name[0:len(name)-len(".json")], 10, 64)
		if err != nil {
			logger.Error("Unable to parse trace index of snapshot file: %v/%v", snapDir, entry.Name())
			continue
		}
		if index > bestSnap && index < traceIndex {
			bestSnap = index
		}
	}
	if bestSnap == 0 {
		return absolutePreState, nil
	}
	startFrom := fmt.Sprintf("%v/%v.json", snapDir, bestSnap)

	return startFrom, nil
}
