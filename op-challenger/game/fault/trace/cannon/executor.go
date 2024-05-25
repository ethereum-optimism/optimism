package cannon

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

const (
	snapsDir     = "snapshots"
	preimagesDir = "preimages"
	finalState   = "final.json.gz"
)

var snapshotNameRegexp = regexp.MustCompile(`^[0-9]+\.json.gz$`)

type snapshotSelect func(logger log.Logger, dir string, absolutePreState string, i uint64) (string, error)
type cmdExecutor func(ctx context.Context, l log.Logger, binary string, args ...string) error

type Executor struct {
	logger           log.Logger
	metrics          CannonMetricer
	l1               string
	l2               string
	inputs           LocalGameInputs
	cannon           string
	server           string
	network          string
	rollupConfig     string
	l2Genesis        string
	absolutePreState string
	snapshotFreq     uint
	infoFreq         uint
	selectSnapshot   snapshotSelect
	cmdExecutor      cmdExecutor
}

func NewExecutor(logger log.Logger, m CannonMetricer, cfg *config.Config, inputs LocalGameInputs) *Executor {
	return &Executor{
		logger:           logger,
		metrics:          m,
		l1:               cfg.L1EthRpc,
		l2:               cfg.CannonL2,
		inputs:           inputs,
		cannon:           cfg.CannonBin,
		server:           cfg.CannonServer,
		network:          cfg.CannonNetwork,
		rollupConfig:     cfg.CannonRollupConfigPath,
		l2Genesis:        cfg.CannonL2GenesisPath,
		absolutePreState: cfg.CannonAbsolutePreState,
		snapshotFreq:     cfg.CannonSnapshotFreq,
		infoFreq:         cfg.CannonInfoFreq,
		selectSnapshot:   findStartingSnapshot,
		cmdExecutor:      runCmd,
	}
}

func (e *Executor) GenerateProof(ctx context.Context, dir string, i uint64) error {
	snapshotDir := filepath.Join(dir, snapsDir)
	start, err := e.selectSnapshot(e.logger, snapshotDir, e.absolutePreState, i)
	if err != nil {
		return fmt.Errorf("find starting snapshot: %w", err)
	}
	proofDir := filepath.Join(dir, proofsDir)
	dataDir := filepath.Join(dir, preimagesDir)
	lastGeneratedState := filepath.Join(dir, finalState)
	args := []string{
		"run",
		"--input", start,
		"--output", lastGeneratedState,
		"--meta", "",
		"--info-at", "%" + strconv.FormatUint(uint64(e.infoFreq), 10),
		"--proof-at", "=" + strconv.FormatUint(i, 10),
		"--proof-fmt", filepath.Join(proofDir, "%d.json.gz"),
		"--snapshot-at", "%" + strconv.FormatUint(uint64(e.snapshotFreq), 10),
		"--snapshot-fmt", filepath.Join(snapshotDir, "%d.json.gz"),
	}
	if i < math.MaxUint64 {
		args = append(args, "--stop-at", "="+strconv.FormatUint(i+1, 10))
	}
	args = append(args,
		"--",
		e.server, "--server",
		"--l1", e.l1,
		"--l2", e.l2,
		"--datadir", dataDir,
		"--l1.head", e.inputs.L1Head.Hex(),
		"--l2.head", e.inputs.L2Head.Hex(),
		"--l2.outputroot", e.inputs.L2OutputRoot.Hex(),
		"--l2.claim", e.inputs.L2Claim.Hex(),
		"--l2.blocknumber", e.inputs.L2BlockNumber.Text(10),
	)
	if e.network != "" {
		args = append(args, "--network", e.network)
	}
	if e.rollupConfig != "" {
		args = append(args, "--rollup.config", e.rollupConfig)
	}
	if e.l2Genesis != "" {
		args = append(args, "--l2.genesis", e.l2Genesis)
	}

	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return fmt.Errorf("could not create snapshot directory %v: %w", snapshotDir, err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("could not create preimage cache directory %v: %w", dataDir, err)
	}
	if err := os.MkdirAll(proofDir, 0755); err != nil {
		return fmt.Errorf("could not create proofs directory %v: %w", proofDir, err)
	}
	e.logger.Info("Generating trace", "proof", i, "cmd", e.cannon, "args", strings.Join(args, ", "))
	execStart := time.Now()
	err = e.cmdExecutor(ctx, e.logger.New("proof", i), e.cannon, args...)
	e.metrics.RecordCannonExecutionTime(time.Since(execStart).Seconds())
	return err
}

func runCmd(ctx context.Context, l log.Logger, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	stdOut := oplog.NewWriter(l, log.LvlInfo)
	defer stdOut.Close()
	// Keep stdErr at info level because cannon uses stderr for progress messages
	stdErr := oplog.NewWriter(l, log.LvlInfo)
	defer stdErr.Close()
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	return cmd.Run()
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
