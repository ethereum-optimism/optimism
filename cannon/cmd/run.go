package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/pkg/profile"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
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

type Proof struct {
	Step uint64 `json:"step"`

	Pre  common.Hash `json:"pre"`
	Post common.Hash `json:"post"`

	StateData hexutil.Bytes `json:"state-data"`
	ProofData hexutil.Bytes `json:"proof-data"`

	OracleKey    hexutil.Bytes `json:"oracle-key,omitempty"`
	OracleValue  hexutil.Bytes `json:"oracle-value,omitempty"`
	OracleOffset uint32        `json:"oracle-offset,omitempty"`
}

type rawHint string

func (rh rawHint) Hint() string {
	return string(rh)
}

type rawKey [32]byte

func (rk rawKey) PreimageKey() [32]byte {
	return rk
}

type ProcessPreimageOracle struct {
	pCl      *preimage.OracleClient
	hCl      *preimage.HintWriter
	cmd      *exec.Cmd
	waitErr  chan error
	cancelIO context.CancelCauseFunc
}

const clientPollTimeout = time.Second * 15

func NewProcessPreimageOracle(name string, args []string, stdout log.Logger, stderr log.Logger) (*ProcessPreimageOracle, error) {
	if name == "" {
		return &ProcessPreimageOracle{}, nil
	}

	pClientRW, pOracleRW, err := preimage.CreateBidirectionalChannel()
	if err != nil {
		return nil, err
	}
	hClientRW, hOracleRW, err := preimage.CreateBidirectionalChannel()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(name, args...) // nosemgrep
	cmd.Stdout = &mipsevm.LoggingWriter{Log: stdout}
	cmd.Stderr = &mipsevm.LoggingWriter{Log: stderr}
	cmd.ExtraFiles = []*os.File{
		hOracleRW.Reader(),
		hOracleRW.Writer(),
		pOracleRW.Reader(),
		pOracleRW.Writer(),
	}

	// Note that the client file descriptors are not closed when the pre-image server exits.
	// So we use the FilePoller to ensure that we don't get stuck in a blocking read/write.
	ctx, cancelIO := context.WithCancelCause(context.Background())
	preimageClientIO := preimage.NewFilePoller(ctx, pClientRW, clientPollTimeout)
	hostClientIO := preimage.NewFilePoller(ctx, hClientRW, clientPollTimeout)
	out := &ProcessPreimageOracle{
		pCl:      preimage.NewOracleClient(preimageClientIO),
		hCl:      preimage.NewHintWriter(hostClientIO),
		cmd:      cmd,
		waitErr:  make(chan error),
		cancelIO: cancelIO,
	}
	return out, nil
}

func (p *ProcessPreimageOracle) Hint(v []byte) {
	if p.hCl == nil { // no hint processor
		return
	}
	p.hCl.Hint(rawHint(v))
}

func (p *ProcessPreimageOracle) GetPreimage(k [32]byte) []byte {
	if p.pCl == nil {
		panic("no pre-image retriever available")
	}
	return p.pCl.Get(rawKey(k))
}

func (p *ProcessPreimageOracle) Start() error {
	if p.cmd == nil {
		return nil
	}
	err := p.cmd.Start()
	go p.wait()
	return err
}

func (p *ProcessPreimageOracle) Close() error {
	if p.cmd == nil {
		return nil
	}
	// Give the pre-image server time to exit cleanly before killing it.
	time.Sleep(time.Second * 1)
	_ = p.cmd.Process.Signal(os.Interrupt)
	return <-p.waitErr
}

func (p *ProcessPreimageOracle) wait() {
	err := p.cmd.Wait()
	var waitErr error
	if err, ok := err.(*exec.ExitError); !ok || !err.Success() {
		waitErr = err
	}
	p.cancelIO(fmt.Errorf("%w: pre-image server has exited", waitErr))
	p.waitErr <- waitErr
	close(p.waitErr)
}

type StepFn func(proof bool) (*mipsevm.StepWitness, error)

func Guard(proc *os.ProcessState, fn StepFn) StepFn {
	return func(proof bool) (*mipsevm.StepWitness, error) {
		wit, err := fn(proof)
		if err != nil {
			if proc.Exited() {
				return nil, fmt.Errorf("pre-image server exited with code %d, resulting in err %w", proc.ExitCode(), err)
			} else {
				return nil, err
			}
		}
		return wit, nil
	}
}

var _ mipsevm.PreimageOracle = (*ProcessPreimageOracle)(nil)

