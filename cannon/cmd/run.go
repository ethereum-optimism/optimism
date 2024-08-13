package cmd

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/cannon/logutil"
	program32 "github.com/ethereum-optimism/optimism/cannon/mipsevm32/program"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm64/multithreaded"
	program64 "github.com/ethereum-optimism/optimism/cannon/mipsevm64/program"
	"github.com/ethereum-optimism/optimism/cannon/run"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/profile"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm32/singlethreaded"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

var (
	RunInputFlag = &cli.PathFlag{
		Name:      "input",
		Usage:     "path of input JSON state. Stdin if left empty.",
		TakesFile: true,
		Value:     "state.json",
		Required:  true,
	}
	RunOutputFlag = &cli.PathFlag{
		Name:      "output",
		Usage:     "path of output JSON state. Not written if empty, use - to write to Stdout.",
		TakesFile: true,
		Value:     "out.json",
		Required:  false,
	}
	patternHelp    = "'never' (default), 'always', '=123' at exactly step 123, '%123' for every 123 steps"
	RunProofAtFlag = &cli.GenericFlag{
		Name:     "proof-at",
		Usage:    "step pattern to output proof at: " + patternHelp,
		Value:    new(StepMatcherFlag),
		Required: false,
	}
	RunProofFmtFlag = &cli.StringFlag{
		Name:     "proof-fmt",
		Usage:    "format for proof data output file names. Proof data is written to stdout if -.",
		Value:    "proof-%d.json",
		Required: false,
	}
	RunSnapshotAtFlag = &cli.GenericFlag{
		Name:     "snapshot-at",
		Usage:    "step pattern to output snapshots at: " + patternHelp,
		Value:    new(StepMatcherFlag),
		Required: false,
	}
	RunSnapshotFmtFlag = &cli.StringFlag{
		Name:     "snapshot-fmt",
		Usage:    "format for snapshot output file names.",
		Value:    "state-%d.json",
		Required: false,
	}
	RunStopAtFlag = &cli.GenericFlag{
		Name:     "stop-at",
		Usage:    "step pattern to stop at: " + patternHelp,
		Value:    new(StepMatcherFlag),
		Required: false,
	}
	RunStopAtPreimageFlag = &cli.StringFlag{
		Name:     "stop-at-preimage",
		Usage:    "stop at the first preimage request matching this key",
		Required: false,
	}
	RunStopAtPreimageTypeFlag = &cli.StringFlag{
		Name:     "stop-at-preimage-type",
		Usage:    "stop at the first preimage request matching this type",
		Required: false,
	}
	RunStopAtPreimageLargerThanFlag = &cli.StringFlag{
		Name:     "stop-at-preimage-larger-than",
		Usage:    "stop at the first step that requests a preimage larger than the specified size (in bytes)",
		Required: false,
	}
	RunMetaFlag = &cli.PathFlag{
		Name:     "meta",
		Usage:    "path to metadata file for symbol lookup for enhanced debugging info during execution.",
		Value:    "meta.json",
		Required: false,
	}
	RunInfoAtFlag = &cli.GenericFlag{
		Name:     "info-at",
		Usage:    "step pattern to print info at: " + patternHelp,
		Value:    MustStepMatcherFlag("%100000"),
		Required: false,
	}
	RunPProfCPU = &cli.BoolFlag{
		Name:  "pprof.cpu",
		Usage: "enable pprof cpu profiling",
	}
	RunDebugFlag = &cli.BoolFlag{
		Name:  "debug",
		Usage: "enable debug mode, which includes stack traces and other debug info in the output. Requires --meta.",
	}
	RunDebugInfoFlag = &cli.PathFlag{
		Name:      "debug-info",
		Usage:     "path to write debug info to",
		TakesFile: true,
		Required:  false,
	}

	OutFilePerm = os.FileMode(0o755)
)

type VM interface {
	CheckInfiniteLoop() bool
	Step(proof bool) (*run.StepWitness, error)
	GetStep() uint64
	GetDebugInfo() *run.DebugInfo
	Traceback()
	LastPreimage() (preimageKey [32]byte, preimage []byte, preimageOffset uint64)
	EncodeWitness() (witness []byte, hash common.Hash)
	GetExited() bool
	GetExitCode() uint8
	InfoLogVars() []interface{}
	GetPC() uint64
	WriteState(path string, perm os.FileMode) error
}

