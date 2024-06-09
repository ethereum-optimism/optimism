package vm

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum/go-ethereum/log"
)

type Metricer interface {
	RecordVmExecutionTime(vmType string, t time.Duration)
}

type Config struct {
	VmType           string
	L1               string
	L1Beacon         string
	L2               string
	VmBin            string // Path to the vm executable to run when generating trace data
	Server           string // Path to the executable that provides the pre-image oracle server
	Network          string
	RollupConfigPath string
	L2GenesisPath    string
	SnapshotFreq     uint // Frequency of snapshots to create when executing (in VM instructions)
	InfoFreq         uint // Frequency of progress log messages (in VM instructions)
}

type Executor struct {
	cfg              Config
	logger           log.Logger
	metrics          Metricer
	absolutePreState string
	inputs           utils.LocalGameInputs
	selectSnapshot   SnapshotSelect
	cmdExecutor      CmdExecutor
}

func NewExecutor(logger log.Logger, m Metricer, cfg Config, prestate string, inputs utils.LocalGameInputs) *Executor {
	return &Executor{
		cfg:              cfg,
		logger:           logger,
		metrics:          m,
		inputs:           inputs,
		absolutePreState: prestate,
		selectSnapshot:   FindStartingSnapshot,
		cmdExecutor:      RunCmd,
	}
}

// GenerateProof executes vm to generate a proof at the specified trace index.
// The proof is stored at the specified directory.
func (e *Executor) GenerateProof(ctx context.Context, dir string, i uint64) error {
	return e.DoGenerateProof(ctx, dir, i, i)
}

// DoGenerateProof executes vm from the specified starting trace index until the end trace index.
// The proof is stored at the specified directory.
func (e *Executor) DoGenerateProof(ctx context.Context, dir string, begin uint64, end uint64, extraVmArgs ...string) error {
	snapshotDir := filepath.Join(dir, SnapsDir)
	start, err := e.selectSnapshot(e.logger, snapshotDir, e.absolutePreState, begin)
	if err != nil {
		return fmt.Errorf("find starting snapshot: %w", err)
	}
	proofDir := filepath.Join(dir, utils.ProofsDir)
	dataDir := PreimageDir(dir)
	lastGeneratedState := filepath.Join(dir, FinalState)
	args := []string{
		"run",
		"--input", start,
		"--output", lastGeneratedState,
		"--meta", "",
		"--info-at", "%" + strconv.FormatUint(uint64(e.cfg.InfoFreq), 10),
		"--proof-at", "=" + strconv.FormatUint(end, 10),
		"--proof-fmt", filepath.Join(proofDir, "%d.json.gz"),
		"--snapshot-at", "%" + strconv.FormatUint(uint64(e.cfg.SnapshotFreq), 10),
		"--snapshot-fmt", filepath.Join(snapshotDir, "%d.json.gz"),
	}
	if end < math.MaxUint64 {
		args = append(args, "--stop-at", "="+strconv.FormatUint(end+1, 10))
	}
	args = append(args, extraVmArgs...)
	args = append(args,
		"--",
		e.cfg.Server, "--server",
		"--l1", e.cfg.L1,
		"--l1.beacon", e.cfg.L1Beacon,
		"--l2", e.cfg.L2,
		"--datadir", dataDir,
		"--l1.head", e.inputs.L1Head.Hex(),
		"--l2.head", e.inputs.L2Head.Hex(),
		"--l2.outputroot", e.inputs.L2OutputRoot.Hex(),
		"--l2.claim", e.inputs.L2Claim.Hex(),
		"--l2.blocknumber", e.inputs.L2BlockNumber.Text(10),
	)
	if e.cfg.Network != "" {
		args = append(args, "--network", e.cfg.Network)
	}
	if e.cfg.RollupConfigPath != "" {
		args = append(args, "--rollup.config", e.cfg.RollupConfigPath)
	}
	if e.cfg.L2GenesisPath != "" {
		args = append(args, "--l2.genesis", e.cfg.L2GenesisPath)
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
	e.logger.Info("Generating trace", "proof", end, "cmd", e.cfg.VmBin, "args", strings.Join(args, ", "))
	execStart := time.Now()
	err = e.cmdExecutor(ctx, e.logger.New("proof", end), e.cfg.VmBin, args...)
	e.metrics.RecordVmExecutionTime(e.cfg.VmType, time.Since(execStart))
	return err
}