func Run(ctx *cli.Context) error {
	if ctx.Bool(RunPProfCPU.Name) {
		defer profile.Start(profile.NoShutdownHook, profile.ProfilePath("."), profile.CPUProfile).Stop()
	}

	state, err := jsonutil.LoadJSON[mipsevm.State](ctx.Path(RunInputFlag.Name))
	if err != nil {
		return err
	}

	guestLogger := Logger(os.Stderr, log.LevelInfo)
	outLog := &mipsevm.LoggingWriter{Log: guestLogger.With("module", "guest", "stream", "stdout")}
	errLog := &mipsevm.LoggingWriter{Log: guestLogger.With("module", "guest", "stream", "stderr")}

	l := Logger(os.Stderr, log.LevelInfo).With("module", "vm")

	stopAtAnyPreimage := false
	var stopAtPreimageKeyPrefix []byte
	stopAtPreimageOffset := uint32(0)
	if ctx.IsSet(RunStopAtPreimageFlag.Name) {
		val := ctx.String(RunStopAtPreimageFlag.Name)
		parts := strings.Split(val, "@")
		if len(parts) > 2 {
			return fmt.Errorf("invalid %v: %v", RunStopAtPreimageFlag.Name, val)
		}
		stopAtPreimageKeyPrefix = common.FromHex(parts[0])
		if len(parts) == 2 {
			x, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid preimage offset: %w", err)
			}
			stopAtPreimageOffset = uint32(x)
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
	po, err := NewProcessPreimageOracle(args[0], args[1:], poOut, poErr)
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

	var meta *mipsevm.Metadata
	if metaPath := ctx.Path(RunMetaFlag.Name); metaPath == "" {
		l.Info("no metadata file specified, defaulting to empty metadata")
		meta = &mipsevm.Metadata{Symbols: nil} // provide empty metadata by default
	} else {
		if m, err := jsonutil.LoadJSON[mipsevm.Metadata](metaPath); err != nil {
			return fmt.Errorf("failed to load metadata: %w", err)
		} else {
			meta = m
		}
	}

	us := mipsevm.NewInstrumentedState(state, po, outLog, errLog)
	debugProgram := ctx.Bool(RunDebugFlag.Name)
	if debugProgram {
		if metaPath := ctx.Path(RunMetaFlag.Name); metaPath == "" {
			return fmt.Errorf("cannot enable debug mode without a metadata file")
		}
		if err := us.InitDebug(meta); err != nil {
			return fmt.Errorf("failed to initialize debug mode: %w", err)
		}
	}
	proofFmt := ctx.String(RunProofFmtFlag.Name)
	snapshotFmt := ctx.String(RunSnapshotFmtFlag.Name)

	stepFn := us.Step
	if po.cmd != nil {
		stepFn = Guard(po.cmd.ProcessState, stepFn)
	}

	start := time.Now()
	startStep := state.Step

	// avoid symbol lookups every instruction by preparing a matcher func
	sleepCheck := meta.SymbolMatcher("runtime.notesleep")

	for !state.Exited {
		if state.Step%100 == 0 { // don't do the ctx err check (includes lock) too often
			if err := ctx.Context.Err(); err != nil {
				return err
			}
		}

		step := state.Step

		if infoAt(state) {
			delta := time.Since(start)
			l.Info("processing",
				"step", step,
				"pc", mipsevm.HexU32(state.Cpu.PC),
				"insn", mipsevm.HexU32(state.Memory.GetMemory(state.Cpu.PC)),
				"ips", float64(step-startStep)/(float64(delta)/float64(time.Second)),
				"pages", state.Memory.PageCount(),
				"mem", state.Memory.Usage(),
				"name", meta.LookupSymbol(state.Cpu.PC),
			)
		}

		if sleepCheck(state.Cpu.PC) { // don't loop forever when we get stuck because of an unexpected bad program
			return fmt.Errorf("got stuck in Go sleep at step %d", step)
		}

		if stopAt(state) {
			l.Info("Reached stop at")
			break
		}

		if snapshotAt(state) {
			if err := jsonutil.WriteJSON(fmt.Sprintf(snapshotFmt, step), state, OutFilePerm); err != nil {
				return fmt.Errorf("failed to write state snapshot: %w", err)
			}
		}

		if proofAt(state) {
			preStateHash, err := state.EncodeWitness().StateHash()
			if err != nil {
				return fmt.Errorf("failed to hash prestate witness: %w", err)
			}
			witness, err := stepFn(true)
			if err != nil {
				return fmt.Errorf("failed at proof-gen step %d (PC: %08x): %w", step, state.Cpu.PC, err)
			}
			postStateHash, err := state.EncodeWitness().StateHash()
			if err != nil {
				return fmt.Errorf("failed to hash poststate witness: %w", err)
			}
			proof := &Proof{
				Step:      step,
				Pre:       preStateHash,
				Post:      postStateHash,
				StateData: witness.State,
				ProofData: witness.MemProof,
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
				return fmt.Errorf("failed at step %d (PC: %08x): %w", step, state.Cpu.PC, err)
			}
		}

		lastPreimageKey, lastPreimageValue, lastPreimageOffset := us.LastPreimage()
		if lastPreimageOffset != ^uint32(0) {
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
	l.Info("Execution stopped", "exited", state.Exited, "code", state.ExitCode)
	if debugProgram {
		us.Traceback()
	}

	if err := jsonutil.WriteJSON(ctx.Path(RunOutputFlag.Name), state, OutFilePerm); err != nil {
		return fmt.Errorf("failed to write state output: %w", err)
	}
	if debugInfoFile := ctx.Path(RunDebugInfoFlag.Name); debugInfoFile != "" {
		if err := jsonutil.WriteJSON(debugInfoFile, us.GetDebugInfo(), OutFilePerm); err != nil {
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
