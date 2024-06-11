package cannon

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum/go-ethereum/log"
)

type Executor struct {
	logger           log.Logger
	metrics          CannonMetricer
	l1               string
	l1Beacon         string
	l2               string
	inputs           utils.LocalGameInputs
	cannon           string
	server           string
	network          string
	rollupConfig     string
	l2Genesis        string
	absolutePreState string
	snapshotFreq     uint
	infoFreq         uint
	selectSnapshot   utils.SnapshotSelect
	cmdExecutor      utils.CmdExecutor
}

func NewExecutor(logger log.Logger, m CannonMetricer, cfg *config.Config, prestate string, inputs utils.LocalGameInputs) *Executor {
	return &Executor{
		logger:           logger,
		metrics:          m,
		l1:               cfg.L1EthRpc,
		l1Beacon:         cfg.L1Beacon,
		l2:               cfg.L2Rpc,
		inputs:           inputs,
		cannon:           cfg.CannonBin,
		server:           cfg.CannonServer,
		network:          cfg.CannonNetwork,
		rollupConfig:     cfg.CannonRollupConfigPath,
		l2Genesis:        cfg.CannonL2GenesisPath,
		absolutePreState: prestate,
		snapshotFreq:     cfg.CannonSnapshotFreq,
		infoFreq:         cfg.CannonInfoFreq,
		selectSnapshot:   utils.FindStartingSnapshot,
		cmdExecutor:      utils.RunCmd,
	}
}

// GenerateProof executes cannon to generate a proof at the specified trace index.
// The proof is stored at the specified directory.
func (e *Executor) GenerateProof(ctx context.Context, dir string, i uint64) error {
	return e.generateProof(ctx, dir, i, i)
}

// generateProof executes cannon from the specified starting trace index until the end trace index.
// The proof is stored at the specified directory.
func (e *Executor) generateProof(ctx context.Context, dir string, begin uint64, end uint64, extraCannonArgs ...string) error {
	snapshotDir := filepath.Join(dir, utils.SnapsDir)
	start, err := e.selectSnapshot(e.logger, snapshotDir, e.absolutePreState, begin)
	if err != nil {
		return fmt.Errorf("find starting snapshot: %w", err)
	}
	proofDir := filepath.Join(dir, utils.ProofsDir)
	dataDir := utils.PreimageDir(dir)
	lastGeneratedState := filepath.Join(dir, utils.FinalState)
	args := []string{
		"run",
		"--input", start,
		"--output", lastGeneratedState,
		"--meta", "",
		"--info-at", "%" + strconv.FormatUint(uint64(e.infoFreq), 10),
		"--proof-at", "=" + strconv.FormatUint(end, 10),
		"--proof-fmt", filepath.Join(proofDir, "%d.json.gz"),
		"--snapshot-at", "%" + strconv.FormatUint(uint64(e.snapshotFreq), 10),
		"--snapshot-fmt", filepath.Join(snapshotDir, "%d.json.gz"),
	}
	if end < math.MaxUint64 {
		args = append(args, "--stop-at", "="+strconv.FormatUint(end+1, 10))
	}
	args = append(args, extraCannonArgs...)
	args = append(args,
		"--",
		e.server, "--server",
		"--l1", e.l1,
		"--l1.beacon", e.l1Beacon,
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
	e.logger.Info("Generating trace", "proof", end, "cmd", e.cannon, "args", strings.Join(args, ", "))
	execStart := time.Now()
	err = e.cmdExecutor(ctx, e.logger.New("proof", end), e.cannon, args...)
	e.metrics.RecordCannonExecutionTime(time.Since(execStart).Seconds())
	return err
}