func Run(ctx *cli.Context) error {
	if ctx.Bool(RunPProfCPU.Name) {
		defer profile.Start(profile.NoShutdownHook, profile.ProfilePath("."), profile.CPUProfile).Stop()
	}

	vmType, err := vmTypeFromString(ctx)
	if err != nil {
		return err
	}

	guestLogger := Logger(os.Stderr, log.LevelInfo)
	outLog := &logutil.LoggingWriter{Log: guestLogger.With("module", "guest", "stream", "stdout")}
	errLog := &logutil.LoggingWriter{Log: guestLogger.With("module", "guest", "stream", "stderr")}

	l := Logger(os.Stderr, log.LevelInfo).With("module", "vm")

	stopAtAnyPreimage := false
	var stopAtPreimageKeyPrefix []byte
	stopAtPreimageOffset := uint64(0)
	if ctx.IsSet(RunStopAtPreimageFlag.Name) {
		val := ctx.String(RunStopAtPreimageFlag.Name)
		parts := strings.Split(val, "@")
		if len(parts) > 2 {
			return fmt.Errorf("invalid %v: %v", RunStopAtPreimageFlag.Name, val)
		}
		stopAtPreimageKeyPrefix = common.FromHex(parts[0])
		if len(parts) == 2 {
			x, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid preimage offset: %w", err)
			}
			stopAtPreimageOffset = uint64(x)
		}
	} else {
		switch ctx.String(RunStopAtPreimageTypeFlag.Name) {
		case "local":
			stopAtPreimageKeyPrefix = []byte{byte(preimage.LocalKeyType)}
		case "keccak":
			stopAtPreimageKeyPrefix = []byte{byte(preimage.Keccak256KeyType)}
		case "sha256":
			stopAtPreimageKeyPrefix = []byte{byte(preimage.Sha256KeyType)}
		case "blob":
			stopAtPreimageKeyPrefix = []byte{byte(preimage.BlobKeyType)}
		case "precompile":
			stopAtPreimageKeyPrefix = []byte{byte(preimage.PrecompileKeyType)}
		case "any":
			stopAtAnyPreimage = true
		case "":
			// 0 preimage type is forbidden so will not stop at any preimage
		default:
			return fmt.Errorf("invalid preimage type %q", ctx.String(RunStopAtPreimageTypeFlag.Name))
		}
	}
	stopAtPreimageLargerThan := ctx.Int(RunStopAtPreimageLargerThanFlag.Name)

	// split CLI args after first '--'
	args := ctx.Args().Slice()
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		args = []string{""}
	}

	poOut := Logger(os.Stdout, log.LevelInfo).With("module", "host")
	poErr := Logger(os.Stderr, log.LevelInfo).With("module", "host")
	po, err := run.NewProcessPreimageOracle(args[0], args[1:], poOut, poErr)
	if err != nil {
		return fmt.Errorf("failed to create pre-image oracle process: %w", err)
	}
	if err := po.Start(); err != nil {
		return fmt.Errorf("failed to start pre-image oracle server: %w", err)
	}
	defer func() {
		if err := po.Close(); err != nil {
			l.Error("failed to close pre-image server", "err", err)
		}
	}()

	stopAt := ctx.Generic(RunStopAtFlag.Name).(*StepMatcherFlag).Matcher()
	proofAt := ctx.Generic(RunProofAtFlag.Name).(*StepMatcherFlag).Matcher()
	snapshotAt := ctx.Generic(RunSnapshotAtFlag.Name).(*StepMatcherFlag).Matcher()
	infoAt := ctx.Generic(RunInfoAtFlag.Name).(*StepMatcherFlag).Matcher()

	var vm VM
	var debugProgram bool
	if vmType == cannonVMType {
		l.Info("Using cannon VM")
		var meta *program32.Metadata
		if metaPath := ctx.Path(RunMetaFlag.Name); metaPath == "" {
			l.Info("no metadata file specified, defaulting to empty metadata")
			meta = &program32.Metadata{Symbols: nil} // provide empty metadata by default
		} else {
			if m, err := jsonutil.LoadJSON[program32.Metadata](metaPath); err != nil {
				return fmt.Errorf("failed to load metadata: %w", err)
			} else {
				meta = m
			}
		}

		cannon, err := singlethreaded.NewInstrumentedStateFromFile(ctx.Path(RunInputFlag.Name), po, outLog, errLog, meta)
		if err != nil {
			return err
		}
		debugProgram = ctx.Bool(RunDebugFlag.Name)
		if debugProgram {
			if metaPath := ctx.Path(RunMetaFlag.Name); metaPath == "" {
				return fmt.Errorf("cannot enable debug mode without a metadata file")
			}
			if err := cannon.InitDebug(); err != nil {
				return fmt.Errorf("failed to initialize debug mode: %w", err)
			}
		}
		vm = cannon
	} else if vmType == mtVMType {
		l.Info("Using cannon multithreaded VM")
		var meta *program64.Metadata
		if metaPath := ctx.Path(RunMetaFlag.Name); metaPath == "" {
			l.Info("no metadata file specified, defaulting to empty metadata")
			meta = &program64.Metadata{Symbols: nil} // provide empty metadata by default
		} else {
			if m, err := jsonutil.LoadJSON[program64.Metadata](metaPath); err != nil {
				return fmt.Errorf("failed to load metadata: %w", err)
			} else {
				meta = m
			}
		}
		cannon, err := multithreaded.NewInstrumentedStateFromFile(ctx.Path(RunInputFlag.Name), po, outLog, errLog, l)
		if err != nil {
			return err
		}
		debugProgram = ctx.Bool(RunDebugFlag.Name)
		if debugProgram {
			if metaPath := ctx.Path(RunMetaFlag.Name); metaPath == "" {
				return fmt.Errorf("cannot enable debug mode without a metadata file")
			}
			if err := cannon.InitDebug(meta); err != nil {
				return fmt.Errorf("failed to initialize debug mode: %w", err)
			}
		}
		vm = cannon
	} else {
		return fmt.Errorf("unknown VM type %q", vmType)
	}

	proofFmt := ctx.String(RunProofFmtFlag.Name)
	snapshotFmt := ctx.String(RunSnapshotFmtFlag.Name)

	stepFn := po.GuardStep(vm.Step)

	start := time.Now()

	//state := vm.GetState()
	startStep := vm.GetStep()

	for !vm.GetExited() {
		step := vm.GetStep()
		if step%100 == 0 { // don't do the ctx err check (includes lock) too often
			if err := ctx.Context.Err(); err != nil {
				return err
			}
		}

		if infoAt(vm) {
			delta := time.Since(start)
			fields := []interface{}{
				"step", step,
				"ips", float64(step-startStep) / (float64(delta) / float64(time.Second)),
			}
			fields = append(fields, vm.InfoLogVars()...)
			l.Info("processing", fields...)
		}

		if vm.CheckInfiniteLoop() {
			// don't loop forever when we get stuck because of an unexpected bad program
			return fmt.Errorf("detected an infinite loop at step %d", step)
		}

		if stopAt(vm) {
			l.Info("Reached stop at")
			break
		}

		if snapshotAt(vm) {
			if err := vm.WriteState(fmt.Sprintf(snapshotFmt, step), OutFilePerm); err != nil {
				return fmt.Errorf("failed to write state snapshot: %w", err)
			}
		}

		if proofAt(vm) {
			witness, err := stepFn(true)
			if err != nil {
				return fmt.Errorf("failed at proof-gen step %d (PC: %08x): %w", step, vm.GetPC(), err)
			}
			_, postStateHash := vm.EncodeWitness()
			proof := &run.Proof{
				Step:      step,
				Pre:       witness.StateHash,
				Post:      postStateHash,
				StateData: witness.State,
				ProofData: witness.ProofData,
			}
			if witness.HasPreimage() {
				proof.OracleKey = witness.PreimageKey[:]
				proof.OracleValue = witness.PreimageValue
				proof.OracleOffset = witness.PreimageOffset
			}
			if err := jsonutil.WriteJSON(fmt.Sprintf(proofFmt, step), proof, OutFilePerm); err != nil {
				return fmt.Errorf("failed to write proof data: %w", err)
			}
		} else {
			_, err = stepFn(false)
			if err != nil {
				return fmt.Errorf("failed at step %d (PC: %08x): %w", step, vm.GetPC(), err)
			}
		}

		lastPreimageKey, lastPreimageValue, lastPreimageOffset := vm.LastPreimage()
		if lastPreimageOffset != ^uint64(0) {
			if stopAtAnyPreimage {
				l.Info("Stopping at preimage read")
				break
			}
			if len(stopAtPreimageKeyPrefix) > 0 &&
				slices.Equal(lastPreimageKey[:len(stopAtPreimageKeyPrefix)], stopAtPreimageKeyPrefix) {
				if stopAtPreimageOffset == lastPreimageOffset {
					l.Info("Stopping at preimage read", "keyPrefix", common.Bytes2Hex(stopAtPreimageKeyPrefix), "offset", lastPreimageOffset)
					break
				}
			}
			if stopAtPreimageLargerThan != 0 && len(lastPreimageValue) > stopAtPreimageLargerThan {
				l.Info("Stopping at preimage read", "size", len(lastPreimageValue), "min", stopAtPreimageLargerThan)
				break
			}
		}
	}
	l.Info("Execution stopped", "exited", vm.GetExited(), "code", vm.GetExitCode())
	if debugProgram {
		vm.Traceback()
	}

	if err := vm.WriteState(ctx.Path(RunOutputFlag.Name), OutFilePerm); err != nil {
		return fmt.Errorf("failed to write state output: %w", err)
	}
	if debugInfoFile := ctx.Path(RunDebugInfoFlag.Name); debugInfoFile != "" {
		if err := jsonutil.WriteJSON(debugInfoFile, vm.GetDebugInfo(), OutFilePerm); err != nil {
			return fmt.Errorf("failed to write benchmark data: %w", err)
		}
	}
	return nil
}

var RunCommand = &cli.Command{
	Name:        "run",
	Usage:       "Run VM step(s) and generate proof data to replicate onchain.",
	Description: "Run VM step(s) and generate proof data to replicate onchain. See flags to match when to output a proof, a snapshot, or to stop early.",
	Action:      Run,
	Flags: []cli.Flag{
		VMTypeFlag,
		RunInputFlag,
		RunOutputFlag,
		RunProofAtFlag,
		RunProofFmtFlag,
		RunSnapshotAtFlag,
		RunSnapshotFmtFlag,
		RunStopAtFlag,
		RunStopAtPreimageFlag,
		RunStopAtPreimageTypeFlag,
		RunStopAtPreimageLargerThanFlag,
		RunMetaFlag,
		RunInfoAtFlag,
		RunPProfCPU,
		RunDebugFlag,
		RunDebugInfoFlag,
	},
}
